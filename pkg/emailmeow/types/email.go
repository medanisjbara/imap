package types

import (
	"net/mail"
)

type EmailMessage struct {
	Subject string
	Body    string
	To      []*mail.Address
	From    *mail.Address
}

func (e *EmailMessage) GetTimestamp() uint64 {
	return 0
}
