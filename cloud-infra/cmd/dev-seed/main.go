package main

import (
	"fmt"
	"log"
	"net/mail"
	"strings"

	"github.com/soul-room/cloud-infra/internal/auth"
	"github.com/soul-room/cloud-infra/internal/config"
	"github.com/soul-room/cloud-infra/internal/storage"
)

func main() {
	cfg := config.Load()
	store, err := storage.Open(cfg.StorePath)
	if err != nil {
		log.Fatal(err)
	}
	name := strings.TrimSpace(cfg.OrganizationName)
	slug := strings.ToLower(strings.TrimSpace(cfg.OrganizationSlug))
	email := strings.ToLower(strings.TrimSpace(cfg.AdminEmail))
	if name == "" || slug == "" {
		log.Fatal("SOULROOM_ORGANIZATION_NAME and SOULROOM_ORGANIZATION_SLUG must not be empty")
	}
	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email {
		log.Fatal("SOULROOM_ADMIN_EMAIL must be a valid email address")
	}
	if len(cfg.AdminPassword) < 12 {
		log.Fatal("SOULROOM_ADMIN_PASSWORD must contain at least 12 characters")
	}
	hash, err := (auth.PBKDF2Hasher{}).Hash(cfg.AdminPassword)
	if err != nil {
		log.Fatal(err)
	}
	created, err := store.BootstrapOrganizationOwner(name, slug, email, hash)
	if err != nil {
		log.Fatal(err)
	}
	if !created {
		fmt.Println("installation already initialized; bootstrap credentials were not reapplied")
		return
	}
	fmt.Printf("initialized %s owner %s; no fleet data was seeded\n", name, email)
}
