package postgres

import (
	"context"
	"fcp/config"
	"fmt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
	"net/url"
)

type txKeyType struct{}

var txKey = txKeyType{}

type Client struct {
	// Placeholder for Postgres client fields
	db *gorm.DB
}

func NewDBClient(cfg *config.Config) *Client {
	db, err := setupPostgresConnection(cfg)
	if err != nil {
		panic(fmt.Sprintf("failed to connect to Postgres: %v", err))
	}
	return &Client{db: db}
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

	sqlDB, _ := db.DB()

	if err := sqlDB.Ping(); err != nil {
		return nil, err
	}
	log.Println("Connected to Postgres successfully")
	return db, nil
}

func (c *Client) GetDB(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(txKey).(*gorm.DB); ok {
		return tx.WithContext(ctx)
	}
	return c.db.WithContext(ctx)
}

func (c *Client) WithTx(ctx context.Context, txFunc func(txCtx context.Context) error) error {
	err := c.db.WithContext(ctx).Transaction(func(tx *gorm.DB) (err error) {
		txCtx := context.WithValue(ctx, txKey, tx)
		err = txFunc(txCtx)
		if err != nil {
			return err
		}

		return nil
	}, nil)
	return err
}
