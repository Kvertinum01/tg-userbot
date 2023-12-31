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
		json.NewEncoder(w).Encode(&Respone{
			Result:  false,
			Message: err.Error(),
		})
		return
	}

	validate := validator.New()
	if err := validate.Struct(createChatReq); err != nil {
		json.NewEncoder(w).Encode(&Respone{
			Result:  false,
			Message: err.Error(),
		})
		return
	}

	uniqueUUID := s.sessions.RegUUID()

	s.sessions.Listener <- bot.EventListener{
		PhoneNumber: createChatReq.PhoneNumber,
		Event: bot.EventObject{
			ID:   bot.CREATE_CHANNEL_EVENT,
			UUID: uniqueUUID,
			Content: bot.CreateChannelEvent{
				ChatName: createChatReq.ChatName,
				Users:    createChatReq.Users,
				Message:  createChatReq.Message,
			},
		},
	}

	resp := <-s.sessions.ListenerResponse[uniqueUUID]

	switch v := resp.(type) {
	case error:
		json.NewEncoder(w).Encode(&Respone{
			Result:  false,
			Message: v.Error(),
		})
	case int64:
		json.NewEncoder(w).Encode(&Respone{
			Result:  true,
			Message: "chat created",
			Content: ResponseChatID{ChatID: v},
		})
	}
}

func (s *server) handleSendMessage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var sendMsgReq SendMessageReq
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&sendMsgReq); err != nil {
		json.NewEncoder(w).Encode(&Respone{
			Result:  false,
			Message: err.Error(),
		})
		return
	}

	validate := validator.New()
	if err := validate.Struct(sendMsgReq); err != nil {
		json.NewEncoder(w).Encode(&Respone{
			Result:  false,
			Message: err.Error(),
		})
		return
	}

	uniqueUUID := s.sessions.RegUUID()

	s.sessions.Listener <- bot.EventListener{
		PhoneNumber: sendMsgReq.PhoneNumber,
		Event: bot.EventObject{
			ID:   bot.SEND_MESSAGE_EVENT,
			UUID: uniqueUUID,
			Content: bot.SendMessageEvent{
				To:         sendMsgReq.To,
				Message:    sendMsgReq.Message,
				Attachment: sendMsgReq.Attachment,
			},
		},
	}

	resp := <-s.sessions.ListenerResponse[uniqueUUID]

	switch v := resp.(type) {
	case error:
		json.NewEncoder(w).Encode(&Respone{
			Result:  false,
			Message: v.Error(),
		})
	case nil:
		json.NewEncoder(w).Encode(&Respone{
			Result:  true,
			Message: "message sent",
		})
	}
}

func (s *server) handleAuth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var authReq AuthReq
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&authReq); err != nil {
		json.NewEncoder(w).Encode(&Respone{
			Result:  false,
			Message: err.Error(),
		})
		return
	}

	validate := validator.New()
	if err := validate.Struct(authReq); err != nil {
		json.NewEncoder(w).Encode(&Respone{
			Result:  false,
			Message: err.Error(),
		})
		return
	}

	if _, ok := s.sessions.ActivePhones[authReq.PhoneNumber]; ok {
		json.NewEncoder(w).Encode(&Respone{
			Result:  false,
			Message: "account is already authorized",
		})
		return
	}

	s.sessions.AuthProcess <- bot.AuthListener{
		PhoneNumber: authReq.PhoneNumber,
		Step:        1,
	}

	json.NewEncoder(w).Encode(&Respone{
		Result:  true,
		Message: "continue auth",
	})
}

func (s *server) handleCode(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var codeReq CodeReq
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&codeReq); err != nil {
		json.NewEncoder(w).Encode(&Respone{
			Result:  false,
			Message: err.Error(),
		})
		return
	}

	validate := validator.New()
	if err := validate.Struct(codeReq); err != nil {
		json.NewEncoder(w).Encode(&Respone{
			Result:  false,
			Message: err.Error(),
		})
		return
	}

	uniqueUUID := s.sessions.RegUUID()

	s.sessions.AuthProcess <- bot.AuthListener{
		UUID:        uniqueUUID,
		PhoneNumber: codeReq.PhoneNumber,
		Step:        2,
		Content:     codeReq.Code,
	}

	resp := <-s.sessions.ListenerResponse[uniqueUUID]

	switch v := resp.(type) {
	case error:
		json.NewEncoder(w).Encode(&Respone{
			Result:  false,
			Message: v.Error(),
		})
	case nil:
		if _, ok := s.sessions.ActivePhones[codeReq.PhoneNumber]; ok {
			json.NewEncoder(w).Encode(&Respone{
				Result:  true,
				Message: "account authorized",
			})
			return
		}

		json.NewEncoder(w).Encode(&Respone{
			Result:  true,
			Message: "continue auth",
		})
	}
}

func (s *server) handlePassword(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var pwdReq PasswordReq
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&pwdReq); err != nil {
		json.NewEncoder(w).Encode(&Respone{
			Result:  false,
			Message: err.Error(),
		})
		return
	}

	validate := validator.New()
	if err := validate.Struct(pwdReq); err != nil {
		json.NewEncoder(w).Encode(&Respone{
			Result:  false,
			Message: err.Error(),
		})
		return
	}

	uniqueUUID := s.sessions.RegUUID()

	s.sessions.AuthProcess <- bot.AuthListener{
		UUID:        uniqueUUID,
		PhoneNumber: pwdReq.PhoneNumber,
		Step:        3,
		Content:     pwdReq.Password,
	}

	resp := <-s.sessions.ListenerResponse[uniqueUUID]

	switch v := resp.(type) {
	case error:
		json.NewEncoder(w).Encode(&Respone{
			Result:  false,
			Message: v.Error(),
		})
	case nil:
		json.NewEncoder(w).Encode(&Respone{
			Result:  true,
			Message: "account authorized",
		})
	}
}
