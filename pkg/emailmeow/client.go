package emailmeow

import (
	"context"
	"net/mail"

	"github.com/rs/zerolog"
)

type Client struct {
	EventHandler func(any)
}

func NewClient(string, string, zerolog.Logger) *Client {
	return &Client{}
}

func (c *Client) SendEmail(ctx context.Context, address *mail.Address, msg *mail.Message) error {
	return nil
}

func (c *Client) Login(ctx context.Context, address string, password string) error {
	return nil
}

func (c *Client) IsLoggedIn() bool {
	return false
}

func (c *Client) GetCurrentUser() (string, error) {
	return "", nil
}

func (cli *Client) handleEvent(evt any) {
	if cli.EventHandler != nil {
		cli.EventHandler(evt)
	}
}
