package telegram

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net/http"
)

// Client is an interface for interacting with the Telegram API.
type Client interface {
	Webhook(domain string, f OnMessage) (*Webhook, error)
	Poll(f OnMessage) error
}

// Message represents a message received from Telegram.
type Message struct {
	Text      string
	replyFunc func(reply *Reply) error
}

// Reply sends a reply to the message that triggered the given message.
func (m *Message) Reply(reply *Reply) error {
	return m.replyFunc(reply)
}

type Reply struct {
	Text   string
	Styled bool
}

// OnMessage is a function that is called for each message received.
type OnMessage func(msg *Message) error

type clientImpl struct {
	bot *tgbotapi.BotAPI
}

func New(token string) (Client, error) {
	if token == "" {
		return nil, errors.New("failed to read empty token")
	}
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, errors.Wrap(err, "failed to construct bot api")
	}
	cl := &clientImpl{bot: bot}
	return cl, nil
}

type Webhook struct {
	Path    string
	Handler func(w http.ResponseWriter, r *http.Request)
}

// Webhook registers a webhook for the given domain and returns a Webhook struct containing the webhook path and handler.
func (c *clientImpl) Webhook(domain string, f OnMessage) (*Webhook, error) {
	if domain == "" {
		return nil, errors.New("failed to read empty domain")
	}
	p := securePath()
	link := fmt.Sprintf("%s%s", domain, p)
	wh, err := tgbotapi.NewWebhook(link)
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize webhook")
	}
	_, err = c.bot.Request(wh)
	if err != nil {
		return nil, errors.Wrap(err, "failed to request webhook creation")
	}
	h := c.handler(f)
	return &Webhook{
		Path:    p,
		Handler: h,
	}, nil
}

func (c *clientImpl) handler(f OnMessage) func(w http.ResponseWriter, r *http.Request) {
	writeError := func(w http.ResponseWriter, error string, status int) {
		errMsg, _ := json.Marshal(map[string]string{"error": error})
		w.WriteHeader(status)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(errMsg)
	}
	return func(w http.ResponseWriter, r *http.Request) {
		update, err := c.bot.HandleUpdate(r)
		if err != nil {
			writeError(w, err.Error(), http.StatusBadRequest)
			return
		}
		msg := c.message(update)
		err = f(msg)
		if err != nil {
			writeError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func (c *clientImpl) message(update *tgbotapi.Update) *Message {
	return &Message{
		Text: update.Message.Text,
		replyFunc: func(reply *Reply) error {
			m := tgbotapi.NewMessage(update.Message.Chat.ID, reply.Text)
			m.ReplyToMessageID = update.Message.MessageID
			if reply.Styled {
				m.ParseMode = tgbotapi.ModeMarkdown
			}
			_, err := c.bot.Send(m)
			return err
		},
	}
}

// Poll starts polling for messages and calls the given function f for each message received.
func (c *clientImpl) Poll(f OnMessage) error {
	ch := c.bot.GetUpdatesChan(tgbotapi.UpdateConfig{})
	for update := range ch {
		msg := c.message(&update)
		err := f(msg)
		if err != nil {
			log.Errorf("failed to process message: %v", err)
			err = msg.Reply(&Reply{
				Text: "Try again",
			})
			if err != nil {
				log.Errorf("failed to reply to message: %v", err)
			}
		}
	}
	return errors.New("failed to receive updates")
}

func securePath() string {
	token := secureToken(16)
	return fmt.Sprintf("/telegram-webhook/%s", token)
}

func secureToken(length int) string {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}
