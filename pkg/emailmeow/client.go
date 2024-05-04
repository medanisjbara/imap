package emailmeow

import (
	"context"
	"net/mail"
)

type Client struct{}

func (c *Client) SendEmail(ctx context.Context, address mail.Address, msg string) error {
	return nil
}
