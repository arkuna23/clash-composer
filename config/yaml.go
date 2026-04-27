package config

import (
	"errors"

	"gopkg.in/yaml.v3"
)

var ErrNilRawConfig = errors.New("config: nil RawConfig")

func UnmarshalRawConfig(data []byte) (*RawConfig, error) {
	cfg := DefaultRawConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func MarshalRawConfig(cfg *RawConfig) ([]byte, error) {
	if cfg == nil {
		return nil, ErrNilRawConfig
	}
	return yaml.Marshal(cfg)
}
