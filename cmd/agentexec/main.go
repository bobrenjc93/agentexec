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

const usage = `agentexec - execute commands via a persistent background daemon

Usage:
  agentexec <command> [args...]
  agentexec [flags]

Commands are forwarded to a long-running daemon process over a Unix
socket. The daemon starts automatically on first use. Commands run
via "sh -c" in the caller's working directory.

Flags:
  -h, --help       Show this help message
  -v, --version    Print version
      --daemon     Start the daemon in the foreground
      --stop       Stop the running daemon

Socket: $XDG_RUNTIME_DIR/agentexec-<uid>.sock (or /tmp)

Examples:
  agentexec echo hello
  agentexec "ls -la | grep go"
  agentexec git status`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(1)
	}

	switch os.Args[1] {
	case "--daemon":
		d := daemon.New(protocol.SocketPath())
		if err := d.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "daemon error: %v\n", err)
			os.Exit(1)
		}
	case "-h", "--help":
		fmt.Println(usage)
	case "-v", "--version":
		fmt.Printf("agentexec %s\n", version)
	case "--stop":
		socketPath := protocol.SocketPath()
		if err := os.Remove(socketPath); err != nil {
			fmt.Fprintf(os.Stderr, "stop: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "removed %s\n", socketPath)
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
