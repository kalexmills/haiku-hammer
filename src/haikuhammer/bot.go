package haikuhammer

import (
	"context"
	"database/sql"
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
	ReactToHaiku bool
	ReactToNonHaiku bool
	DeleteNonHaiku bool
	ExplainNonHaiku bool
	ServeRandomHaiku bool

	BotUsername string
	PositiveReacts []string
	NegativeReacts []string

	Debug bool

	DBPath string
}

func (c Config) String() string {
	return fmt.Sprintf("\tReactToHaiku: %t\n\tReactToNonHaiku: %t\n\tDeleteNonHaiku: %t\n\tExplainNonHaiku: %t\n\tServeRandomHaiku: %t\n",
		c.ReactToHaiku, c.ReactToNonHaiku, c.DeleteNonHaiku, c.ExplainNonHaiku, c.ServeRandomHaiku)
}

type HaikuHammer struct {
	session *discordgo.Session
	db *sql.DB

	config  Config

	channelCache map[string]*discordgo.Channel
	dmCache map[string]*discordgo.Channel

	botID string
}

func NewHaikuHammer(config Config) HaikuHammer {
	log.Printf("Haiku Bot Config:\n%v", config)
	return HaikuHammer{
		config: config,
		channelCache: make(map[string]*discordgo.Channel),
		dmCache: make(map[string]*discordgo.Channel),
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

	h.session.AddHandler(h.ReceiveMessageCreate)
	h.session.AddHandler(h.ReceiveMessageEdit)

	h.session.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsDirectMessages
	if h.config.ReactToNonHaiku || h.config.ReactToHaiku {
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
	h.HandleMessage(s, m.Message)
}

func (h *HaikuHammer) ReceiveMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	h.HandleMessage(s, m.Message)
}

func (h *HaikuHammer) HandleMessage(s *discordgo.Session, m *discordgo.Message) {
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
	if r := myReaction(m); h.config.ReactToHaiku && r == nil {
		h.react(s, m, randomString(h.config.PositiveReacts))
		h.saveHaiku(m)
	}
}

func (h *HaikuHammer) HandleNonHaiku(s *discordgo.Session, m *discordgo.Message, err error) {
	if h.config.ServeRandomHaiku {
		if h.mentionsMe(m) {
			h.replyWithRandomHaiku(s, m)
			return
		}
	}

	if h.config.DeleteNonHaiku {
		h.Delete(s, m)
		return
	}

	if h.config.ReactToHaiku {
		h.removeReaction(s, m)
	}

	if h.config.ReactToNonHaiku {
		h.react(s, m, randomString(h.config.NegativeReacts))
		log.Println("reacted to non-haiku,", m.ID, strings.ReplaceAll(m.Content, "\n", "\\n"))
	}

	if isDM, err2 := h.isDM(s, m.ChannelID); err2 == nil && isDM && h.config.ExplainNonHaiku {
		h.ExplainHaiku(s, m, err)
	} else if err2 != nil {
		log.Println("could not lookup channel,", err)
	}
}

func (h *HaikuHammer) Delete(s *discordgo.Session, m *discordgo.Message) {
	err := s.ChannelMessageDelete(m.ChannelID, m.ID)
	if err != nil {
		log.Println("could not delete message from channel,", err)
		return
	}
	dmChannel, err := h.createDMChannel(s, m.Author.ID)
	if err != nil {
		log.Println("could not create user DM channel,", err)
		return
	}
	c, err := h.lookupChannel(s, m.ChannelID)
	if err != nil {
		log.Println("could not lookup message ChannelID,", err)
		return
	}
	explanation := fmt.Sprintf("I deleted the message you just sent to %s since I didn't think it was a proper Haiku:\n %s", c.Mention(), quote(m.Content))
	_, err = s.ChannelMessageSend(dmChannel.ID, explanation)
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
	dmChannel, err := h.createDMChannel(s, m.Author.ID)
	if err != nil {
		log.Println("could not create user DM channel,", err)
		return
	}
	_, err = s.ChannelMessageSendReply(dmChannel.ID, explainErr.Error(), m.MessageReference)
	if err != nil {
		log.Println("could not send message to user DM channel,", err)
		return
	}
}

func (h *HaikuHammer) isDM(s *discordgo.Session, channelID string) (bool, error) {
	c, err := h.lookupChannel(s, channelID)
	if err != nil {
		return false, err
	}
	return c.Type == discordgo.ChannelTypeDM && len(c.Recipients) == 1, nil
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

func (h *HaikuHammer) createDMChannel(s *discordgo.Session, authorID string) (*discordgo.Channel, error) {
	if c, ok := h.dmCache[authorID]; ok {
		return c, nil
	}
	c, err := s.UserChannelCreate(authorID)
	if err != nil {
		return nil, err
	}
	log.Println("retrieved new DM channel for user", authorID)
	h.channelCache[c.ID] = c
	h.dmCache[authorID] = c
	return c, nil
}

func (h *HaikuHammer) lookupChannel(s *discordgo.Session, channelID string) (*discordgo.Channel, error) {
	if c, ok := h.channelCache[channelID]; ok {
		return c, nil
	}
	c, err := s.Channel(channelID)
	if err != nil {
		return nil, err
	}
	log.Println("looked up channel", channelID)
	h.channelCache[channelID] = c
	if c.Type == discordgo.ChannelTypeDM && len(c.Recipients) == 1 {
		h.dmCache[c.Recipients[0].ID] = c
	}
	return c, nil
}

func (h *HaikuHammer) saveHaiku(m *discordgo.Message) error {
	gid, err := strconv.Atoi(m.GuildID)
	if err != nil {
		log.Println("could not parse guildID as integer,", m.GuildID)
		return err
	}
	cid, err := strconv.Atoi(m.ChannelID)
	if err != nil {
		log.Println("could not parse channelID as integer,", m.ChannelID)
		return err
	}
	mid, err := strconv.Atoi(m.ID)
	if err != nil {
		log.Println("could not parse messageID as integer,", m.ID)
		return err
	}
	// TODO: deduplicate haiku on content (i.e. find plagiarism)
	_, err = db.HaikuDAO.Upsert(context.Background(), h.db, db.Haiku{gid, cid, mid, m.Author.Mention(), m.Content})
	if err != nil {
		log.Println("could not save haiku to database,", err)
		return err
	}
	return nil
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
	_, err = s.ChannelMessageSendReply(m.ChannelID, presentHaiku(haiku), m.MessageReference)
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

func presentHaiku(h db.Haiku) string {
	return fmt.Sprintf("%s\n> - %s", quote(h.Content), h.AuthorMention)
}

func randomString(strs []string) string {
	return strs[rand.Intn(len(strs))]
}

func quote(str string) string {
	return "> " + strings.ReplaceAll(str, "\n", "\n> ")
}