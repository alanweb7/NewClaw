package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"newclaw/internal/app"
	"newclaw/internal/httpapi"
	"newclaw/internal/modelconfig"
	"newclaw/internal/skills"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}
	root, err := os.Getwd()
	if err != nil {
		fatal(err)
	}

	switch os.Args[1] {
	case "init":
		_, _, err := app.Bootstrap(root)
		if err != nil {
			fatal(err)
		}
		fmt.Println("NewClaw initialized at", root)
	case "run":
		runCmd(root, os.Args[2:])
	case "chat":
		chatCmd(root, os.Args[2:])
	case "session":
		sessionCmd(root, os.Args[2:])
	case "skill":
		skillCmd(root, os.Args[2:])
	case "model":
		modelCmd(root, os.Args[2:])
	default:
		printUsage()
		os.Exit(1)
	}
}

func runCmd(root string, args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	agentID := fs.String("agent", "main", "agent id")
	_ = fs.Parse(args)

	cfg, svc, err := app.Bootstrap(root)
	if err != nil {
		fatal(err)
	}
	_ = agentID

	srv := httpapi.New(root, svc)
	addr := fmt.Sprintf("%s:%d", cfg.HTTP.Host, cfg.HTTP.Port)
	fmt.Println("NewClaw HTTP listening on", addr)
	fatal(http.ListenAndServe(addr, srv.Handler()))
}

func chatCmd(root string, args []string) {
	fs := flag.NewFlagSet("chat", flag.ExitOnError)
	msg := fs.String("message", "", "user message")
	sessionID := fs.String("session", "", "session id")
	agentID := fs.String("agent", "main", "agent id")
	_ = fs.Parse(args)
	if *msg == "" {
		fatal(errors.New("--message is required"))
	}

	_, svc, err := app.Bootstrap(root)
	if err != nil {
		fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	resp, err := svc.SendMessage(ctx, *agentID, *sessionID, *msg)
	if err != nil {
		fatal(err)
	}
	fmt.Println(resp.Content)
}

func sessionCmd(root string, args []string) {
	if len(args) == 0 {
		fatal(errors.New("session subcommand required: list|history"))
	}
	_, svc, err := app.Bootstrap(root)
	if err != nil {
		fatal(err)
	}
	switch args[0] {
	case "list":
		list, err := svc.ListSessions("main")
		if err != nil {
			fatal(err)
		}
		printJSON(list)
	case "history":
		fs := flag.NewFlagSet("history", flag.ExitOnError)
		id := fs.String("id", "", "session id")
		_ = fs.Parse(args[1:])
		if *id == "" {
			fatal(errors.New("--id is required"))
		}
		history, err := svc.History("main", *id)
		if err != nil {
			fatal(err)
		}
		printJSON(history)
	default:
		fatal(errors.New("unknown session subcommand"))
	}
}

func skillCmd(root string, args []string) {
	if len(args) == 0 || args[0] != "list" {
		fatal(errors.New("skill subcommand required: list"))
	}
	_, _, err := app.Bootstrap(root)
	if err != nil {
		fatal(err)
	}
	list, err := skills.List(root)
	if err != nil {
		fatal(err)
	}
	printJSON(list)
}

func modelCmd(root string, args []string) {
	if len(args) == 0 || args[0] != "config" {
		fatal(errors.New("model subcommand required: config"))
	}
	_, _, err := app.Bootstrap(root)
	if err != nil {
		fatal(err)
	}
	if err := modelconfig.Run(root); err != nil {
		fatal(err)
	}
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  newclaw init")
	fmt.Println("  newclaw run --agent main")
	fmt.Println("  newclaw chat --message \"...\" [--session <id>]")
	fmt.Println("  newclaw session list")
	fmt.Println("  newclaw session history --id <id>")
	fmt.Println("  newclaw skill list")
	fmt.Println("  newclaw model config")
}

func printJSON(v interface{}) {
	b, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(b))
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
