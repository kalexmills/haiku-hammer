package haikuhammer

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/kalexmills/haiku-enforcer/src/haikuhammer/db"
	"log"
	"math/rand"
	"runtime/debug"
	"strconv"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

type Config struct {
	Token string
	ActionFlags db.ConfigFlag // globally enabled/disabled actions

	BotUsername string
	PositiveReacts []string
	NegativeReacts []string

	Debug bool

	DBPath string
}

func (c Config) String() string {
	return fmt.Sprintf("\tReactToHaiku: %t\n\tReactToNonHaiku: %t\n\tDeleteNonHaiku: %t\n\tExplainNonHaiku: %t\n\tServeRandomHaiku: %t\n",
		c.ActionFlags.ReactToHaiku(), c.ActionFlags.ReactToNonHaiku(), c.ActionFlags.DeleteNonHaiku(), c.ActionFlags.ExplainNonHaiku(), c.ActionFlags.ServeRandomHaiku())
}

type HaikuHammer struct {
	session *discordgo.Session
	db *sql.DB

	config  Config

	dmCache map[string]bool // maps from channelIDs to whether they're DM channels or not
	dmChannelCache map[string]string // maps from userIDs to their DM channel ID

	botID string
}

func NewHaikuHammer(config Config) HaikuHammer {
	log.Printf("Haiku Bot Config:\n%v", config)
	return HaikuHammer{
		config: config,
		dmCache: make(map[string]bool),
		dmChannelCache: make(map[string]string),
	}
}

func (h *HaikuHammer) Open() error {
	err := h.OpenDB()
	if err != nil {
		return err
	}

	h.session, err = discordgo.New("Bot " + h.config.Token)
	if err != nil {
		log.Println("error creating Discord session,", err)
		return err
	}

	if h.config.Debug {
		h.session.LogLevel = discordgo.LogDebug
	}
	h.session.StateEnabled = true

	h.session.AddHandler(h.ReceiveMessageCreate)
	h.session.AddHandler(h.ReceiveMessageEdit)

	h.session.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsDirectMessages
	if h.config.ActionFlags.ReactToNonHaiku() || h.config.ActionFlags.ReactToHaiku() {
		h.session.Identify.Intents |= discordgo.IntentsGuildMessageReactions | discordgo.IntentsDirectMessageReactions
	}

	err = h.session.Open()
	if err != nil {
		log.Println("error opening connection,", err)
		return err
	}

	user, err := h.session.User("@me")
	if err != nil {
		log.Println("error looking up bot user", err)
		return err
	}
	h.botID = user.ID
	log.Println("Bot running as username: ", user.Username + "#" + user.Discriminator)

	return nil
}

func (h *HaikuHammer) OpenDB() error {
	DB, err := sql.Open("sqlite3", h.config.DBPath)
	if err != nil {
		return fmt.Errorf("cannot open database from %s: %w", h.config.DBPath, err)
	}

	err = db.BootstrapDB(DB)
	if err != nil {
		return fmt.Errorf("could not bootstrap database: %w", err)
	}

	h.db = DB
	return nil
}

func (h *HaikuHammer) Close() error {
	return h.session.Close()
}

func (h *HaikuHammer) ReceiveMessageEdit(s *discordgo.Session, m *discordgo.MessageUpdate) {
	h.HandleMessage(h.session, m.Message)
}

func (h *HaikuHammer) ReceiveMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if strings.HasPrefix(m.Content, "!haiku ") {
		h.HandleAdminCommand(h.session, m.Message)
		return
	}
	h.HandleMessage(h.session, m.Message)
}

