package emailmeow

import (
	"context"
	"net/mail"
)

type Client struct{}

func (c *Client) SendEmail(ctx context.Context, address *mail.Address, msg *mail.Message) error {
	return nil
}

func (c *Client) IsLoggedIn() bool {
	return false
}
