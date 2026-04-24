package main

import (
	"flag"
	"fmt"
	"os"
)

var Version = "dev"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "version":
		fmt.Printf("zasca-tunnel %s\n", Version)
	case "install":
		installCmd := flag.NewFlagSet("install", flag.ExitOnError)
		token := installCmd.String("token", "", "tunnel token from ZASCA platform")
		server := installCmd.String("server", "", "gateway server address (e.g., wss://gateway.zasca.com:9000)")
		installCmd.Parse(os.Args[2:])

		if *token == "" || *server == "" {
			fmt.Fprintln(os.Stderr, "token and server are required")
			installCmd.Usage()
			os.Exit(1)
		}

		if err := runInstall(*token, *server); err != nil {
			fmt.Fprintf(os.Stderr, "install failed: %v\n", err)
			os.Exit(1)
		}

	case "run":
		runCmd := flag.NewFlagSet("run", flag.ExitOnError)
		configPath := runCmd.String("config", "C:\\ProgramData\\ZASCA\\tunnel.yaml", "config file path")
		runCmd.Parse(os.Args[2:])

		if err := runService(*configPath); err != nil {
			fmt.Fprintf(os.Stderr, "run failed: %v\n", err)
			os.Exit(1)
		}

	case "uninstall":
		if err := runUninstall(); err != nil {
			fmt.Fprintf(os.Stderr, "uninstall failed: %v\n", err)
			os.Exit(1)
		}

	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("ZASCA Tunnel - Edge Service")
	fmt.Printf("Version: %s\n", Version)
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  version   - Show version")
	fmt.Println("  install   - Install as Windows service")
	fmt.Println("  run       - Run the tunnel service")
	fmt.Println("  uninstall - Uninstall the Windows service")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  zasca-tunnel.exe install -token <TOKEN> -server <WSS_URL>")
	fmt.Println("  zasca-tunnel.exe run -config <PATH>")
	fmt.Println("  zasca-tunnel.exe uninstall")
}
