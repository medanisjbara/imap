package emailmeow

import "context"

type EmailConnectionStatus struct {
	Event string
	Err   error
}

func (c *Client) StartReceiveLoops(ctx context.Context) (chan EmailConnectionStatus, error) {
	return nil, nil
}
