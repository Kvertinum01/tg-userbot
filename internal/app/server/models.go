package server

type CreateChatReq struct {
	PhoneNumber string   `json:"phone" validate:"required"`
	ChatName    string   `json:"chat_name" validate:"required"`
	Users       []string `json:"users" validate:"required"`
}

type BadRequestResp struct {
	Error string `json:"error"`
}

type GoodRequestResp struct {
	Message string `json:"message"`
}
