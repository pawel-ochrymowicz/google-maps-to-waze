package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/pawel-ochrymowicz/google-maps-to-waze/pkg/maps"
	"github.com/pawel-ochrymowicz/google-maps-to-waze/pkg/telegram"
	"github.com/pawel-ochrymowicz/google-maps-to-waze/pkg/text"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type telegramOpts struct {
	Token       string
	WebhookLink *url.URL
}

type opts struct {
	telegram           telegramOpts
	disableHealthCheck bool
}

const (
	healthCheckPath = "/health"
	serverPort      = 8080
)

func envOpts() *opts {
	webhookLinkRaw := os.Getenv("TELEGRAM_WEBHOOK_LINK")
	var webhookLink *url.URL
	if webhookLinkRaw != "" {
		var err error
		webhookLink, err = url.Parse(webhookLinkRaw)
		if err != nil {
			panic(errors.Wrap(err, "failed to parse webhook link"))
		}
	}

	return &opts{
		telegram: telegramOpts{
			Token:       os.Getenv("TELEGRAM_TOKEN"),
			WebhookLink: webhookLink,
		},
		disableHealthCheck: os.Getenv("DISABLE_HEALTH_CHECK") == "true",
	}
}

func main() {
	opts := envOpts()
	tg, err := telegram.New(opts.telegram.Token)
	if err != nil {
		panic(errors.Wrap(err, "failed to initialize telegram"))
	}

	// Initialize polling api when no webhook link provided
	ch := make(chan error)
	if opts.telegram.WebhookLink == nil {
		go func() {
			// Close possible webhook
			if err := tg.CloseWebhook(); err != nil {
				ch <- err
			}

			if err := tg.Poll(onMessage); err != nil {
				ch <- err
			}
		}()
	}
	var serverOpts []serverOpt
	if opts.telegram.WebhookLink != nil {
		var wh *telegram.Webhook
		wh, err = tg.Webhook(opts.telegram.WebhookLink, onMessage)
		if err != nil {
			panic(errors.Wrap(err, "failed to initialize webhook"))
		}
		serverOpts = append(serverOpts, withTelegramWebhook(opts.telegram.WebhookLink.Path, wh))
	}

	if !opts.disableHealthCheck {
		serverOpts = append(serverOpts, withHealthCheck())
	}

	if len(serverOpts) > 0 {
		go func() {
			log.Infof("Starting server on port %d", serverPort)
			srv := server(serverOpts...)
			if err := srv.ListenAndServe(); err != nil {
				ch <- err
			}
		}()
	}
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
var httpClient = &http.Client{Timeout: 15 * time.Second, Jar: nil}

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
	var googleMapsLink *maps.GoogleMapsLink
	googleMapsLink, err = maps.ParseGoogleMapsFromURL(u, maps.HttpGetToInput(httpClient))
	if err != nil {
		return errors.Wrapf(err, "failed to parse google maps link: %s", u)
	}
	var wazeLink *maps.WazeLink
	wazeLink, err = maps.WazeFromLocation(googleMapsLink)
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
func withTelegramWebhook(path string, wh *telegram.Webhook) serverOpt {
	return func(mux *http.ServeMux) {
		mux.Handle(path, wh.Handler)
	}
}

// withHealthCheck is a serverOpt that adds a health check endpoint to the server.
func withHealthCheck() serverOpt {
	return func(mux *http.ServeMux) {
		mux.Handle(healthCheckPath, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(200)
		}))
	}
}

// server creates a http server with a health check endpoint and a webhook endpoint.
func server(opts ...serverOpt) *http.Server {
	mux := http.NewServeMux()
	for _, o := range opts {
		o(mux)
	}
	return &http.Server{
		Addr:              fmt.Sprintf("%s:%d", "", serverPort),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      5 * time.Second,
		IdleTimeout:       30 * time.Second,
	}
}
