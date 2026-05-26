package main

import (
	"flag"
	"fmt"
	"os"
)

const Version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		printHelp()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "acp":
		handleACP()
	case "start":
		handleStart()
	case "stop":
		handleStop()
	case "status":
		handleStatus()
	case "version", "--version", "-v":
		handleVersion()
	case "help", "--help", "-h":
		printHelp()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printHelp()
		os.Exit(1)
	}
}

func handleACP() {
	// Start ACP stdio server — reads JSON-RPC from stdin, writes to stdout
	runACPServer()
}

func handleStart() {
	startCmd := flag.NewFlagSet("start", flag.ExitOnError)
	daemon := startCmd.Bool("daemon", false, "Run as daemon (background)")
	startCmd.Parse(os.Args[2:])

	if *daemon {
		startDaemon()
	} else {
		fmt.Fprintln(os.Stderr, "Use --daemon to start in background, or use 'acp' subcommand directly")
		os.Exit(1)
	}
}

func handleStop() {
	stopDaemon()
}

func handleStatus() {
	checkDaemonStatus()
}

func handleVersion() {
	fmt.Printf("agy-acp-bridge v%s\n", Version)
}

func printHelp() {
	fmt.Println("agy-acp-bridge - ACP stdio bridge for agy (Antigravity CLI)")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  agy-acp-bridge <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  acp         Start ACP stdio server (launch this as your ACP agent)")
	fmt.Println("  start       Start bridge as background daemon")
	fmt.Println("  stop        Stop the running daemon")
	fmt.Println("  status      Show daemon status")
	fmt.Println("  version     Show version information")
	fmt.Println("  help        Show this help message")
	fmt.Println()
	fmt.Println("Start Options:")
	fmt.Println("  --daemon    Run in background (required for start)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  agy-acp-bridge acp                   # used by ACP clients")
	fmt.Println("  agy-acp-bridge start --daemon         # background daemon")
	fmt.Println("  agy-acp-bridge stop                   # kill daemon")
	fmt.Println("  agy-acp-bridge status                 # check daemon")
	fmt.Println()
	fmt.Println("ACP Client config example (acpc/acp-ui):")
	fmt.Println(`  { "command": "agy-acp-bridge", "args": ["acp"] }`)
}
