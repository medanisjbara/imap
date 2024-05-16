package main

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/emersion/go-message/mail"

	"mybridge/database"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/appservice"
	"maunium.net/go/mautrix/id"
)

type Puppet struct {
	*database.Puppet

	bridge *MyBridge
	log    zerolog.Logger

	MXID id.UserID

	customIntent *appservice.IntentAPI
	customUser   *User
}

// CustomIntent implements bridge.Ghost.
func (*Puppet) CustomIntent() *appservice.IntentAPI {
	panic("unimplemented")
}

func (puppet *Puppet) IntentFor(portal *Portal) *appservice.IntentAPI {
	if puppet != nil {
		if puppet.customIntent == nil || portal.EmailAddress == puppet.EmailAddress {
			return puppet.DefaultIntent()
		}
		return puppet.customIntent
	}
	return nil
}

// GetMXID implements bridge.Ghost.
func (*Puppet) GetMXID() id.UserID {
	panic("unimplemented")
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

func (br *MyBridge) GetPuppetByEmailAddress(addr string) *Puppet {
	// FIXME
	if addr == "" {
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
		return br.loadPuppet(context.TODO(), dbPuppet, addr)
	}
	return puppet
}

func (br *MyBridge) NewPuppet(dbPuppet *database.Puppet) *Puppet {
	return &Puppet{
		Puppet: dbPuppet,
		bridge: br,
		log:    br.ZLog.With().Str("user_id", dbPuppet.EmailAddress).Logger(),

		MXID: br.FormatPuppetMXID(dbPuppet.EmailAddress),
	}
}

func (br *MyBridge) FormatPuppetMXID(emailAddr string) id.UserID {
	return id.NewUserID(
		br.Config.Bridge.FormatUsername(emailAddr),
		br.Config.Homeserver.Domain,
	)
}

var userIDRegex *regexp.Regexp

func ParseFromRFC5322(addrr string) (string, error) {
	// FIXME
	return "Barry Gibbs <bg@example.com>", nil
}

func (br *MyBridge) ParsePuppetMXID(mxid id.UserID) (string, bool) {
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
			return "", false
		}

		parsed, err := mail.ParseAddress(parsedRFC5322)
		if err != nil {
			return "", false
		}
		return parsed.Address, true
	}

	return "", false
}

func (br *MyBridge) loadPuppet(ctx context.Context, dbPuppet *database.Puppet, email string) *Puppet {
	if dbPuppet == nil {
		if email == "" {
			return nil
		}
		dbPuppet = br.DB.Puppet.New()
		dbPuppet.EmailAddress = email
		err := dbPuppet.Insert(ctx)
		if err != nil {
			br.ZLog.Error().Err(err).Str("email_address", email).Msg("Failed to insert new puppet")
			return nil
		}
	}

	puppet := br.NewPuppet(dbPuppet)
	br.puppets[puppet.EmailAddress] = puppet
	if puppet.CustomMXID != "" {
		br.puppetsByCustomMXID[puppet.CustomMXID] = puppet
	}
	return puppet
}

func (br *MyBridge) GetPuppetByCustomMXID(mxid id.UserID) *Puppet {
	br.puppetsLock.Lock()
	defer br.puppetsLock.Unlock()

	puppet, ok := br.puppetsByCustomMXID[mxid]
	if !ok {
		dbPuppet, err := br.DB.Puppet.GetByCustomMXID(context.TODO(), mxid)
		if err != nil {
			br.ZLog.Err(err).Msg("Failed to get puppet from database")
			return nil
		}
		return br.loadPuppet(context.TODO(), dbPuppet, "")
	}
	return puppet
}

func (br *MyBridge) GetAllPuppetsWithCustomMXID() []*Puppet {
	puppets, err := br.DB.Puppet.GetAllWithCustomMXID(context.TODO())
	if err != nil {
		br.ZLog.Error().Err(err).Msg("Failed to get all puppets with custom MXID")
		return nil
	}
	return br.dbPuppetsToPuppets(puppets)
}

func (br *MyBridge) dbPuppetsToPuppets(dbPuppets []*database.Puppet) []*Puppet {
	br.puppetsLock.Lock()
	defer br.puppetsLock.Unlock()

	output := make([]*Puppet, len(dbPuppets))
	for index, dbPuppet := range dbPuppets {
		if dbPuppet == nil {
			continue
		}
		puppet, ok := br.puppets[dbPuppet.EmailAddress]
		if !ok {
			puppet = br.loadPuppet(context.TODO(), dbPuppet, "")
		}
		output[index] = puppet
	}
	return output
}

func Format(addr string) string {
	return strings.ReplaceAll(addr, "@", "_")

}

func (puppet *Puppet) UpdateInfo(ctx context.Context, source *User) {
	var err error

	log.Trace().Msg("Updating puppet info")

	update := false
	if puppet.EmailAddress != source.EmailAddress {
		puppet.EmailAddress = source.EmailAddress
		update = true
	}
	update = puppet.updateName(ctx, Format(puppet.EmailAddress)) || update
	if update {
		puppet.NameSet = false
		puppet.UpdateContactInfo(ctx)
		err = puppet.Update(ctx)
		if err != nil {
			log.Err(err).Msg("Failed to save puppet to database after updating")
		}
		go puppet.updatePortalMeta(ctx)
		log.Debug().Msg("Puppet info updated")
	}
}

func (puppet *Puppet) updateName(ctx context.Context, contact string) bool {
	// TODO set name quality
	newName := strings.ReplaceAll(contact, "@", "_")
	if puppet.NameSet && puppet.Name == newName {
		return false
	}
	puppet.Name = newName
	puppet.NameSet = false
	err := puppet.DefaultIntent().SetDisplayName(ctx, newName)
	if err != nil {
		zerolog.Ctx(ctx).Err(err).Msg("Failed to update user displayname")
	} else {
		puppet.NameSet = true
	}
	return true
}

func (puppet *Puppet) UpdateContactInfo(ctx context.Context) {
	if !puppet.bridge.SpecVersions.Supports(mautrix.BeeperFeatureArbitraryProfileMeta) || puppet.NameSet {
		return
	}

	identifiers := []string{
		fmt.Sprintf("email:%s", puppet.EmailAddress),
	}
	contactInfo := map[string]any{
		"com.beeper.bridge.identifiers": identifiers,
		"com.beeper.bridge.remote_id":   puppet.EmailAddress,
		"com.beeper.bridge.service":     "email",
		"com.beeper.bridge.network":     "email",
	}
	err := puppet.DefaultIntent().BeeperUpdateProfile(ctx, contactInfo)
	if err != nil {
		zerolog.Ctx(ctx).Err(err).Msg("Failed to store custom contact info in profile")
	} else {
		puppet.NameSet = true
	}
}

func (puppet *Puppet) updatePortalMeta(ctx context.Context) {
	for _, portal := range puppet.bridge.FindPrivateChatPortalsWith(puppet.EmailAddress) {
		// Get room create lock to prevent races between receiving contact info and room creation.
		portal.roomCreateLock.Lock()
		// portal.UpdateDMInfo(ctx, false)
		portal.roomCreateLock.Unlock()
	}
}
