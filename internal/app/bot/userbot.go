package bot

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/gotd/contrib/middleware/floodwait"
	"github.com/gotd/contrib/middleware/ratelimit"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/telegram/updates"
	"github.com/gotd/td/tg"
	"github.com/pkg/errors"
	"golang.org/x/time/rate"
)

type Sessions struct {
	appID   int
	appHash string

	Listener chan EventListener
}

func sessionFolder(phone string) string {
	var out []rune
	for _, r := range phone {
		if r >= '0' && r <= '9' {
			out = append(out, r)
		}
	}
	return "phone-" + string(out)
}

func Setup(config *Config) *Sessions {
	s := &Sessions{
		appID:   config.AppID,
		appHash: config.AppHash,

		Listener: make(chan EventListener),
	}

	for _, currPhone := range config.Phones {
		endAuth := make(chan bool)

		go func(phone string) {
			err := s.runPhone(phone, endAuth)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %+v\n", err)
			}
			os.Exit(1)
		}(currPhone)

		<-endAuth
	}

	return s
}

func (s *Sessions) runPhone(phoneNumber string, endAuth chan bool) error {
	sessionDir := filepath.Join("sessions", sessionFolder(phoneNumber))
	if err := os.MkdirAll(sessionDir, 0700); err != nil {
		return err
	}

	fmt.Printf("Storing session in %s\n", sessionDir)

	sessionStorage := &telegram.FileSessionStorage{
		Path: filepath.Join(sessionDir, "session.json"),
	}

	dispatcher := tg.NewUpdateDispatcher()

	gaps := updates.New(updates.Config{
		Handler: dispatcher,
	})

	waiter := floodwait.NewWaiter().WithCallback(func(ctx context.Context, wait floodwait.FloodWait) {
		fmt.Println("Got FLOOD_WAIT. Will retry after", wait.Duration)
	})

	options := telegram.Options{
		SessionStorage: sessionStorage,
		UpdateHandler:  gaps,
		Middlewares: []telegram.Middleware{
			waiter,
			ratelimit.New(rate.Every(time.Millisecond*100), 5),
		},
		Device: telegram.DeviceConfig{
			DeviceModel:    "Acer Swift 14",
			SystemVersion:  "Windows 10",
			SystemLangCode: "ru",
			LangCode:       "ru",
			AppVersion:     "1.5.3",
		},
	}
	client := telegram.NewClient(s.appID, s.appHash, options)
	api := client.API()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	flow := auth.NewFlow(Terminal{PhoneNumber: phoneNumber}, auth.SendCodeOptions{})

	return waiter.Run(ctx, func(ctx context.Context) error {
		if err := client.Run(ctx, func(ctx context.Context) error {
			if err := client.Auth().IfNecessary(ctx, flow); err != nil {
				return errors.Wrap(err, "auth")
			}
			endAuth <- true

			self, err := client.Self(ctx)
			if err != nil {
				return errors.Wrap(err, "call self")
			}

			name := self.FirstName
			if self.Username != "" {
				name = fmt.Sprintf("%s (@%s)", name, self.Username)
			}

			go s.eventListener(ctx, phoneNumber, api)

			return gaps.Run(ctx, api, self.ID, updates.AuthOptions{
				IsBot: false,
				OnStart: func(ctx context.Context) {
					fmt.Println(name, "started")
				},
			})
		}); err != nil {
			return errors.Wrap(err, "run")
		}
		return nil
	})
}
