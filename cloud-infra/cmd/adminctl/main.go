package main

import (
	"fmt"
	"log"
	"os"

	"github.com/soul-room/cloud-infra/internal/config"
	"github.com/soul-room/cloud-infra/internal/storage"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	switch os.Args[1] {
	case "migrate":
		fmt.Println("migrations are available in migrations/; apply with your PostgreSQL migration tool")
	case "organizations":
		cfg := config.Load()
		store, err := storage.Open(cfg.StorePath)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%+v\n", store.ListOrganizations())
	default:
		usage()
	}
}

func usage() {
	fmt.Println("adminctl migrate | organizations")
	os.Exit(2)
}
