package protocol

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type Request struct {
	Command string `json:"command"`
	Cwd     string `json:"cwd"`
}

type ChunkType byte

const (
	ChunkStdout ChunkType = 1
	ChunkStderr ChunkType = 2
	ChunkExit   ChunkType = 3
)

type Chunk struct {
	Type ChunkType `json:"type"`
	Data []byte    `json:"data,omitempty"`
	Code int       `json:"code,omitempty"`
}

func SocketPath() string {
	dir := os.Getenv("XDG_RUNTIME_DIR")
	if dir == "" {
		dir = os.TempDir()
	}
	return filepath.Join(dir, fmt.Sprintf("agentexec-%d.sock", os.Getuid()))
}

func WriteMsg(w io.Writer, data []byte) error {
	length := uint32(len(data))
	if err := binary.Write(w, binary.LittleEndian, length); err != nil {
		return err
	}
	_, err := w.Write(data)
	return err
}

func ReadMsg(r io.Reader) ([]byte, error) {
	var length uint32
	if err := binary.Read(r, binary.LittleEndian, &length); err != nil {
		return nil, err
	}
	buf := make([]byte, length)
	_, err := io.ReadFull(r, buf)
	return buf, err
}

func WriteRequest(w io.Writer, req *Request) error {
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}
	return WriteMsg(w, data)
}

func ReadRequest(r io.Reader) (*Request, error) {
	data, err := ReadMsg(r)
	if err != nil {
		return nil, err
	}
	var req Request
	err = json.Unmarshal(data, &req)
	return &req, err
}

func WriteChunk(w io.Writer, chunk *Chunk) error {
	data, err := json.Marshal(chunk)
	if err != nil {
		return err
	}
	return WriteMsg(w, data)
}

func ReadChunk(r io.Reader) (*Chunk, error) {
	data, err := ReadMsg(r)
	if err != nil {
		return nil, err
	}
	var chunk Chunk
	err = json.Unmarshal(data, &chunk)
	return &chunk, err
}
