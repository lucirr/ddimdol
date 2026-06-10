package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	DatabaseURL       string
	NatsURL           string
	KeycloakURL       string
	KeycloakRealm     string
	KeycloakClientID  string
	HarborURL         string
	HarborUser        string
	HarborPassword    string
	ServerPort        int
	AgentPort         int
	AgentTLSEnabled   bool
	AgentTLSCAPath    string
	AgentTLSCertPath  string
	AgentTLSKeyPath   string
}

func Load() (*Config, error) {
	viper.SetDefault("SERVER_PORT", 8080)
	viper.SetDefault("AGENT_PORT", 8081)
	viper.SetDefault("AGENT_TLS_ENABLED", false)
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
		AgentTLSEnabled:  viper.GetBool("AGENT_TLS_ENABLED"),
		AgentTLSCAPath:   viper.GetString("AGENT_TLS_CA"),
		AgentTLSCertPath: viper.GetString("AGENT_TLS_CERT"),
		AgentTLSKeyPath:  viper.GetString("AGENT_TLS_KEY"),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	if cfg.AgentTLSEnabled && (cfg.AgentTLSCAPath == "" || cfg.AgentTLSCertPath == "" || cfg.AgentTLSKeyPath == "") {
		return nil, fmt.Errorf("AGENT_TLS_CA, AGENT_TLS_CERT, and AGENT_TLS_KEY are required when AGENT_TLS_ENABLED is true")
	}

	return cfg, nil
}
