package store

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/drmitchell85/finsys/internal/config"
	_ "github.com/lib/pq"
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
