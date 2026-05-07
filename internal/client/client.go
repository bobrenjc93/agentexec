package client

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"time"

	"github.com/bobrenjc93/agentexec/internal/protocol"
)

func EnsureDaemon(socketPath string) error {
	conn, err := net.DialTimeout("unix", socketPath, 500*time.Millisecond)
	if err == nil {
		conn.Close()
		return nil
	}

	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable: %w", err)
	}

	cmd := exec.Command(self, "--daemon")
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.SysProcAttr = daemonSysProcAttr()
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start daemon: %w", err)
	}
	cmd.Process.Release()

	for i := 0; i < 50; i++ {
		time.Sleep(50 * time.Millisecond)
		conn, err := net.DialTimeout("unix", socketPath, 200*time.Millisecond)
		if err == nil {
			conn.Close()
			return nil
		}
	}

	return fmt.Errorf("daemon did not start within timeout")
}

func Run(socketPath string, command string, cwd string) (int, error) {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return 1, fmt.Errorf("connect to daemon: %w", err)
	}
	defer conn.Close()

	req := &protocol.Request{Command: command, Cwd: cwd}
	if err := protocol.WriteRequest(conn, req); err != nil {
		return 1, fmt.Errorf("send request: %w", err)
	}

	for {
		chunk, err := protocol.ReadChunk(conn)
		if err != nil {
			return 1, fmt.Errorf("read chunk: %w", err)
		}

		switch chunk.Type {
		case protocol.ChunkStdout:
			os.Stdout.Write(chunk.Data)
		case protocol.ChunkStderr:
			os.Stderr.Write(chunk.Data)
		case protocol.ChunkExit:
			return chunk.Code, nil
		}
	}
}
