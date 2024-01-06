package bot

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/gotd/td/tg"
)

type routers struct {
	phoneNumber string
	api         *tg.Client
}

func (r *routers) onMessage(ctx context.Context, e tg.Entities, update *tg.UpdateNewMessage) error {
	var jsonData map[string]interface{}
	var updateMessage *tg.Message

	switch v := update.Message.(type) {
	case *tg.Message:
		updateMessage = v
	default:
		return nil
	}

	switch v := updateMessage.PeerID.(type) {
	case *tg.PeerUser:
		var userInf map[string]interface{}

		userData, ok := e.Users[v.UserID]
		if !ok {
			userInf = map[string]interface{}{
				"type":    "user",
				"user_id": v.UserID,
			}
		} else {
			userInf = map[string]interface{}{
				"type":     "user",
				"user_id":  v.UserID,
				"username": userData.Username,
				"phone":    userData.Phone,
			}
		}

		jsonData = map[string]interface{}{
			"phone":   r.phoneNumber,
			"from":    userInf,
			"message": updateMessage.Message,
		}
	case *tg.PeerChat:
		jsonData = map[string]interface{}{
			"phone": r.phoneNumber,
			"from": map[string]interface{}{
				"type":    "chat",
				"chat_id": v.ChatID,
			},
			"message": updateMessage.Message,
		}
	default:
		return nil
	}

	marshalled, err := json.Marshal(jsonData)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(
		"POST", "http://127.0.0.1:8080/test",
		bytes.NewReader(marshalled),
	)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	return err
}
