package bot

const (
	CREATE_CHANNEL_EVENT = 1
	AUTH_PHONE_EVENT     = 2
	SEND_MESSAGE_EVENT   = 3
)

type CreateChannelEvent struct {
	Users    []string
	ChatName string
	Message  string
}

type SendMessageEvent struct {
	To         string
	Message    string
	Attachment string
}

type AuthPhoneEvent struct {
	Phone string
}

type EventObject struct {
	ID      int
	UUID    string
	Content interface{}
}

type EventListener struct {
	PhoneNumber string
	Event       EventObject
}

type AuthListener struct {
	UUID        string
	PhoneNumber string
	Step        int
	Content     interface{}
}

type AuthInfo struct {
	UUID    string
	Content string
}
