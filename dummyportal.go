package main

import (
	"bytes"
	"fmt"
	"imap-bridge/database"
	"time"

	"github.com/emersion/go-message/mail"
)

func (br *IMAPBridge) StartDummyPortalCreation() {
	go br.createDummyPortals()
}

func (br *IMAPBridge) createDummyPortals() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for i := 0; i < 3; i++ {
		select {
		case <-ticker.C:
			br.createDummyPortal(i)
		}
	}
}

func (br *IMAPBridge) createDummyPortal(i int) {
	email := fmt.Sprintf("dummy_%d@example.com", i)

	// Correctly initialize and set mail headers
	header := mail.Header{}
	header.Set("From", email)
	header.Set("To", "recipient@example.com")
	header.Set("Subject", fmt.Sprintf("Dummy Email %d", i))

	body := bytes.NewBufferString(fmt.Sprintf("This is the body of dummy email %d", i))
	emailPart := &mail.Part{
		Header: &header,
		Body:   body,
	}

	user := br.GetUserByEmailAddress(email)

	// Create a new portal key
	key := database.NewPortalKey(email, "recipient@example.com")
	portal := br.GetPortalByThreadID(key)

	portal.handleEmailMessage(portalEmailMessage{
		message: emailPart,
		user:    user,
	})
}
