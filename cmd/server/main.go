package main

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vpbuyanov/syncplay/internal/config"
	"github.com/vpbuyanov/syncplay/internal/model"
	"github.com/vpbuyanov/syncplay/internal/server"
	"github.com/vpbuyanov/syncplay/internal/store/postgresql"
)

func main() {
	cfg := config.MustConfig(nil)
	ctx := context.Background()

	db, err := pgxpool.New(ctx, cfg.Postgres.String())
	if err != nil {
		log.Fatal("error connect db")
	}
	defer db.Close()

	rep := postgresql.NewRepos(db)

	modelR := model.NewModelRoom(rep)

	s, err := server.NewServer(cfg.Server, modelR)
	if err != nil {
		log.Panic(err)
	}

	err = s.Listen()
	if err != nil {
		log.Panic(err)
	}
}
