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

	Signal struct {
		DeviceName string `yaml:"device_name"`
	} `yaml:"signal"`

	Bridge BridgeConfig `yaml:"bridge"`
}
