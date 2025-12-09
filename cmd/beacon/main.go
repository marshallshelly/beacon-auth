package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/marshallshelly/beacon-auth/cmd/beacon/schema"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "init":
		handleInit()
	case "generate":
		handleGenerate(os.Args[2:])
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Print(`BeaconAuth CLI

Usage:
  beacon <command> [flags]

Commands:
  init      Initialize BeaconAuth configuration
  generate  Generate SQL schema for your database

Generate Flags:
  --adapter   Database adapter (postgres, mysql, sqlite, mssql) [required]
  --plugins   Comma-separated list of plugins (e.g., twofa,oauth)
  --id-type   ID generation strategy (string, uuid, serial) [default: string]

Examples:
  beacon generate --adapter postgres --plugins twofa --id-type uuid
  beacon generate --adapter sqlite --id-type string
`)
}

func handleInit() {
	fmt.Println("Initializing BeaconAuth...")
	// For now, just print a message. Future: create a config file.
	fmt.Println("Configuration file creation is not yet implemented. Use command line flags for now.")
}

func handleGenerate(args []string) {
	generateCmd := flag.NewFlagSet("generate", flag.ExitOnError)
	adapter := generateCmd.String("adapter", "", "Database adapter (postgres, mysql, sqlite, mssql)")
	plugins := generateCmd.String("plugins", "", "Comma-separated list of plugins")
	idType := generateCmd.String("id-type", "string", "ID generation strategy (string, uuid, serial)")

	if err := generateCmd.Parse(args); err != nil {
		fmt.Printf("Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	if *adapter == "" {
		fmt.Println("Error: --adapter is required")
		generateCmd.PrintDefaults()
		os.Exit(1)
	}

	validAdapters := map[string]bool{"postgres": true, "mysql": true, "sqlite": true, "mssql": true}
	if !validAdapters[*adapter] {
		fmt.Printf("Error: invalid adapter '%s'. Must be one of: postgres, mysql, sqlite, mssql\n", *adapter)
		os.Exit(1)
	}

	validIDTypes := map[string]bool{"string": true, "uuid": true, "serial": true}
	if !validIDTypes[*idType] {
		fmt.Printf("Error: invalid id-type '%s'. Must be one of: string, uuid, serial\n", *idType)
		os.Exit(1)
	}

	pluginList := []string{}
	if *plugins != "" {
		pluginList = strings.Split(*plugins, ",")
		for i := range pluginList {
			pluginList[i] = strings.TrimSpace(pluginList[i])
		}
	}

	cfg := &schema.Config{
		Adapter: *adapter,
		Plugins: pluginList,
		IDType:  *idType,
	}

	sql, err := schema.GenerateSQL(cfg)
	if err != nil {
		fmt.Printf("Error generating SQL: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(sql)
}
