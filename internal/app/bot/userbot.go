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
	"github.com/gotd/td/telegram/updates"
	"github.com/gotd/td/tg"
	"github.com/pkg/errors"
	"golang.org/x/time/rate"
)

func DirPathWalkDir(root string) ([]string, error) {
	var dirs []string

	files, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}

	for _, fileInfo := range files {
		if fileInfo.IsDir() {
			dirs = append(dirs, fileInfo.Name())
		}
	}

	return dirs, nil
}

type PhoneData struct {
	Ctx context.Context
	Api *tg.Client
}

type Sessions struct {
	appID   int
	appHash string

	ActivePhones map[string]PhoneData

	Listener         chan EventListener
	ListenerResponse map[string](chan interface{})

	AuthProcess chan AuthListener
	AuthData    map[string](chan AuthInfo)
}

func Setup(config *Config) *Sessions {
	s := &Sessions{
		appID:   config.AppID,
		appHash: config.AppHash,

		ActivePhones: make(map[string]PhoneData),

		Listener:         make(chan EventListener),
		ListenerResponse: make(map[string]chan interface{}),

		AuthProcess: make(chan AuthListener),
		AuthData:    make(map[string]chan AuthInfo),
	}

	s.runSavedPhones()
	go s.authListener()
	go s.eventListener()

	return s
}

func (s *Sessions) runSavedPhones() error {
	dirPhones, err := DirPathWalkDir("sessions")
	if err != nil {
		return err
	}

	for _, currPhone := range dirPhones {
		go func(phone string) {
			err := s.runPhone(phone)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %+v\n", err)
			}
		}(currPhone)
	}

	return nil
}

func (s *Sessions) runPhone(phoneNumber string) error {
	sessionDir := filepath.Join("sessions", phoneNumber)
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
			DeviceModel:    "Linux User",
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

	router := routers{phoneNumber: phoneNumber, api: api}

	dispatcher.OnNewMessage(router.onMessage)

	return waiter.Run(ctx, func(ctx context.Context) error {
		if err := client.Run(ctx, func(ctx context.Context) error {
			authStatus, err := client.Auth().Status(ctx)
			if err != nil {
				return errors.Wrap(err, "auth")
			}

			if !authStatus.Authorized {
				os.RemoveAll(sessionDir)
				return errors.New(fmt.Sprintf("%s not authorized", phoneNumber))
			}

			self := authStatus.User
			if err != nil {
				return errors.Wrap(err, "call self")
			}

			name := self.FirstName
			if self.Username != "" {
				name = fmt.Sprintf("%s (@%s)", name, self.Username)
			}

			s.ActivePhones[phoneNumber] = PhoneData{Api: api, Ctx: ctx}

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
