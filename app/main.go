package main

import (
	"fmt"
	"github.com/pawel-ochrymowicz/google-maps-to-waze/pkg/maps"
	"github.com/pawel-ochrymowicz/google-maps-to-waze/pkg/telegram"
	"github.com/pawel-ochrymowicz/google-maps-to-waze/pkg/text"
	"github.com/pkg/errors"
	"net/http"
	"os"
	"time"
)

type telegramOpts struct {
	Token  string
	Domain string
}

type opts struct {
	telegram telegramOpts
}

func main() {
	opts := &opts{
		telegram: telegramOpts{
			Token:  os.Getenv("TELEGRAM_TOKEN"),
			Domain: os.Getenv("TELEGRAM_DOMAIN")}}

	tg, err := telegram.New(opts.telegram.Token)
	if err != nil {
		panic(errors.Wrap(err, "failed to initialize telegram"))
	}

	// Initialize polling api when no domain provided
	ch := make(chan error)
	if opts.telegram.Domain == "" {
		go func() {
			if err := tg.Poll(onMessage); err != nil {
				ch <- err
			}
		}()
	}
	var serverOpts []serverOpt
	if opts.telegram.Domain != "" {
		var wh *telegram.Webhook
		wh, err = tg.Webhook(opts.telegram.Domain, onMessage)
		if err != nil {
			panic(errors.Wrap(err, "failed to initialize webhook"))
		}
		serverOpts = append(serverOpts, withTelegramWebhook(wh))
	}
	go func() {
		srv := server(serverOpts...)
		if err := srv.ListenAndServe(); err != nil {
			ch <- err
		}
	}()
	panic(<-ch)
}

const (
	// welcomeMessage is a message that is sent when a user starts the bot.
	welcomeMessage = `
Welcome to Google Maps to Waze bot!
Send me a Google Maps link and I will send you a Waze link.

Examples:
- Shortened: https://goo.gl/maps/1JZ8Zq4J1Z8Zq4
- Full: https://www.google.com/maps/dir/?api=1&destination=51.107885,17.038538
- Any text with a link: foo bar https://www.google.com/maps/dir/?api=1&destination=51.107885,17.038538
`
)

// httpClient is a http client used to make requests to Google Maps
var httpClient = &http.Client{Timeout: 15 * time.Second}

// onMessage is a callback function that is called when a message is received.
func onMessage(message *telegram.Message) error {
	if message.Text == "/start" {
		return message.Reply(&telegram.Reply{
			Text:   welcomeMessage,
			Styled: true,
		})
	}

	u, err := text.ParseFirstUrl(message.Text)
	if err != nil {
		return errors.Wrap(err, "failed to parse url from message")
	}
	var wazeLink *maps.WazeLink
	wazeLink, err = maps.GoogleMapsUrlToWazeLink(u, maps.HttpGetToInput(httpClient))
	if err != nil {
		return errors.Wrap(err, "failed to map google maps url to waze link")
	}
	return message.Reply(&telegram.Reply{
		Text: wazeLink.URL().String(),
	})
}

// serverOpt is a function that modifies a http.ServeMux.
type serverOpt func(*http.ServeMux)

// withTelegramWebhook is a serverOpt that adds a webhook endpoint to the server.
func withTelegramWebhook(wh *telegram.Webhook) serverOpt {
	return func(mux *http.ServeMux) {
		mux.Handle(wh.Path, http.HandlerFunc(wh.Handler))
	}
}

const (
	healthCheckPath = "/health"
	serverPort      = 8080
)

// server creates a http server with a health check endpoint and a webhook endpoint.
func server(opts ...serverOpt) *http.Server {
	mux := http.NewServeMux()
	for _, o := range opts {
		o(mux)
	}
	mux.Handle(healthCheckPath, http.HandlerFunc(healthCheck))
	return &http.Server{
		Addr:              fmt.Sprintf("%s:%d", "", serverPort),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      5 * time.Second,
		IdleTimeout:       30 * time.Second,
	}
}

// healthCheck is a handler for health check endpoint.
func healthCheck(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(200)
}
