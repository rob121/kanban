package cmd

import (
	"flag"
	"fmt"
	"os"

	"github.com/rob121/kanban/internal/bootstrap"
	"github.com/rob121/kanban/internal/config"
	"github.com/rob121/kanban/internal/database"
)

func RunBootstrapAdmin(args []string) error {
	fs := flag.NewFlagSet("bootstrap-admin", flag.ExitOnError)
	username := fs.String("username", "", "admin username (required)")
	email := fs.String("email", "", "admin email (required)")
	password := fs.String("password", "", "admin password (generated if omitted)")
	name := fs.String("name", "", "display name (defaults to username)")
	update := fs.Bool("update", false, "update an existing user with the same username")
	if err := fs.Parse(args); err != nil {
		return err
	}

	config.Load()
	if err := database.Connect(); err != nil {
		return err
	}

	generated, err := bootstrap.CreateAdmin(bootstrap.AdminOptions{
		Username: *username,
		Email:    *email,
		Password: *password,
		Name:     *name,
		Update:   *update,
	})
	if err != nil {
		return err
	}

	verb := "created"
	if *update {
		verb = "updated"
	}
	fmt.Printf("Admin user %q %s (%s).\n", *username, verb, *email)
	if generated != "" {
		fmt.Printf("Generated password: %s\n", generated)
		fmt.Println("Store this password securely; it will not be shown again.")
	}
	return nil
}

func Usage() {
	fmt.Fprintf(os.Stderr, "Usage:\n")
	fmt.Fprintf(os.Stderr, "  %s                          start the web server\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s bootstrap-admin [flags]  create an admin user\n\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "bootstrap-admin flags:\n")
	fs := flag.NewFlagSet("bootstrap-admin", flag.ExitOnError)
	fs.SetOutput(os.Stderr)
	fs.String("username", "", "admin username (required)")
	fs.String("email", "", "admin email (required)")
	fs.String("password", "", "admin password (generated if omitted)")
	fs.String("name", "", "display name (defaults to username)")
	fs.Bool("update", false, "update an existing user with the same username")
	fs.PrintDefaults()
}
