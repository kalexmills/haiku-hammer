package haikuhammer

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"log"
	"math/rand"
	"strings"
)

type Config struct {
	Token string
	ReactToHaiku bool
	ReactToNonHaiku bool
	DeleteNonHaiku bool
	PositiveReacts []string
	NegativeReacts []string
}

func (c Config) String() string {
	return fmt.Sprintf("\tReactToHaiku: %t\n\tReactToNonHaiku: %t\n\tDeleteNonHaiku: %t",
		c.ReactToHaiku, c.ReactToNonHaiku, c.DeleteNonHaiku)
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
	h.session.AddHandler(h.ReceiveMessage)

	h.session.Identify.Intents = discordgo.IntentsGuildMessages
	if h.config.ReactToNonHaiku || h.config.ReactToHaiku {
		h.session.Identify.Intents |= discordgo.IntentsGuildMessageReactions
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

func (h *HaikuHammer) ReceiveMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	if IsHaiku(m.Content) {
		log.Printf("received haiku: %s\n", strings.ReplaceAll(m.Content, "\n","\\n"))
		h.HandleHaiku(s, m)
	} else {
		h.HandleNonHaiku(s, m)
	}
}

func (h *HaikuHammer) HandleHaiku(s *discordgo.Session, m *discordgo.MessageCreate) {
	if h.config.ReactToHaiku {
		h.react(s, m, randomString(h.config.PositiveReacts))
	}
}

func (h *HaikuHammer) HandleNonHaiku(s *discordgo.Session, m *discordgo.MessageCreate) {
	if h.config.DeleteNonHaiku {
		h.Delete(s, m)
		return
	}

	if h.config.ReactToNonHaiku {
		h.react(s, m, randomString(h.config.NegativeReacts))
		log.Println("reacted to non-haiku,", m.ID, strings.ReplaceAll(m.Content, "\n", "\\n"))
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
	}
	c, err := h.lookupChannel(s, m.ChannelID)
	if err != nil {
		log.Println("could not lookup message ChannelID,", err)
	}
	explanation := fmt.Sprintf("I deleted the message you just sent to %s since I didn't think it was a proper Haiku:\n %s", c.Mention(), quote(m.Content))
	_, err = s.ChannelMessageSend(dmChannel.ID, explanation)
	if err != nil {
		log.Println("could not send message to user DM channel,", err)
	}
	log.Println("deleted message,", m.ID, strings.ReplaceAll(m.Content, "\n", "\\n"))
}

func (h *HaikuHammer) react(s *discordgo.Session, m *discordgo.MessageCreate, reaction string) {
	err := s.MessageReactionAdd(m.ChannelID, m.Message.ID, reaction)
	if err != nil {
		log.Println("could not add emoji reaction,", err)
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