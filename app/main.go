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

const (
	healthCheckPath = "/health"
	serverPort      = 8080
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

	// Fallback to polling when no domain provided
	if opts.telegram.Domain == "" {
		panic(tg.Poll(onMessage))
	}

	wh, err := tg.Webhook(opts.telegram.Domain, onMessage)
	if err != nil {
		panic(errors.Wrap(err, "failed to initialize webhook"))
	}
	srv := server(wh)
	panic(srv.ListenAndServe())
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
var httpClient = &http.Client{Timeout: 5 * time.Second}

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

// server creates a http server with a health check endpoint and a webhook endpoint.
func server(wh *telegram.Webhook) *http.Server {
	mux := http.NewServeMux()
	mux.Handle(wh.Path, http.HandlerFunc(wh.Handler))
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
