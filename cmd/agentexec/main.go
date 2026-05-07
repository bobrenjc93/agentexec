package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/bobrenjc93/agentexec/internal/client"
	"github.com/bobrenjc93/agentexec/internal/daemon"
	"github.com/bobrenjc93/agentexec/internal/protocol"
)

var version = "dev"

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: agentexec <command> [args...]\n")
		fmt.Fprintf(os.Stderr, "       agentexec --daemon\n")
		fmt.Fprintf(os.Stderr, "       agentexec --version\n")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "--daemon":
		d := daemon.New(protocol.SocketPath())
		if err := d.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "daemon error: %v\n", err)
			os.Exit(1)
		}
	case "--version":
		fmt.Printf("agentexec %s\n", version)
	default:
		command := strings.Join(os.Args[1:], " ")

		cwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "getwd: %v\n", err)
			os.Exit(1)
		}

		socketPath := protocol.SocketPath()

		if err := client.EnsureDaemon(socketPath); err != nil {
			fmt.Fprintf(os.Stderr, "ensure daemon: %v\n", err)
			os.Exit(1)
		}

		exitCode, err := client.Run(socketPath, command, cwd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "run: %v\n", err)
			os.Exit(1)
		}
		os.Exit(exitCode)
	}
}
