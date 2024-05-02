package msgconv

import (
	"errors"
	"fmt"
	"strings"
    "net/mail"
    "context"

    "maunium.net/go/mautrix/event"

	"golang.org/x/exp/constraints"
    "mybridge/pkg/emailmeow/types"
)

var (
	ErrUnsupportedMsgType  = errors.New("unsupported msgtype")
	ErrMediaDownloadFailed = errors.New("failed to download media")
	ErrMediaDecryptFailed  = errors.New("failed to decrypt media")
	ErrMediaConvertFailed  = errors.New("failed to convert")
	ErrMediaUploadFailed   = errors.New("failed to upload media")
	ErrInvalidGeoURI       = errors.New("invalid `geo:` URI in message")
)

func maybeInt[T constraints.Integer](v T) *T {
	if v == 0 {
		return nil
	}
	return &v
}

func parseGeoURI(uri string) (lat, long string, err error) {
	if !strings.HasPrefix(uri, "geo:") {
		err = fmt.Errorf("uri doesn't have geo: prefix")
		return
	}
	// Remove geo: prefix and anything after ;
	coordinates := strings.Split(strings.TrimPrefix(uri, "geo:"), ";")[0]
	splitCoordinates := strings.Split(coordinates, ",")
	if len(splitCoordinates) != 2 {
		err = fmt.Errorf("didn't find exactly two numbers separated by a comma")
	} else {
		lat = splitCoordinates[0]
		long = splitCoordinates[1]
	}
	return
}

func (mc *MessageConverter) ToEmail(ctx context.Context, evt *event.Event, content *event.MessageEventContent, ) (*types.EmailMessage, error) {
	// Extract necessary information from the event and content
	subject := "Your email subject here"
	body := content.Body // Assuming it's plain text
	to := []*mail.Address{
		{
			Name:    "Recipient Name",
			Address: "recipient@example.com",
		},
	}
	from := &mail.Address{
		Name:    "Sender Name",
		Address: "sender@example.com",
	}

	// Construct the email message
	email := &types.EmailMessage{
		Subject: subject,
		Body:    body,
		To:      to,
		From:    from,
		// Add more fields as needed
	}

	// Optionally, you can handle attachments here

	return email, nil
}