func (h *HaikuHammer) HandleMessage(s *discordgo.Session, m *discordgo.Message) { // TODO: remove s from everywhere.
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic on content, %s, panicking on: %v\n%v", strings.ReplaceAll(m.Content, "\n","\\n"), r, debug.Stack())
			panic(r)
		}
	}()
	if m == nil || m.Author == nil || m.Author.Bot { // prevent dumb APIs and bot messages
		return
	}

	gid := m.GuildID // store original guild ID
	m, err := s.ChannelMessage(m.ChannelID, m.ID)
	if err != nil {
		log.Println("could not look up message from channel", err)
		return
	}
	m.GuildID = gid

	if err := IsHaiku(m.Content); err == nil {
		log.Printf("received haiku: %s\n", strings.ReplaceAll(m.Content, "\n","\\n"))
		h.HandleHaiku(s, m)
	} else {
		h.HandleNonHaiku(s, m, err)
	}
}

func (h *HaikuHammer) HandleHaiku(s *discordgo.Session, m *discordgo.Message) {
	if r := myReaction(m); h.actionsEnabled(m, db.ConfigReactToHaiku) && r == nil {
		h.react(s, m, randomString(h.config.PositiveReacts))
	}
	h.saveHaiku(m)
}

func (h *HaikuHammer) HandleNonHaiku(s *discordgo.Session, m *discordgo.Message, err error) {
	if h.actionsEnabled(m, db.ConfigServeRandomHaiku) {
		if h.mentionsMe(m) {
			h.replyWithRandomHaiku(s, m)
			return
		}
	}

	if h.actionsEnabled(m, db.ConfigDeleteNonHaiku) {
		h.Delete(s, m)
		return
	}

	if h.actionsEnabled(m, db.ConfigReactToHaiku) {
		h.removeReaction(s, m)
	}

	if h.actionsEnabled(m, db.ConfigReactToNonHaiku) {
		h.react(s, m, randomString(h.config.NegativeReacts))
		log.Println("reacted to non-haiku,", m.ID, strings.ReplaceAll(m.Content, "\n", "\\n"))
	}

	if isDM, err2 := h.isDM(s, m.ChannelID); err2 == nil &&
		((isDM && h.config.ActionFlags.ExplainNonHaiku()) || // explain non-haiku in all DMs if globally configured
			h.actionsEnabled(m, db.ConfigExplainNonHaiku)) { // also explain non-haiku in any specially-enabled channels
		h.ExplainHaiku(s, m, err)
	} else if err2 != nil {
		log.Println("could not lookup channel,", err)
	}
}

var AdminHelp = `All commands must be sent in the guild they are meant to apply to.
  ~~~!haiku feature on [target] [feature feature...]~~~
  ~~~!haiku feature off [target] [feature feature...]~~~
  ~~~!haiku feature list [target]~~~
  
~~~[target]~~~ can be either a channel mention or ~~~global~~~ to enable features for every channel in the guild.
~~~[feature feature...]~~~ is a space-separated list of features from the below list.

   - ~~~ReactToHaiku~~~ - adds an emoji reaction to any detected haiku
   - ~~~ReactToNonHaiku~~~ - adds an emoji reaction to any detected non-haiku
   - ~~~DeleteNonHaiku~~~ - deletes any messages which are not valid haiku -- requires MANAGE_MESSAGES permission
   - ~~~ExplainNonHaiku~~~ - respond publicly in channel with an explanation of why a message is not a haiku
   - ~~~ServeRandomHaiku~~~ - reacts to mentions by publicly quoting some haiku previously detected in the same guild.
`

func init() {
	AdminHelp = strings.ReplaceAll(AdminHelp, "~~~", "`")
}

