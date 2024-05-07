package config

import (
	"maunium.net/go/mautrix/bridge/bridgeconfig"
	"maunium.net/go/mautrix/id"
)

func (config *Config) CanAutoDoublePuppet(userID id.UserID) bool {
	_, homeserver, _ := userID.Parse()
	_, hasSecret := config.Bridge.DoublePuppetConfig.SharedSecretMap[homeserver]

	return hasSecret
}

type Config struct {
	*bridgeconfig.BaseConfig `yaml:",inline"`

	Metrics struct {
		Enabled bool   `yaml:"enabled"`
		Listen  string `yaml:"listen"`
	} `yaml:"metrics"`

	Email struct {
		EmailAddress string `yaml:"emailAddress"`
	} `yaml:"email"`

	Bridge BridgeConfig `yaml:"bridge"`
}
