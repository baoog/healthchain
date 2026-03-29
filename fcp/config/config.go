package config

import (
	"log"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	HTTPAddr     string       `mapstructure:"http_addr"`
	FabricConfig FabricConfig `mapstructure:"fabric"`
	DBConfig     DBConfig     `mapstructure:"db"`
	VaultConfig  VaultConfig  `mapstructure:"vault"`
	JWTConfig    JWTConfig    `mapstructure:"jwt"`
}

type JWTConfig struct {
	SigningKey string        `mapstructure:"signing_key"`
	TTL        time.Duration `mapstructure:"ttl"` // in minutes
}

type VaultConfig struct {
	VaultAddr  string `mapstructure:"addr"`
	VaultToken string `mapstructure:"token"`
}

type DBConfig struct {
	Host   string `mapstructure:"host"`
	Port   string `mapstructure:"port"`
	User   string `mapstructure:"user"`
	Pass   string `mapstructure:"pass"`
	Schema string `mapstructure:"schema"`
}

type FabricConfig struct {
	TLSEnable    bool   `mapstructure:"tls_enable"`
	MSP          string `mapstructure:"msp"`
	Channel      string `mapstructure:"channel"`
	Chaincode    string `mapstructure:"chaincode"`
	PeerEndpoint string `mapstructure:"peer_endpoint"`
	TLSCertPath  string `mapstructure:"tls_cert_path"`
	CertPath     string `mapstructure:"cert_path"`
	KeyPath      string `mapstructure:"key_path"`
	StartBlock   string `mapstructure:"start_block"`
}

func Load() *Config {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./chaincode/config")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("../config")

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("read config file: %v", err)
	}

	viper.SetConfigType("env")
	viper.SetConfigFile("./config/.env")

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		log.Fatalf("unmarshal config: %v", err)
	}

	return &cfg
}