func (h *HaikuHammer) HandleAdminCommand(s *discordgo.Session, m *discordgo.Message) {
	gid := m.GuildID // store original guild ID
	m, err := s.ChannelMessage(m.ChannelID, m.ID)
	if err != nil {
		log.Println("could not look up message from channel", err)
		return
	}
	m.GuildID = gid

	perms, err := h.Permissions(s,m)
	if err != nil {
		log.Println("could not retrieve permissions for user, ignoring admin command,", err)
		return
	}
	if perms & discordgo.PermissionAdministrator == 0 && perms & discordgo.PermissionManageChannels == 0 {
		h.DM(s, m, fmt.Sprintf("You do not have permissions to manage HaikuHammer in <#%s>", m.ChannelID))
		return
	}
	commandRaw := strings.TrimPrefix(m.Content, "!haiku ")
	command, err := parseCommand(commandRaw)
	if err != nil {
		s.ChannelMessageSendReply(m.ChannelID, err.Error(), m.MessageReference)
		return
	}

	switch command.Operation {
	case OpFeatureOn:
		h.updateFeatures(m, command, EnableFeatures)
		s.ChannelMessageSendReply(m.ChannelID, fmt.Sprintf("Enabled features %s for target %s", command.Features.String(), command.MentionTarget()), m.MessageReference)
	case OpFeatureOff:
		h.updateFeatures(m, command, DisableFeatures)
		s.ChannelMessageSendReply(m.ChannelID, fmt.Sprintf("Disabled features %s for target %s", command.Features.String(), command.MentionTarget()), m.MessageReference)
	case OpFeatureList:
		h.handleFeatureList(s, m, command)
	case OpHelp:
		s.ChannelMessageSendReply(m.ChannelID, AdminHelp, m.MessageReference)
	}
}

func (h *HaikuHammer) Permissions(s *discordgo.Session, m *discordgo.Message) (int64, error) {
	g, err := s.Guild(m.GuildID)
	if err != nil {
		return 0, err
	}
	if g.OwnerID == m.Author.ID {
		return discordgo.PermissionAll, nil
	}
	member, err := s.GuildMember(m.GuildID, m.Author.ID)
	if err != nil {
		return 0, err
	}
	roles, err := s.GuildRoles(m.GuildID)
	if err != nil {
		return 0, err
	}
	roleMap := make(map[string]int64)
	for _, role := range roles {
		roleMap[role.Name] = role.Permissions
	}
	permissions := roleMap["@everyone"]
	for _, role := range member.Roles {
		permissions |= roleMap[role]
	}
	if permissions & discordgo.PermissionAdministrator == discordgo.PermissionAdministrator {
		return discordgo.PermissionAll, nil
	}
	return permissions, nil
}

func (h *HaikuHammer) handleFeatureList(s *discordgo.Session, m *discordgo.Message, command Command) {
	ctx := context.Background()
	switch command.Target {
	case "global":
		gid, err := strconv.Atoi(m.GuildID)
		if err != nil {
			log.Println("could not parse guildID as integer,", m.GuildID)
			return
		}
		currConfig, err := db.GuildConfigDAO.FindByID(ctx, h.db, gid)
		if err != nil {
			log.Println("could not read guild config from database,", err)
			return
		}
		s.ChannelMessageSendReply(m.ChannelID, fmt.Sprintf("Features enabled for target %s: %s", command.MentionTarget(), currConfig.Flags), m.MessageReference)
	default:
		cid, err := strconv.Atoi(command.Target)
		if err != nil {
			log.Println("could not parse channelID as integer,", m.ChannelID)
			return
		}
		currConfig, err := db.ChannelConfigDAO.FindByID(ctx, h.db, cid)
		if err != nil {
			log.Println("could not read channel config from database,", err)
			return
		}
		s.ChannelMessageSendReply(m.ChannelID, fmt.Sprintf("Features enabled for target %s: %s", command.MentionTarget(), currConfig.Flags), m.MessageReference)
	}
}

type featureMutator func(db.ConfigFlag, db.ConfigFlag) db.ConfigFlag

func EnableFeatures(current db.ConfigFlag, feats db.ConfigFlag) db.ConfigFlag {
	return current.Or(feats)
}

func DisableFeatures(current db.ConfigFlag, feats db.ConfigFlag) db.ConfigFlag {
	return current.And(^feats) // and with bitwise not
}

