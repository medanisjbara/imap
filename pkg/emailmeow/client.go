package emailmeow

import (
	"context"
	"fmt"
	"io"

	"github.com/MakMoinee/go-mith/pkg/email"
	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-message/mail"
	"github.com/rs/zerolog"
)

type Client struct {
	emailAddress string
	password     string
	Zlog         zerolog.Logger

	EventHandler func(*imapclient.UnilateralDataMailbox)

	emailService email.EmailIntf

	imapClient   *imapclient.Client
	IMAPServer   string
	selectedMbox *imap.SelectData
	idleCmd      *imapclient.IdleCommand
	imapOptions  imapclient.Options

	connectionStatus chan (EmailConnectionStatus)
}

func NewClient(address string, password string) *Client {
	emailService := email.NewEmailService(587, "smtp.gmail.com", address, password)
	return &Client{
		emailAddress: address,
		password:     password,
		emailService: emailService,
	}
}

func (c *Client) SendEmail(ctx context.Context, reciever string, msg string) error {
	isSent, err := c.emailService.SendEmail(reciever, "Forwarder From Matrix", msg)
	if err != nil {
		c.Zlog.Err(err).Msg("Couldn't send email")
		return err
	}

	if isSent {
		c.Zlog.Debug().Msg("Email Sent")
	} else {
		c.Zlog.Debug().Msg("Email Not Sent")
	}
	return nil
}

func (cli *Client) Login(ctx context.Context, address string, password string) error {
	// Check https://github.com/MakMoinee/go-mith/commit/9f22c396ea1adbf24a8721fa29cafea2cea1954f
	cli.emailService = email.NewEmailService(587, "smtp.gmail.com", address, password)

	cli.imapOptions = imapclient.Options{
		UnilateralDataHandler: &imapclient.UnilateralDataHandler{
			Expunge: func(seqNum uint32) {
				cli.Zlog.Printf("message %v has been expunged", seqNum)
			},
			Mailbox: cli.EventHandler,
		},
	}

	imapcli, err := imapclient.DialTLS(cli.IMAPServer, &cli.imapOptions)
	if err != nil {
		cli.Zlog.Err(err).Msg("failed to dial IMAP server: %v")
		return err
	}

	cli.imapClient = imapcli

	if err := cli.imapClient.Login(cli.emailAddress, cli.password).Wait(); err != nil {
		cli.Zlog.Err(err).Msg("failed to login: %v")
		return err
	}

	mboxIndex, err := cli.imapClient.Select("INBOX", nil).Wait()
	if err != nil {
		cli.Zlog.Err(err).Msg("failed to select INBOX: %v")
		return err
	}

	cli.selectedMbox = mboxIndex

	return nil
}

func (c *Client) IsLoggedIn() bool {
	return c.emailService != nil
}

func (c *Client) GetCurrentUser() (string, error) {
	return c.emailAddress, nil
}

func (cli *Client) handleEvent(evt *imapclient.UnilateralDataMailbox) {
	if cli.EventHandler != nil {
		cli.EventHandler(evt)
	}
}

func (cli *Client) FetchLastMessagePart() (mail.Part, error) {
	seqSet := imap.SeqSetNum(cli.selectedMbox.NumMessages)
	fetchOptions := &imap.FetchOptions{
		BodySection: []*imap.FetchItemBodySection{{}},
	}
	fetchCmd := cli.imapClient.Fetch(seqSet, fetchOptions)
	defer fetchCmd.Close()

	msg := fetchCmd.Next()
	if msg == nil {
		return mail.Part{}, fmt.Errorf("FETCH command did not return any message")
	}

	var bodySection imapclient.FetchItemDataBodySection
	for item := msg.Next(); item != nil; item = msg.Next() {
		if bs, ok := item.(imapclient.FetchItemDataBodySection); ok {
			bodySection = bs
			break
		}
	}

	mr, err := mail.CreateReader(bodySection.Literal)
	if err != nil {
		return mail.Part{}, fmt.Errorf("failed to create mail reader: %v", err)
	}

	part, err := mr.NextPart()
	if err != nil {
		if err == io.EOF {
			return mail.Part{}, fmt.Errorf("no parts found in the message")
		}
		return mail.Part{}, fmt.Errorf("failed to read message part: %v", err)
	}

	return *part, nil
}
