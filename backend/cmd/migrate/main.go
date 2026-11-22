// Command migrate manages the database schema and reference data.
//
//	go run ./cmd/migrate up|down|status|seed|create <name> [-test]
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/pressly/goose/v3"

	"mauzo/internal/config"
	"mauzo/internal/core/db"
	"mauzo/internal/core/logger"
	"mauzo/seeds"
)

func main() {
	useTest := flag.Bool("test", false, "target TEST_DB_NAME instead of DB_NAME")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: migrate [-test] up|down|status|seed|create <name>")
	}
	flag.Parse()
	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(2)
	}
	cmd := flag.Arg(0)

	if cmd == "create" {
		if flag.NArg() < 2 {
			log.Fatal("create needs a name, e.g. create tenants_create")
		}
		goose.SetSequential(true)
		if err := goose.Create(nil, "migrations", flag.Arg(1), "sql"); err != nil {
			log.Fatal(err)
		}
		return
	}

	switch cmd {
	case "up", "down", "status", "seed":
	default:
		flag.Usage()
		os.Exit(2)
	}

	if err := run(cmd, *useTest); err != nil {
		log.Fatal(err)
	}
}

func run(cmd string, useTest bool) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	name := cfg.DB.Name
	if useTest {
		name = cfg.DB.TestName
	}
	ctx := context.Background()
	conn, err := db.Open(ctx, cfg.DB, name)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()

	switch cmd {
	case "up":
		return db.MigrateUp(ctx, conn.DB)
	case "down":
		return db.MigrateDown(ctx, conn.DB)
	case "status":
		return db.MigrateStatus(ctx, conn.DB)
	default: // "seed"
		return seeds.Run(ctx, conn, logger.New(cfg.Log.Level, true))
	}
}
