package msgconv

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"maunium.net/go/mautrix/event"

	"golang.org/x/exp/constraints"
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

func (mc *MessageConverter) ToEmail(ctx context.Context, evt *event.Event, content *event.MessageEventContent) (string, error) {
	// FIXME
	return content.Body, nil
}
