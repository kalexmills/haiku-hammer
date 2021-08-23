package haikuhammer

import (
	"context"
	"errors"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/kalexmills/haiku-enforcer/src/haikuhammer/db"
	"log"
	"strconv"
	"strings"
)

// adminCommandPerms is a bitmask for the min permissions required to send admin commands. If any flag is set, the
// user can send HaikuHammer admin commands.
const adminCommandPerms = discordgo.PermissionAdministrator | discordgo.PermissionManageChannels | discordgo.PermissionManageServer

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
	if perms & adminCommandPerms == 0 {
		if h.config.Debug {
			log.Printf("could not verify admin permissions, found perms %d, expected %d", perms, adminCommandPerms)
		}
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

	// parse channel mention
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