package main

import (
	"context"
	"net/mail"
	"regexp"

	"github.com/rs/zerolog"
	"maunium.net/go/mautrix/appservice"
	"maunium.net/go/mautrix/id"
	"mybridge/database"
)

type Puppet struct {
	*database.Puppet

	bridge *MyBridge
	log    zerolog.Logger

	MXID id.UserID

	customIntent *appservice.IntentAPI
	customUser   *User
}

func (puppet *Puppet) DefaultIntent() *appservice.IntentAPI {
	return puppet.bridge.AS.Intent(puppet.MXID)
}

// Bridge functions
func (br *MyBridge) GetPuppetByMXID(mxid id.UserID) *Puppet {
	emailAddr, ok := br.ParsePuppetMXID(mxid)
	if !ok {
		return nil
	}

	return br.GetPuppetByEmailAddress(emailAddr)
}

func (br *MyBridge) GetPuppetByEmailAddress(addr mail.Address) *Puppet {
	// FIXME
	if addr.Address == "" {
		br.ZLog.Warn().Msg("Trying to get puppet with empty email_address")
		return nil
	}

	br.puppetsLock.Lock()
	defer br.puppetsLock.Unlock()

	puppet, ok := br.puppets[addr]
	if !ok {
		dbPuppet, err := br.DB.Puppet.GetByEmailAddress(context.TODO(), addr)
		if err != nil {
			br.ZLog.Err(err).Msg("Failed to get puppet from database")
			return nil
		}
		return br.loadPuppet(context.TODO(), dbPuppet, &addr)
	}
	return puppet
}

var userIDRegex *regexp.Regexp

func ParseFromRFC5322(addrr string) string {
	// FIXME
	return "Barry Gibbs <bg@example.com>"
}

func (br *MyBridge) ParsePuppetMXID(mxid id.UserID) (mail.Address, bool) {
	if userIDRegex == nil {
		pattern := fmt.Sprintf(
			"^@%s:%s$",
			br.Config.Bridge.FormatUsername(`(\d+)`),
			br.Config.Homeserver.Domain,
		)
		userIDRegex = regexp.MustCompile(pattern)
	}

	match := userIDRegex.FindStringSubmatch(string(mxid))
	if len(match) == 2 {
		parsedRFC5322, err := ParseFromRFC5322(match[1])
		if err != nil {
			return 0, false
		}

		parsed, err := mail.Address.ParseAddress(parsedRFC5322)
		if err != nil {
			return 0, false
		}
		return parsed, true
	}

	return 0, false
}
