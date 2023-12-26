package server

import (
	"encoding/json"
	"net/http"

	"github.com/Kvertinum01/userbot-api/internal/app/bot"
	"github.com/go-playground/validator"
)

func (s *server) handleCreateChat(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var createChatReq CreateChatReq
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&createChatReq); err != nil {
		json.NewEncoder(w).Encode(&BadRequestResp{
			Error: err.Error(),
		})
		return
	}

	validate := validator.New()
	if err := validate.Struct(createChatReq); err != nil {
		json.NewEncoder(w).Encode(&BadRequestResp{
			Error: err.Error(),
		})
		return
	}

	s.sessions.Listener <- bot.EventListener{
		PhoneNumber: createChatReq.PhoneNumber,
		Event: bot.EventObject{
			ID: bot.CREATE_CHANNEL_EVENT,
			Content: bot.CreateChannelEvent{
				ChatName: createChatReq.ChatName,
				Users:    createChatReq.Users,
			},
		},
	}

	json.NewEncoder(w).Encode(&GoodRequestResp{
		Message: "Created chat",
	})
}
