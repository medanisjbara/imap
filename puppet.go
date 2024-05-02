package main

import (
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

func (br *MyBridge) GetPuppetByEmailAddress(address string) *Puppet {
	if id == "" {
		br.ZLog.Warn().Msg("Trying to get puppet with empty email_address")
		return nil
	}

	br.puppetsLock.Lock()
	defer br.puppetsLock.Unlock()

	puppet, ok := br.puppets[id]
	if !ok {
		dbPuppet, err := br.DB.Puppet.GetByEmailAddress(context.TODO(), id)
		if err != nil {
			br.ZLog.Err(err).Msg("Failed to get puppet from database")
			return nil
		}
		return br.loadPuppet(context.TODO(), dbPuppet, &id)
	}
	return puppet
}