func (h *HaikuHammer) updateFeatures(m *discordgo.Message, command Command, mutator featureMutator) {
	ctx := context.Background()
	switch command.Target {
	case "global":
		gid, err := strconv.Atoi(m.GuildID)
		if err != nil {
			log.Println("could not parse guildID as integer,", m.GuildID)
			return
		}
		currConfig, err := db.GuildConfigDAO.FindByID(ctx, h.db, gid) // read
		if err != nil {
			log.Println("could not retrieve guild permissions,", err)
		}

		// modify
		currConfig.GuildID = gid
		currConfig.Flags = mutator(currConfig.Flags, command.Features)

		_, err = db.GuildConfigDAO.Upsert(ctx, h.db, currConfig) // write
		if err != nil {
			log.Println("could not update guild permissions,", err)
		}
	default: // channel ID (target was verified by caller)
		cid, err := strconv.Atoi(command.Target)
		if err != nil {
			log.Println("could not parse guildID as integer,", m.GuildID)
			return
		}
		currConfig, err := db.ChannelConfigDAO.FindByID(ctx, h.db, cid) // read
		if err != nil {
			log.Println("could not retrieve channel permissions,", err)
		}

		currConfig.Flags = mutator(currConfig.Flags, command.Features)

		_, err = db.ChannelConfigDAO.Upsert(ctx, h.db, cid, currConfig.Flags) // write
		if err != nil {
			log.Println("could not update guild permissions,", err)
		}
	}
}

type Operation uint8

const (
	OpFeatureOn Operation = iota
	OpFeatureOff
	OpFeatureList
	OpHelp
)

type Command struct {
	Operation Operation
	Target string
	Features db.ConfigFlag
}

func (c Command) MentionTarget() string {
	if c.Target == "global" {
		return "global"
	}
	return fmt.Sprintf("<#%s>", c.Target)
}

func parseCommand(content string) (Command, error) {
	var err error
	tokens := strings.Split(content, " ")
	var trimmed []string
	for _, token := range tokens {
		if token != "" {
			trimmed = append(trimmed, token)
		}
	}
	if len(tokens) < 1 {
		return Command{}, errors.New("expected a valid command after `!haiku`; send `!haiku help` for help")
	}
	command := tokens[0]
	if len(tokens) > 1 {
		command += " " + tokens[1]
	}
	result := Command{}
	switch command {
	case "feature on":
		result.Operation = OpFeatureOn
		if len(tokens) < 4 {
			return Command{}, errors.New("expected a target and list of features after `feature on`; send `!haiku help` for help")
		}
	case "feature off":
		result.Operation = OpFeatureOff
		if len(tokens) < 4 {
			return Command{}, errors.New("expected a target and list of features after `feature off`; send `!haiku help` for help")
		}
	case "feature list":
		result.Operation = OpFeatureList
		if len(tokens) < 3 {
			return Command{}, errors.New("expected a target after `feature list`; send `!haiku help` for help")
		}
	case "help":
		result.Operation = OpHelp
		return result, nil
	default:
		return Command{}, fmt.Errorf("could not understand command %s", command)
	}

	result.Target = tokens[2]
	if result.Target != "global" && strings.HasPrefix(result.Target, "<#") {
		id, err := strconv.Atoi(strings.TrimSuffix(result.Target[2:], ">"))
		if err != nil {
			return Command{}, fmt.Errorf("couldn't parse target '%s' as valid channel mention", result.Target)
		}
		result.Target = fmt.Sprintf("%d", id)
	} else if result.Target != "global" {
		return Command{}, fmt.Errorf("couldn't parse target '%s' as valid target", result.Target)
	}

	result.Features, err = parseFeatures(tokens[3:])
	if err != nil {
		return Command{}, err
	}
	return result, nil
}

