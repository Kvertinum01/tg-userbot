package bot

import (
	"context"
	"log"

	"github.com/gotd/td/tg"
)

func (s *Sessions) eventListener(ctx context.Context, phone string, api *tg.Client) {
	for newEvent := range s.Listener {
		log.Println(newEvent.PhoneNumber, phone)
		if newEvent.PhoneNumber != phone {
			continue
		}

		switch newEvent.Event.ID {
		case CREATE_CHANNEL_EVENT:
			createChannelObj := newEvent.Event.Content.(CreateChannelEvent)

			resUsers := []tg.InputUserClass{}

			for _, userName := range createChannelObj.Users {
				contactResp, err := api.ContactsResolveUsername(ctx, userName)

				if err != nil {
					log.Println(err)
					return
				}

				userInfo := contactResp.Users[0].(*tg.User)

				resUsers = append(resUsers, &tg.InputUser{
					UserID:     userInfo.ID,
					AccessHash: userInfo.AccessHash,
				})
			}

			if _, err := api.MessagesCreateChat(ctx, &tg.MessagesCreateChatRequest{
				Title: createChannelObj.ChatName,
				Users: resUsers,
			}); err != nil {
				log.Println(err)
				return
			}
		}
	}
}
