package emailmeow

import (
	"context"

	"github.com/MakMoinee/go-mith/pkg/email"
	"github.com/rs/zerolog"
)

type Client struct {
	emailAddress string
	password     string
	log          zerolog.Logger

	emailService email.EmailIntf
	EventHandler func(any)
}

func NewClient(email string, password string, log zerolog.Logger) *Client {
	return &Client{
		emailAddress: email,
		password:     password,
		log:          log,
	}
}

func (c *Client) SendEmail(ctx context.Context, reciever string, msg string) error {
	isSent, err := c.emailService.SendEmail(reciever, "Forwarder From Matrix", msg)
	if err != nil {
		c.log.Err(err).Msg("Couldn't send email")
		return err
	}

	if isSent {
		c.log.Debug().Msg("Email Sent")
	} else {
		c.log.Debug().Msg("Email Not Sent")
	}
	return nil
}

func (c *Client) Login(ctx context.Context, address string, password string) error {
	// Check https://github.com/MakMoinee/go-mith/commit/9f22c396ea1adbf24a8721fa29cafea2cea1954f
	c.emailService = email.NewEmailService(587, "smtp.gmail.com", address, password)
	return nil
}

func (c *Client) IsLoggedIn() bool {
	return c.emailService != nil
}

func (c *Client) GetCurrentUser() (string, error) {
	return c.emailAddress, nil
}

func (cli *Client) handleEvent(evt any) {
	if cli.EventHandler != nil {
		cli.EventHandler(evt)
	}
}
