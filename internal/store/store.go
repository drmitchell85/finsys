package store

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strconv"

	"github.com/drmitchell85/finsys/internal/config"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

func InitDB(config config.Config) (*sql.DB, error) {
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		config.Database.Host, config.Database.Port, config.Database.User, config.Database.Password, config.Database.Name)

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		return nil, fmt.Errorf("could not connect to db: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("error connecting to db: %w", err)
	}

	log.Println("connected to db")

	return db, nil
}

func InitCache(ctx context.Context, config config.Config) (*redis.Client, error) {
	rds := redis.NewClient(&redis.Options{
    Addr:     fmt.Sprintf("%s:%s", config.Redis.Host, strconv.Itoa(config.Redis.Port)),
		Password: "",
		DB:       0,
	})

	_, err := rds.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to init cache: %w", err)
	}

	log.Println("connected to cache")

	return rds, nil
}
