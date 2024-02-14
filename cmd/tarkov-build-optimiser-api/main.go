package main

import (
	"tarkov-build-optimiser/internal/db"
	"tarkov-build-optimiser/internal/env"
	"tarkov-build-optimiser/internal/router"
)

func main() {
	env, err := env.Get()
	if err != nil {
		panic(err)
	}

	db, err := db.NewDatabase(db.Config{
		Host:     env.PgHost,
		Port:     env.PgPort,
		User:     env.PgUser,
		Password: env.PgPassword,
		Name:     env.PgName,
	})
	if err != nil {
		panic(err)
	}

	router := router.NewRouter(router.RouterConfig{DB: db})

	router.Start(":8080")
}
