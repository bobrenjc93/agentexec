package daemon

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"sync"

	"github.com/bobrenjc93/agentexec/internal/protocol"
)

type Daemon struct {
	socketPath string
	listener   net.Listener
	wg         sync.WaitGroup
}

func New(socketPath string) *Daemon {
	return &Daemon{socketPath: socketPath}
}

func (d *Daemon) Start() error {
	os.Remove(d.socketPath)

	ln, err := net.Listen("unix", d.socketPath)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	d.listener = ln
	os.Chmod(d.socketPath, 0700)

	fmt.Fprintf(os.Stderr, "agentexec daemon listening on %s (pid %d)\n", d.socketPath, os.Getpid())

	for {
		conn, err := ln.Accept()
		if err != nil {
			return nil
		}
		d.wg.Add(1)
		go func() {
			defer d.wg.Done()
			d.handle(conn)
		}()
	}
}

func (d *Daemon) handle(conn net.Conn) {
	defer conn.Close()

	req, err := protocol.ReadRequest(conn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read request: %v\n", err)
		return
	}

	fmt.Fprintf(os.Stderr, "exec: %s (cwd: %s)\n", req.Command, req.Cwd)

	cmd := exec.Command("sh", "-c", req.Command)
	cmd.Dir = req.Cwd

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		sendError(conn, err)
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		sendError(conn, err)
		return
	}

	if err := cmd.Start(); err != nil {
		sendError(conn, err)
		return
	}

	var streamWg sync.WaitGroup
	streamWg.Add(2)

	go func() {
		defer streamWg.Done()
		buf := make([]byte, 4096)
		for {
			n, err := stdout.Read(buf)
			if n > 0 {
				chunk := &protocol.Chunk{Type: protocol.ChunkStdout, Data: buf[:n]}
				if werr := protocol.WriteChunk(conn, chunk); werr != nil {
					return
				}
			}
			if err != nil {
				return
			}
		}
	}()

	go func() {
		defer streamWg.Done()
		buf := make([]byte, 4096)
		for {
			n, err := stderr.Read(buf)
			if n > 0 {
				chunk := &protocol.Chunk{Type: protocol.ChunkStderr, Data: buf[:n]}
				if werr := protocol.WriteChunk(conn, chunk); werr != nil {
					return
				}
			}
			if err != nil {
				return
			}
		}
	}()

	streamWg.Wait()

	exitCode := 0
	if err := cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}

	protocol.WriteChunk(conn, &protocol.Chunk{Type: protocol.ChunkExit, Code: exitCode})
}

func sendError(conn net.Conn, err error) {
	msg := fmt.Sprintf("agentexec daemon error: %v\n", err)
	protocol.WriteChunk(conn, &protocol.Chunk{Type: protocol.ChunkStderr, Data: []byte(msg)})
	protocol.WriteChunk(conn, &protocol.Chunk{Type: protocol.ChunkExit, Code: 1})
}
