package server

type CreateChatReq struct {
	PhoneNumber string   `json:"phone" validate:"required"`
	ChatName    string   `json:"chat_name" validate:"required"`
	Users       []string `json:"users" validate:"required"`
	Message     string   `json:"message" validate:"required"`
}

type SendMessageReq struct {
	PhoneNumber string `json:"phone" validate:"required"`
	To          string `json:"to" validate:"required"`
	Message     string `json:"message" validate:"required"`
	Attachment  string `json:"attachment"`
}

type AuthReq struct {
	PhoneNumber string `json:"phone" validate:"required"`
}

type CodeReq struct {
	PhoneNumber string `json:"phone" validate:"required"`
	Code        string `json:"code" validate:"required"`
}

type PasswordReq struct {
	PhoneNumber string `json:"phone" validate:"required"`
	Password    string `json:"password" validate:"required"`
}

type ResponseChatID struct {
	ChatID int64 `json:"chat_id"`
}

type Respone struct {
	Result  bool        `json:"result"`
	Message string      `json:"message"`
	Content interface{} `json:"content"`
}