func parseFeatures(features []string) (db.ConfigFlag, error) {
	var result db.ConfigFlag
	for _, feature := range features {
		switch feature {
		case "ReactToHaiku":
			result |= db.ConfigReactToHaiku
		case "ReactToNonHaiku":
			result |= db.ConfigReactToNonHaiku
		case "DeleteNonHaiku":
			result |= db.ConfigDeleteNonHaiku
		case "ExplainNonHaiku":
			result |= db.ConfigExplainNonHaiku
		case "ServeRandomHaiku":
			result |= db.ConfigServeRandomHaiku
		case "": // ignore
		default:
			return 0, fmt.Errorf("could not understand '%s' as a valid feature; send `!haiku help` for help", feature)
		}
	}
	return result, nil
}

func (h *HaikuHammer) Delete(s *discordgo.Session, m *discordgo.Message) {
	err := s.ChannelMessageDelete(m.ChannelID, m.ID)
	if err != nil {
		log.Println("could not delete message from channel,", err)
		return
	}
	dmChannelID, err := h.getDMChannelID(s, m.Author.ID)
	if err != nil {
		log.Println("could not create user DM channel,", err)
		return
	}
	explanation := fmt.Sprintf("I deleted the message you just sent to %s since I didn't think it was a proper Haiku:\n %s", channelMention(m.ChannelID), quote(m.Content))
	_, err = s.ChannelMessageSend(dmChannelID, explanation)
	if err != nil {
		log.Println("could not send message to user DM channel,", err)
		return
	}
	log.Println("deleted message,", m.ID, strings.ReplaceAll(m.Content, "\n", "\\n"))
}

func (h *HaikuHammer) ExplainHaiku(s *discordgo.Session, m *discordgo.Message, explainErr error) {
	if explainErr == nil {
		log.Println("tried to explain a non-haiku without an error,", strings.ReplaceAll(m.Content, "\n", "\\n"))
		return
	}
	dmChannelID, err := h.getDMChannelID(s, m.Author.ID)
	if err != nil {
		log.Println("could not create user DM channel,", err)
		return
	}
	_, err = s.ChannelMessageSendReply(dmChannelID, explainErr.Error(), m.MessageReference)
	if err != nil {
		log.Println("could not send message to user DM channel,", err)
		return
	}
}

func (h *HaikuHammer) DM(s *discordgo.Session, m *discordgo.Message, response string) {
	dmChannelID, err := h.getDMChannelID(s, m.Author.ID)
	if err != nil {
		log.Println("could not create user DM channel,", err)
		return
	}
	_, err = s.ChannelMessageSend(dmChannelID, response)
	if err != nil {
		log.Println("could not send message to user DM channel,", err)
		return
	}
}

func (h *HaikuHammer) isDM(s *discordgo.Session, channelID string) (bool, error) {
	if result, ok := h.dmCache[channelID]; ok {
		return result, nil
	}
	c, err := s.Channel(channelID)
	if err != nil {
		return false, err
	}
	log.Println("looked up channel", channelID)
	result := c.Type == discordgo.ChannelTypeDM && len(c.Recipients) == 1
	h.dmCache[channelID] = result
	return result, nil
}

func myReaction(m *discordgo.Message) *discordgo.MessageReactions {
	for _, reaction := range m.Reactions {
		if reaction.Me {
			return reaction
		}
	}
	return nil
}

func (h *HaikuHammer) removeReaction(s *discordgo.Session, m *discordgo.Message) {
	r := myReaction(m)
	if r == nil {
		return
	}
	err := s.MessageReactionRemove(m.ChannelID, m.ID, r.Emoji.Name, h.botID)
	if err != nil {
		log.Println("could not remove emoji reaction", err)
		return
	}
}

func (h *HaikuHammer) react(s *discordgo.Session, m *discordgo.Message, reaction string) {
	err := s.MessageReactionAdd(m.ChannelID, m.ID, reaction)
	if err != nil {
		log.Println("could not add emoji reaction,", err)
		return
	}
}

