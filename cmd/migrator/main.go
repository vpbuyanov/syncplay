package main

import (
	"log"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/vpbuyanov/syncplay/internal/config"
)

func main() {
	cfg := config.MustConfig(nil)

	sourceURL := "file://migrations"

	m, err := migrate.New(sourceURL, cfg.Postgres.String())
	if err != nil {
		log.Println(cfg.Postgres.String())
		log.Fatal("err create migrate")
	}

	if err = m.Up(); err != nil && err.Error() != "no change" {
		log.Fatal("err up")
	}

	log.Println("Success")
}
