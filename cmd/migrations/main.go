package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/pressly/goose/v3"
	_ "tarkov-build-optimiser/migrations"
)

var (
	flags = flag.NewFlagSet("goose", flag.ExitOnError)
	dir   = flags.String("dir", ".", "directory with migration files")
)

func main() {
	fmt.Println("start")
	err := flags.Parse(os.Args[1:])
	if err != nil {
		return
	}
	args := flags.Args()

	fmt.Println("args parsed")

	if len(args) < 3 {
		fmt.Println(args)
		flags.Usage()
		return
	}

	dbString, command := args[1], args[2]

	db, err := goose.OpenDBWithDriver("postgres", dbString)
	if err != nil {
		log.Fatalf("goose: failed to open DB: %v\n", err)
	}

	defer func() {
		if err := db.Close(); err != nil {
			log.Fatalf("goose: failed to close DB: %v\n", err)
		}
	}()

	fmt.Println("applying migrations")

	var arguments []string
	if len(args) > 3 {
		arguments = append(arguments, args[3:]...)
	}

	if err := goose.RunContext(context.Background(), command, db, *dir, arguments...); err != nil {
		log.Fatalf("goose %v: %v", command, err)
	}
}
