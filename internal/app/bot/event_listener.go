package bot

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/gotd/contrib/middleware/floodwait"
	"github.com/gotd/contrib/middleware/ratelimit"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/telegram/updates"
	"github.com/gotd/td/tg"
	"github.com/pkg/errors"
	"golang.org/x/time/rate"
)

var (
	NO_AUTH_ERROR   = errors.New("number is not authorized")
	UNK_VALUE_ERROR = errors.New("unknown value received")
)

func (s *Sessions) RegUUID() string {
	uuid := uuid.New().String()
	s.ListenerResponse[uuid] = make(chan interface{})
	return uuid
}

func (s *Sessions) eventListener() {
	for newEvent := range s.Listener {
		phoneData, ok := s.ActivePhones[newEvent.PhoneNumber]

		if !ok {
			s.ListenerResponse[newEvent.Event.UUID] <- NO_AUTH_ERROR
			continue
		}

		switch newEvent.Event.ID {
		case CREATE_CHANNEL_EVENT:
			go s.createChannel(phoneData.Ctx, phoneData.Api, newEvent)
		case SEND_MESSAGE_EVENT:
			go s.sendMessage(phoneData.Ctx, phoneData.Api, newEvent)
		}
	}
}

func (s *Sessions) sendMessage(ctx context.Context, api *tg.Client, newEvent EventListener) {
	sendMessageObj := newEvent.Event.Content.(SendMessageEvent)

	var resPeer tg.InputPeerClass

	resPeer = &tg.InputPeerChat{}

	intValue, err := strconv.ParseInt(sendMessageObj.To, 10, 64)
	if err == nil {
		resPeer = &tg.InputPeerChat{ChatID: intValue}
	} else {
		userData, err := findUser(ctx, api, sendMessageObj.To)
		if err != nil {
			s.ListenerResponse[newEvent.Event.UUID] <- err
			return
		}
		resPeer = &tg.InputPeerUser{
			UserID:     userData.UserID,
			AccessHash: userData.AccessHash,
		}
	}

	rand.Seed(time.Now().UnixNano())

	if _, err := api.MessagesSendMessage(ctx, &tg.MessagesSendMessageRequest{
		Peer:     resPeer,
		RandomID: rand.Int63(),
		Message:  sendMessageObj.Message,
	}); err != nil {
		s.ListenerResponse[newEvent.Event.UUID] <- err
		return
	}

	s.ListenerResponse[newEvent.Event.UUID] <- nil
}

func (s *Sessions) authListener() {
	for newEvent := range s.AuthProcess {
		phone := newEvent.PhoneNumber
		switch newEvent.Step {
		case 1:
			s.AuthData[phone] = make(chan AuthInfo)
			go s.authNewPhone(phone)
		case 2:
			code := newEvent.Content.(string)
			s.AuthData[phone] <- AuthInfo{Content: code, UUID: newEvent.UUID}
		case 3:
			pwd := newEvent.Content.(string)
			s.AuthData[phone] <- AuthInfo{Content: pwd, UUID: newEvent.UUID}
		}
	}
}

func (s *Sessions) authNewPhone(phoneNumber string) {
	sessionDir := filepath.Join("sessions", phoneNumber)
	if err := os.MkdirAll(sessionDir, 0700); err != nil {
		return
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

	if err := waiter.Run(ctx, func(ctx context.Context) error {
		if err := client.Run(ctx, func(ctx context.Context) error {
			authObj, err := client.Auth().SendCode(ctx, phoneNumber, auth.SendCodeOptions{})
			if err != nil {
				return errors.Wrap(err, "auth")
			}
			authInfo := authObj.(*tg.AuthSentCode)

			authCode := <-s.AuthData[phoneNumber]

			if _, err := client.Auth().SignIn(
				ctx, phoneNumber, authCode.Content, authInfo.PhoneCodeHash,
			); err != nil {
				if errors.Is(err, auth.ErrPasswordAuthNeeded) {
					s.ListenerResponse[authCode.UUID] <- nil

					authPwd := <-s.AuthData[phoneNumber]
					if _, err := client.Auth().Password(ctx, authPwd.Content); err != nil {
						os.RemoveAll(sessionDir)
						s.ListenerResponse[authPwd.UUID] <- err
						return errors.Wrap(err, "sign in with password")
					}

					s.ListenerResponse[authPwd.UUID] <- nil
				} else {
					os.RemoveAll(sessionDir)
					s.ListenerResponse[authCode.UUID] <- err
					return errors.Wrap(err, "sign in with code")
				}
			}

			s.ActivePhones[phoneNumber] = PhoneData{Ctx: ctx, Api: api}
			s.ListenerResponse[authCode.UUID] <- nil

			self, err := client.Self(ctx)
			if err != nil {
				return errors.Wrap(err, "call self")
			}

			name := self.FirstName
			if self.Username != "" {
				name = fmt.Sprintf("%s (@%s)", name, self.Username)
			}

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
	}); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %+v\n", err)
	}
}

func (s *Sessions) createChannel(ctx context.Context, api *tg.Client, newEvent EventListener) {
	createChannelObj := newEvent.Event.Content.(CreateChannelEvent)

	resUsers := []tg.InputUserClass{}

	for _, userValue := range createChannelObj.Users {
		userInput, err := findUser(ctx, api, userValue)

		if err != nil {
			s.ListenerResponse[newEvent.Event.UUID] <- err
			return
		}

		resUsers = append(resUsers, userInput)
	}

	chatUpd, err := api.MessagesCreateChat(ctx, &tg.MessagesCreateChatRequest{
		Title: createChannelObj.ChatName,
		Users: resUsers,
	})

	if err != nil {
		s.ListenerResponse[newEvent.Event.UUID] <- err
		return
	}

	updatesClass := chatUpd.(*tg.Updates).Updates

	for _, currUpd := range updatesClass {
		switch v := currUpd.(type) {
		case *tg.UpdateChatParticipants:
			rand.Seed(time.Now().UnixNano())

			chatID := v.Participants.(*tg.ChatParticipants).ChatID
			if _, err := api.MessagesSendMessage(ctx, &tg.MessagesSendMessageRequest{
				Peer:     &tg.InputPeerChat{ChatID: chatID},
				Message:  createChannelObj.Message,
				RandomID: rand.Int63(),
			}); err != nil {
				s.ListenerResponse[newEvent.Event.UUID] <- err
			}

			s.ListenerResponse[newEvent.Event.UUID] <- chatID
			return
		default:
			continue
		}
	}
}

func findUser(ctx context.Context, api *tg.Client, name_or_phone string) (*tg.InputUser, error) {
	first_letter := string(name_or_phone[0])

	switch first_letter {
	case "@":
		userName := name_or_phone[1:]
		contactResp, err := api.ContactsResolveUsername(ctx, userName)

		if err != nil {
			return nil, err
		}

		userInfo := contactResp.Users[0].(*tg.User)

		return &tg.InputUser{
			UserID:     userInfo.ID,
			AccessHash: userInfo.AccessHash,
		}, nil
	case "+":
		phoneNumber := name_or_phone

		contactResp, err := api.ContactsResolvePhone(ctx, phoneNumber)

		if err != nil {
			return nil, err
		}

		userInfo := contactResp.Users[0].(*tg.User)

		return &tg.InputUser{
			UserID:     userInfo.ID,
			AccessHash: userInfo.AccessHash,
		}, nil
	default:
		return nil, UNK_VALUE_ERROR
	}
}
