package main

import (
	"tarkov-build-optimiser/internal/db"
	"tarkov-build-optimiser/internal/router"
)

func main() {
	dbClient, err := db.CreateBuildOptimiserDBClient()
	if err != nil {
		panic(err)
	}

	cfg := router.Config{DB: dbClient}
	r := router.NewRouter(cfg)

	err = r.Start(":8080")
	if err != nil {
		return
	}
}
