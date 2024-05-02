package events

type MessageInfo struct {
	Sender   string
	ThreadID string

	ThreadName string
}

type ChatEvent struct {
	Info MessageInfo
	// Event
}
