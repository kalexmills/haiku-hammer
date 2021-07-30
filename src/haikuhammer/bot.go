package haikuhammer

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"log"
	"math/rand"
	"runtime/debug"
	"strings"
)

type Config struct {
	Token string
	ReactToHaiku bool
	ReactToNonHaiku bool
	DeleteNonHaiku bool
	ExplainNonHaiku bool
	PositiveReacts []string
	NegativeReacts []string

	Debug bool
}

func (c Config) String() string {
	return fmt.Sprintf("\tReactToHaiku: %t\n\tReactToNonHaiku: %t\n\tDeleteNonHaiku: %t\n\tExplainNonHaiku: %t\n",
		c.ReactToHaiku, c.ReactToNonHaiku, c.DeleteNonHaiku, c.ExplainNonHaiku)
}

// TODO: customize the emoji reaction given from a random set

type HaikuHammer struct {
	session *discordgo.Session

	config            Config

	channelCache map[string]*discordgo.Channel
	dmCache map[string]*discordgo.Channel
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
	var err error
	h.session, err = discordgo.New("Bot " + h.config.Token)
	if err != nil {
		log.Println("error creating Discord session,", err)
		return err
	}

	if h.config.Debug {
		h.session.LogLevel = discordgo.LogDebug
	}

	h.session.AddHandler(h.ReceiveNewMessage)

	h.session.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsDirectMessages
	if h.config.ReactToNonHaiku || h.config.ReactToHaiku {
		h.session.Identify.Intents |= discordgo.IntentsGuildMessageReactions | discordgo.IntentsDirectMessageReactions
	}

	err = h.session.Open()
	if err != nil {
		log.Println("error opening connection,", err)
		return err
	}
	return nil
}

func (h *HaikuHammer) Close() error {
	return h.session.Close()
}

func (h *HaikuHammer) ReceiveNewMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Recovered from panic on content, %s, panicking on: %v\n%v", strings.ReplaceAll(m.Content, "\n","\\n"), r, debug.Stack())
			panic(r)
		}
	}()
	if m.Author.Bot { // prevent SkyNet; don't talk to bots
		return
	}
	if err := IsHaiku(m.Content); err == nil {
		log.Printf("received haiku: %s\n", strings.ReplaceAll(m.Content, "\n","\\n"))
		h.HandleHaiku(s, m)
	} else {
		h.HandleNonHaiku(s, m, err)
	}
}

func (h *HaikuHammer) HandleHaiku(s *discordgo.Session, m *discordgo.MessageCreate) {
	if h.config.ReactToHaiku {
		h.react(s, m, randomString(h.config.PositiveReacts))
	}
}

func (h *HaikuHammer) HandleNonHaiku(s *discordgo.Session, m *discordgo.MessageCreate, err error) {
	if h.config.DeleteNonHaiku {
		h.Delete(s, m)
		return
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

func (h *HaikuHammer) Delete(s *discordgo.Session, m *discordgo.MessageCreate) {
	err := s.ChannelMessageDelete(m.ChannelID, m.Message.ID)
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

func (h *HaikuHammer) ExplainHaiku(s *discordgo.Session, m *discordgo.MessageCreate, explainErr error) {
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

func (h *HaikuHammer) react(s *discordgo.Session, m *discordgo.MessageCreate, reaction string) {
	err := s.MessageReactionAdd(m.ChannelID, m.Message.ID, reaction)
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

func randomString(strs []string) string {
	return strs[rand.Intn(len(strs))]
}

func quote(str string) string {
	return "> " + strings.ReplaceAll(str, "\n", "\n> ")
}