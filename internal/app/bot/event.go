package bot

const (
	CREATE_CHANNEL_EVENT = 1
)

type CreateChannelEvent struct {
	Users    []string
	ChatName string
}

type EventObject struct {
	ID      int
	Content interface{}
}

type EventListener struct {
	PhoneNumber string
	Event       EventObject
}