func (h *HaikuHammer) getDMChannelID(s *discordgo.Session, authorID string) (string, error) {
	if c, ok := h.dmChannelCache[authorID]; ok {
		return c, nil
	}
	c, err := s.UserChannelCreate(authorID)
	if err != nil {
		return "", err
	}
	log.Println("retrieved new DM channel for user", authorID)
	h.dmCache[c.ID] = true
	h.dmChannelCache[authorID] = c.ID
	return c.ID, nil
}

func (h *HaikuHammer) getMemberNick(s *discordgo.Session, guildID string, userID string) (string, error) {
	member, err := s.GuildMember(guildID, userID)
	if err != nil {
		return "", err
	}
	return memberNick(member), nil
}

func memberNick(m *discordgo.Member) string {
	if m.Nick != "" {
		return m.Nick
	}
	return m.User.Username
}

func (h *HaikuHammer) saveHaiku(m *discordgo.Message) {
	gid, err := strconv.Atoi(m.GuildID)
	if err != nil {
		log.Println("could not parse guildID as integer,", m.GuildID)
		return
	}
	cid, err := strconv.Atoi(m.ChannelID)
	if err != nil {
		log.Println("could not parse channelID as integer,", m.ChannelID)
		return
	}
	mid, err := strconv.Atoi(m.ID)
	if err != nil {
		log.Println("could not parse messageID as integer,", m.ID)
		return
	}
	// TODO: deduplicate haiku on content (i.e. find plagiarism)
	_, err = db.HaikuDAO.Upsert(context.Background(), h.db, db.Haiku{gid, cid, mid, m.Author.ID, m.Content})
	if err != nil {
		log.Println("could not save haiku to database,", err)
	}
}

func (h *HaikuHammer) replyWithRandomHaiku(s *discordgo.Session, m *discordgo.Message) {
	haiku, err := db.HaikuDAO.Random(context.Background(), h.db, m.GuildID)
	if err != nil {
		log.Println("could not retrieve random haiku for guild", err)
		return
	}
	if haiku.Content == "" {
		log.Println("could not find any haiku for guild", m.GuildID)
		return
	}
	_, err = s.ChannelMessageSendReply(m.ChannelID, h.presentHaiku(s, haiku), m.MessageReference)
	if err != nil {
		log.Println("could not send message reply", err)
		return
	}
}

func (h *HaikuHammer) mentionsMe(m *discordgo.Message) bool {
	for _, m := range m.Mentions {
		if m.ID == h.botID {
			return true
		}
	}
	return false
}

func (h *HaikuHammer) presentHaiku(s *discordgo.Session, haiku db.Haiku) string {
	nick, err := h.getMemberNick(s, strconv.Itoa(haiku.GuildID), haiku.AuthorID)
	if err != nil {
		log.Println("could not retrieve member nick for guildID:", haiku.GuildID, "authorID:", haiku.AuthorID)
		return fmt.Sprintf("%s\n> - Unknown", quote(haiku.Content))
	}
	return fmt.Sprintf("%s\n> - %s", quote(haiku.Content), nick)
}

func (h *HaikuHammer) actionsEnabled(m *discordgo.Message, flags db.ConfigFlag) bool {
	guildID, err := strconv.Atoi(m.GuildID)
	if err != nil {
		log.Println("could not parse guildID as integer,", m.GuildID)
		return false
	}
	channelID, err := strconv.Atoi(m.ChannelID)
	if err != nil {
		log.Println("could not parse channelID as integer,", m.GuildID)
		return false
	}
	found, err := db.LookupFlags(context.Background(), h.db, guildID, channelID)
	if err != nil {
		log.Println("could not retrieve flags for guildID:", guildID, "channelID:", channelID)
	}
	return (h.config.ActionFlags & flags & found) == flags
}

func randomString(strs []string) string {
	return strs[rand.Intn(len(strs))]
}

func quote(str string) string {
	return "> " + strings.ReplaceAll(str, "\n", "\n> ")
}

func channelMention(channelID string) string {
	return fmt.Sprintf("<#%s>", channelID)
}