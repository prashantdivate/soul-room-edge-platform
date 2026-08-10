package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/unified-fleet/cloud-infra/internal/auth"
	"github.com/unified-fleet/cloud-infra/internal/config"
	"github.com/unified-fleet/cloud-infra/internal/model"
	"github.com/unified-fleet/cloud-infra/internal/storage"
)

func main() {
	cfg := config.Load()
	store, err := storage.Open(cfg.StorePath)
	if err != nil {
		log.Fatal(err)
	}
	hash, err := (auth.PBKDF2Hasher{}).Hash("change-me-local")
	if err != nil {
		log.Fatal(err)
	}
	org, err := store.CreateOrganization("My Organization", "local")
	if err != nil && err != storage.ErrDuplicate {
		log.Fatal(err)
	}
	if err == storage.ErrDuplicate {
		for _, existing := range store.ListOrganizations() {
			if existing.Slug == "local" {
				org = existing
			}
		}
	}
	user, err := store.UpsertUserPassword(strings.ToLower("admin@example.local"), hash)
	if err != nil {
		log.Fatal(err)
	}
	if err := store.AddMembership(model.Membership{TenantID: org.ID, UserID: user.ID, Role: "organization_owner"}); err != nil && err != storage.ErrDuplicate {
		log.Fatal(err)
	}
	fmt.Println("initialized local administrator; no fleet data was seeded")
}
