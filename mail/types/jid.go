// Package: types.
// for structs and types.
package types

import (
	"fmt"
	"strings"
)

type MessageID = string

type JID struct {
	User       string
	Integrator uint16
	Server     string
}

func (jid JID) String() string {
	return fmt.Sprintf("%s@%s", jid.User, jid.Server)
}

func ParseJID(jid string) (JID, error) {
	parts := strings.Split(jid, "@")
	return NewJID(parts[0], parts[1]), nil
}

func NewJID(user, server string) JID {
	return JID{
		User:   user,
		Server: server,
	}
}

func (jid JID) IsEmpty() bool {
	return len(jid.Server) == 0
}
