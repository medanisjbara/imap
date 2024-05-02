package config

import (
	"strings"
	"text/template"
	"time"

	"maunium.net/go/mautrix/bridge/bridgeconfig"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

type BridgeConfig struct {
	// Define your configuration fields here
	UsernameTemplate      string `yaml:"username_template"`
	DisplaynameTemplate   string `yaml:"displayname_template"`
	PrivateChatPortalMeta string `yaml:"private_chat_portal_meta"`
	PortalMessageBuffer   int    `yaml:"portal_message_buffer"`
	// Add more configuration fields as needed

    DoublePuppetConfig bridgeconfig.DoublePuppetConfig `yaml:",inline"`
    Encryption bridgeconfig.EncryptionConfig `yaml:"encryption"`
    CommandPrefix         string
    ManagementRoomText      bridgeconfig.ManagementRoomTexts `yaml:"management_room_text"`


	// Configuration for backfilling messages
	Backfill struct {
		Enabled              bool `yaml:"enabled"`
		InboxFetchPages      int  `yaml:"inbox_fetch_pages"`
		HistoryFetchPages    int  `yaml:"history_fetch_pages"`
		CatchupFetchPages    int  `yaml:"catchup_fetch_pages"`
		UnreadHoursThreshold int  `yaml:"unread_hours_threshold"`
		Queue                struct {
			PagesAtOnce       int           `yaml:"pages_at_once"`
			MaxPages          int           `yaml:"max_pages"`
			SleepBetweenTasks time.Duration `yaml:"sleep_between_tasks"`
			DontFetchXMA      bool          `yaml:"dont_fetch_xma"`
		} `yaml:"queue"`
	} `yaml:"backfill"`
	DisableXMA bool `yaml:"disable_xma"`

	// Configuration for provisioning
	Provisioning struct {
		Prefix         string `yaml:"prefix"`
		SharedSecret   string `yaml:"shared_secret"`
		DebugEndpoints bool   `yaml:"debug_endpoints"`
	} `yaml:"provisioning"`

	// Configuration for permissions
	Permissions bridgeconfig.PermissionConfig `yaml:"permissions"`

	// Configuration for relay bot
	Relay RelaybotConfig `yaml:"relay"`

	// Internal fields for template parsing
	usernameTemplate    *template.Template `yaml:"-"`
	displaynameTemplate *template.Template `yaml:"-"`
}

// Validate validates the bridge configuration.
func (bc *BridgeConfig) Validate() error {
	// Implement validation logic here
	return nil
}

// UnmarshalYAML unmarshals the YAML data into BridgeConfig struct.
func (bc *BridgeConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// Implement unmarshaling logic here
	return nil
}

// GetDoublePuppetConfig returns the double puppet configuration.
func (bc BridgeConfig) GetDoublePuppetConfig() bridgeconfig.DoublePuppetConfig {
	return bc.DoublePuppetConfig
}

// GetEncryptionConfig returns the encryption configuration.
func (bc BridgeConfig) GetEncryptionConfig() bridgeconfig.EncryptionConfig {
	return bc.Encryption
}

// GetCommandPrefix returns the command prefix.
func (bc BridgeConfig) GetCommandPrefix() string {
	return bc.CommandPrefix
}

// GetManagementRoomTexts returns the management room texts.
func (bc BridgeConfig) GetManagementRoomTexts() bridgeconfig.ManagementRoomTexts {
	return bc.ManagementRoomText
}

// FormatUsername formats the username based on the template.
func (bc BridgeConfig) FormatUsername(userID string) string {
	var buffer strings.Builder
	_ = bc.usernameTemplate.Execute(&buffer, userID)
	return buffer.String()
}

// DisplaynameParams represents the parameters for formatting display name.
type DisplaynameParams struct {
	DisplayName string
	Username    string
	ID          int64
}

// FormatDisplayname formats the display name based on the parameters.
func (bc BridgeConfig) FormatDisplayname(params DisplaynameParams) string {
	var buffer strings.Builder
	_ = bc.displaynameTemplate.Execute(&buffer, params)
	return buffer.String()
}

// RelaybotConfig represents the configuration for relay bot.
type RelaybotConfig struct {
	Enabled          bool                         `yaml:"enabled"`
	AdminOnly        bool                         `yaml:"admin_only"`
	MessageFormats   map[event.MessageType]string `yaml:"message_formats"`
	messageTemplates *template.Template           `yaml:"-"`
}

// UnmarshalYAML unmarshals the YAML data into RelaybotConfig struct.
func (rc *RelaybotConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// Implement unmarshaling logic here
	return nil
}

// FormatMessage formats the message based on the content, sender, and member information.
func (rc *RelaybotConfig) FormatMessage(content *event.MessageEventContent, sender id.UserID, member event.MemberEventContent) (string, error) {
	// Implement message formatting logic here
	return "", nil
}
