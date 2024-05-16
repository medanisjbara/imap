package emailmeow

import (
	"context"

	"github.com/emersion/go-message/mail"
)

type EmailConnectionStatus struct {
	Event mail.Part
	Err   error
}

func (cli *Client) StartReceiveLoops(ctx context.Context) (chan EmailConnectionStatus, error) {
	defer func() {
		if err := cli.idleCmd.Close(); err != nil {
			cli.Zlog.Err(err).Msg("failed to stop idling: %v")
		}
	}()

	// Start idling
	initialIdleCmd, err := cli.imapClient.Idle()
	if err != nil {
		cli.Zlog.Err(err).Msg("IDLE command failed: %v")
		return nil, err
	}

	cli.idleCmd = initialIdleCmd

	return cli.connectionStatus, nil
}
