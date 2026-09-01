package config

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"slices"

	"github.com/spf13/viper"
	"github.com/tab58/huma-http-server/config"
)

type Environment string

type Config struct {
	Env               config.AppMode `mapstructure:"ENV"`
	ServerPort        string         `mapstructure:"SERVER_PORT"`     // the port to bind the listening server to
	ExternalServerURL string         `mapstructure:"SERVER_BASE_URL"` // the URL to ping from external services (used for webhooks)

	// AWS specific configuration
	AWSRegion string `mapstructure:"AWS_REGION"`

	// AWS S3 configuration for file uploads
	S3BaseEndpoint string `mapstructure:"S3_BASE_ENDPOINT"`
	S3UploadBucket string `mapstructure:"S3_UPLOAD_BUCKET"`

	// for Asynq server and caching
	RedisURL string `mapstructure:"REDIS_URL" json:"-"` // DSN may carry credentials; keep out of the config log line

	// main database
	MainDBURL string `mapstructure:"MAIN_DB_URL" json:"-"` // DSN carries credentials; keep out of the config log line

	// Cloudflare Access verification for /admin routes (set both or neither;
	// unset in production fails closed, unset in development leaves admin open)
	CFAccessTeamDomain string `mapstructure:"CF_ACCESS_TEAM_DOMAIN"` // e.g. team.cloudflareaccess.com
	CFAccessAUD        string `mapstructure:"CF_ACCESS_AUD"`         // Access application AUD tag
}

func bindEnvVars() {
	t := reflect.TypeFor[Config]()
	for i := range t.NumField() {
		field := t.Field(i)
		if tag := field.Tag.Get("mapstructure"); tag != "" {
			viper.BindEnv(tag)
		}
	}
}

func Load() (*Config, error) {
	cfg := &Config{}

	// pull in environment variables
	bindEnvVars()
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	} else {
		fmt.Println("config file not found, using environment variables...")
	}
	viper.AutomaticEnv()

	// build the config object
	err := viper.Unmarshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// print configuration (secret fields carry json:"-" so they never land in logs)
	cfgJson, err := json.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}
	fmt.Println("config: ", string(cfgJson))

	// validate the config
	if err := Validate(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "failed to validate config: %v\n", err)
		os.Exit(1)
	}

	return cfg, nil
}

func Validate(cfg *Config) error {
	if cfg.Env == "" {
		return fmt.Errorf("ENV is required")
	}

	envs := []config.AppMode{config.AppModeDevelopment, config.AppModeProduction}
	if !slices.Contains(envs, cfg.Env) {
		return fmt.Errorf("ENV must be one of %v", envs)
	}

	if cfg.ServerPort == "" {
		return fmt.Errorf("SERVER_PORT is required")
	}

	return nil
}
