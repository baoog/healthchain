package config

import (
	"log"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	HTTPAddr     string       `mapstructure:"http_addr"`
	APIKey       string       `mapstructure:"api_key"`
	FabricConfig FabricConfig `mapstructure:"fabric"`
	DBConfig     DBConfig     `mapstructure:"db"`
	VaultConfig  VaultConfig  `mapstructure:"vault"`
	MinIOConfig  MinIOConfig  `mapstructure:"minio"`
	JWTConfig    JWTConfig    `mapstructure:"jwt"`
	FCPConfig    FCPConfig    `mapstructure:"fcp"`
}

type FCPConfig struct {
	APIKey  string `mapstructure:"api_key"`
	BaseURL string `mapstructure:"base_url"`
}

type JWTConfig struct {
	SigningKey string        `mapstructure:"signing_key"`
	TTL        time.Duration `mapstructure:"ttl"` // in minutes
}

type VaultConfig struct {
	VaultAddr        string `mapstructure:"addr"`
	VaultToken       string `mapstructure:"token"`
	PrivateKeyPath   string `mapstructure:"private_key_path"`
	BundlePrivateKey string `mapstructure:"bundle_private_key"`
	HospitalIDPath   string `mapstructure:"hospital_id_path"`
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

type DBConfig struct {
	Host   string `mapstructure:"host"`
	Port   string `mapstructure:"port"`
	User   string `mapstructure:"user"`
	Pass   string `mapstructure:"pass"`
	Schema string `mapstructure:"schema"`
}

type MinIOConfig struct {
	Endpoint        string `mapstructure:"endpoint"`
	AccessKeyID     string `mapstructure:"access_key_id"`
	SecretAccessKey string `mapstructure:"secret_access_key"`
	Bucket          string `mapstructure:"bucket"`
	UseSSL          bool   `mapstructure:"use_ssl"`
}

func Load() *Config {
	//viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./chaincode/config")
	//viper.AddConfigPath("./config")
	//viper.AddConfigPath("../config")

	//viper.SetConfigName("config_b")
	//viper.AddConfigPath("./config")

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
