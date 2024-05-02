package config

import (
	"maunium.net/go/mautrix/bridge/bridgeconfig"
)

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
