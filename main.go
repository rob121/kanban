package main

import (
	"embed"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/rob121/kanban/internal/auth"
	"github.com/rob121/kanban/internal/cmd"
	"github.com/rob121/kanban/internal/config"
	"github.com/rob121/kanban/internal/database"
	"github.com/rob121/kanban/internal/mail"
	"github.com/rob121/kanban/internal/server"
)

//go:embed templates/*
var views embed.FS

//go:embed static/*
var staticFiles embed.FS

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "bootstrap-admin":
			if err := cmd.RunBootstrapAdmin(os.Args[2:]); err != nil {
				log.Fatal(err)
			}
			return
		case "help", "-h", "--help":
			cmd.Usage()
			return
		default:
			if looksLikeBootstrapFlags(os.Args[1:]) {
				fmt.Fprintf(os.Stderr, "Admin user flags require the bootstrap-admin subcommand.\n\n")
				cmd.Usage()
				os.Exit(2)
			}
		}
	}

	config.Load()
	mail.Init()
	auth.Init()

	if err := database.Connect(); err != nil {
		log.Fatalf("database: %v", err)
	}

	srv, err := server.New(views, staticFiles)
	if err != nil {
		log.Fatalf("server: %v", err)
	}

	addr := fmt.Sprintf(":%d", config.C.Port)
	log.Printf("Kanban listening on %s", config.C.BaseURL)
	if err := http.ListenAndServe(addr, srv.Handler()); err != nil {
		log.Fatal(err)
	}
}

func looksLikeBootstrapFlags(args []string) bool {
	for _, a := range args {
		a = strings.TrimPrefix(a, "-")
		a = strings.TrimPrefix(a, "-")
		switch strings.SplitN(a, "=", 2)[0] {
		case "username", "email", "password", "name":
			return true
		}
	}
	return false
}
