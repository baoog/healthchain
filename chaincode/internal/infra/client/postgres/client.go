package postgres

import (
	"consentcc/config"
	"fmt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
	"net/url"
)

type Client struct {
	// Placeholder for Postgres client fields
	DB *gorm.DB
}

func NewDBClient(cfg *config.Config) *Client {
	db, err := setupPostgresConnection(cfg)
	if err != nil {
		panic(fmt.Sprintf("failed to connect to Postgres: %v", err))
	}
	return &Client{DB: db}
}

func setupPostgresConnection(cfg *config.Config) (*gorm.DB, error) {
	// Placeholder for actual Postgres connection setup using GORM
	query := url.Values{}
	query.Add("database", cfg.DBConfig.Schema)

	dbUrl := &url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(cfg.DBConfig.User, cfg.DBConfig.Pass),
		Host:     fmt.Sprintf("%s:%s", cfg.DBConfig.Host, cfg.DBConfig.Port),
		RawQuery: query.Encode(),
	}

	connStr := dbUrl.String()

	db, err := gorm.Open(postgres.Open(connStr), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	log.Println("Connected to Postgres successfully")
	return db, nil
}
