// Command seed-admin creates the initial super_admin account. Reads
// SEED_ADMIN_USERNAME, SEED_ADMIN_EMAIL, and SEED_ADMIN_PASSWORD from the
// environment and refuses to run if a super_admin already exists — this
// is a one-time setup command, not a routine one.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"

	"github.com/wageesha/sl-finance-compare/internal/adminauth"
	"github.com/wageesha/sl-finance-compare/internal/db"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("seed-admin: %v", err)
	}
}

func run() error {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		return err
	}

	username := os.Getenv("SEED_ADMIN_USERNAME")
	email := os.Getenv("SEED_ADMIN_EMAIL")
	password := os.Getenv("SEED_ADMIN_PASSWORD")
	if username == "" || email == "" || password == "" {
		return errors.New("SEED_ADMIN_USERNAME, SEED_ADMIN_EMAIL, and SEED_ADMIN_PASSWORD must all be set")
	}
	if len(password) < 8 {
		return errors.New("SEED_ADMIN_PASSWORD must be at least 8 characters")
	}

	connString := os.Getenv("DATABASE_URL")
	if connString == "" {
		return errors.New("DATABASE_URL is not set (check your .env file)")
	}

	ctx := context.Background()
	database, err := db.New(ctx, connString)
	if err != nil {
		return err
	}
	defer database.Close()

	count, err := database.CountSuperAdmins(ctx)
	if err != nil {
		return fmt.Errorf("check existing super admins: %w", err)
	}
	if count > 0 {
		return errors.New("a super_admin already exists — this command is for initial setup only; manage further admins from Settings once logged in")
	}

	hash, err := adminauth.HashPassword(password)
	if err != nil {
		return err
	}
	id, err := database.CreateAdminUser(ctx, username, email, hash, "super_admin")
	if err != nil {
		return fmt.Errorf("create super admin: %w", err)
	}

	fmt.Printf("Created super_admin %q (id %d). Log in at /admin/login and change the seeded password from Settings.\n", username, id)
	return nil
}
