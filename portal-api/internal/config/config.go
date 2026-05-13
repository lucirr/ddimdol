package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	DatabaseURL      string
	NatsURL          string
	KeycloakURL      string
	KeycloakRealm    string
	KeycloakClientID string
	HarborURL        string
	HarborUser       string
	HarborPassword   string
	ServerPort       int
	AgentPort        int
}

func Load() (*Config, error) {
	viper.SetDefault("SERVER_PORT", 8080)
	viper.SetDefault("AGENT_PORT", 8081)
	viper.AutomaticEnv()

	cfg := &Config{
		DatabaseURL:      viper.GetString("DATABASE_URL"),
		NatsURL:          viper.GetString("NATS_URL"),
		KeycloakURL:      viper.GetString("KEYCLOAK_URL"),
		KeycloakRealm:    viper.GetString("KEYCLOAK_REALM"),
		KeycloakClientID: viper.GetString("KEYCLOAK_CLIENT_ID"),
		HarborURL:        viper.GetString("HARBOR_URL"),
		HarborUser:       viper.GetString("HARBOR_USER"),
		HarborPassword:   viper.GetString("HARBOR_PASSWORD"),
		ServerPort:       viper.GetInt("SERVER_PORT"),
		AgentPort:        viper.GetInt("AGENT_PORT"),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	return cfg, nil
}
