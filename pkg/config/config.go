package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	Backup struct {
		Enabled   bool   `mapstructure:"enabled"`
		Retention string `mapstructure:"retention"`
		Path      string `mapstructure:"path"`
	} `mapstructure:"backup"`

	Git struct {
		AutoCommit    bool   `mapstructure:"auto_commit"`
		DefaultBranch string `mapstructure:"default_branch"`
		AutoInit      bool   `mapstructure:"auto_init"`
	} `mapstructure:"git"`

	UI struct {
		Theme         string `mapstructure:"theme"`
		ConfirmCreate bool   `mapstructure:"confirm_create"`
	} `mapstructure:"ui"`

	Validators []Validator `mapstructure:"validators"`
	Templates  []Template  `mapstructure:"templates"`
}

type Validator struct {
	Extension string   `mapstructure:"extension"`
	Command   string   `mapstructure:"command"`
	Args      []string `mapstructure:"args"`
}

type Template struct {
	Name        string                 `mapstructure:"name"`
	Description string                 `mapstructure:"description"`
	Structure   map[string]interface{} `mapstructure:"structure"`
}

func Load() (*Config, error) {
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
