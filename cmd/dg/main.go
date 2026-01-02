package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"dg/internal/cron"
	"dg/internal/run"
	"dg/internal/version"
)

func main() {
	if len(os.Args) < 2 {
		helpCmd()
		return
	}

	switch os.Args[1] {
	case "run":
		runCmd()
	case "install":
		installCmd()
	case "uninstall":
		uninstallCmd()
	case "help":
		helpCmd()
	case "version":
		fmt.Println(version.Version)
	default:
		helpCmd()
	}
}

func runCmd() {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	cfg := fs.String("config", "./.dg/config.yml", "path to config.yml")
	_ = fs.Parse(os.Args[2:])
	cfgAbs, _ := filepath.Abs(*cfg)
	code := run.Run(cfgAbs)
	os.Exit(code)
}

func installCmd() {
	fs := flag.NewFlagSet("install", flag.ExitOnError)
	cfg := fs.String("config", "./.dg/config.yml", "path to config.yml")
	_ = fs.Parse(os.Args[2:])
	cfgAbs, _ := filepath.Abs(*cfg)
	if err := cron.Install(cfgAbs); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func uninstallCmd() {
	fs := flag.NewFlagSet("uninstall", flag.ExitOnError)
	cfg := fs.String("config", "./.dg/config.yml", "path to config.yml")
	_ = fs.Parse(os.Args[2:])
	cfgAbs, _ := filepath.Abs(*cfg)
	if err := cron.Uninstall(cfgAbs); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func helpCmd() {
	fmt.Println("dg run [-config ./.dg/config.yml]")
	fmt.Println("dg install [-config ./.dg/config.yml]")
	fmt.Println("dg uninstall [-config ./.dg/config.yml]")
	fmt.Println("dg help")
	fmt.Println("dg version")
}
