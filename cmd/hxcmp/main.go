package main

import (
	"fmt"
	"os"

	"github.com/pthm/hxcmp/lib/generator"
)

const version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "generate":
		if err := runGenerate(args); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "clean":
		if err := runClean(args); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "version":
		fmt.Printf("hxcmp version %s\n", version)
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`hxcmp - HTMX Component System for Go

Usage:
  hxcmp <command> [arguments]

Commands:
  generate [packages]   Generate code for components (e.g., ./... or ./components/...)
  clean [packages]      Remove generated files (*_hx.go)
  version               Print version
  help                  Show this help

Options for generate:
  --dry-run             Show what would be generated without writing files

Examples:
  hxcmp generate ./...                    Generate for all packages
  hxcmp generate ./components/fileviewer  Generate for specific package
  hxcmp generate --dry-run ./...          Preview generation
  hxcmp clean ./...                       Remove all generated files`)
}

func runGenerate(args []string) error {
	var dryRun bool
	var patterns []string

	for _, arg := range args {
		if arg == "--dry-run" {
			dryRun = true
		} else {
			patterns = append(patterns, arg)
		}
	}

	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}

	gen := generator.New(generator.Options{
		DryRun: dryRun,
	})

	return gen.Generate(patterns...)
}

func runClean(args []string) error {
	patterns := args
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}

	gen := generator.New(generator.Options{})
	return gen.Clean(patterns...)
}
