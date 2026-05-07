# agentexec

A lightweight CLI that delegates command execution to a persistent background daemon. Run commands through a long-lived process without paying shell startup costs on every invocation.

## How it works

```
┌─────────────┐        Unix Socket        ┌─────────────────┐
│  agentexec   │ ──── command + cwd ────▶ │     daemon       │
│  (client)    │ ◀── stdout/stderr/exit ── │  (long-running)  │
└─────────────┘                            └─────────────────┘
```

1. You run `agentexec <command>`.
2. If no daemon is running, the client spawns one in the background automatically.
3. The daemon listens on a Unix socket at `$XDG_RUNTIME_DIR/agentexec-<uid>.sock` (falls back to `/tmp`).
4. The client sends the command and caller's working directory to the daemon.
5. The daemon runs `sh -c <command>` in that directory, streams stdout and stderr back to the client in real time, and returns the exit code.
6. The client exits with the same exit code as the executed command.

The daemon persists across invocations—subsequent calls reuse the same process. This is useful when you want a stable, long-lived execution environment (e.g., warm caches, loaded environment variables, persistent connections) without re-initializing on every command.

## Installation

### From GitHub Releases

Download the latest binary for your platform from the [releases page](https://github.com/bobrenjc93/agentexec/releases):

```bash
# Linux (amd64)
curl -L -o agentexec \
  https://github.com/bobrenjc93/agentexec/releases/latest/download/agentexec-linux-amd64
chmod +x agentexec
mv agentexec ~/.local/bin/
```

```bash
# macOS (Apple Silicon)
curl -L -o agentexec \
  https://github.com/bobrenjc93/agentexec/releases/latest/download/agentexec-darwin-arm64
chmod +x agentexec
mv agentexec /usr/local/bin/
```

### From source

Requires Go 1.22+:

```bash
go install github.com/bobrenjc93/agentexec/cmd/agentexec@latest
```

## Usage

### Run a command

```bash
agentexec echo hello world
```

```bash
agentexec "ls -la | grep go"
```

Arguments are joined into a single shell command, so both of these work:

```bash
# These are equivalent
agentexec git status
agentexec "git status"
```

### Working directory

Commands execute in the directory where `agentexec` was called, not where the daemon was started:

```bash
cd /tmp
agentexec pwd
# /tmp

cd ~/projects
agentexec pwd
# /home/you/projects
```

### Exit codes

The client propagates the exit code from the executed command:

```bash
agentexec false
echo $?
# 1
```

### Start the daemon manually

The daemon starts automatically on first use. To start it explicitly:

```bash
agentexec --daemon
```

### Stop the daemon

Kill the daemon process:

```bash
pkill -f "agentexec --daemon"
```

Or remove the socket to force a new daemon on next invocation:

```bash
rm "${XDG_RUNTIME_DIR:-/tmp}/agentexec-$(id -u).sock"
```

### Version

```bash
agentexec --version
```

## Architecture

```
cmd/agentexec/main.go       Entry point — routes to daemon or client mode
internal/protocol/           Wire protocol (length-prefixed JSON over Unix socket)
internal/daemon/             Daemon: accepts connections, executes commands, streams output
internal/client/             Client: connects to daemon, sends commands, prints output
```

### Protocol

Communication uses length-prefixed JSON messages over a Unix domain socket.

**Request** (client → daemon):
```json
{"command": "ls -la", "cwd": "/home/user/project"}
```

**Response** (daemon → client): a stream of chunks:
```json
{"type": 1, "data": "base64-encoded stdout bytes"}
{"type": 2, "data": "base64-encoded stderr bytes"}
{"type": 3, "code": 0}
```

Chunk types: `1` = stdout, `2` = stderr, `3` = exit (terminates the stream).

### Security

- The Unix socket is created with `0700` permissions (owner-only access).
- The socket path includes the user's UID, preventing collisions between users.
- Commands are executed via `sh -c`, inheriting the daemon's environment.

## Supported platforms

| OS      | Architecture |
|---------|-------------|
| Linux   | amd64, arm64 |
| macOS   | amd64, arm64 |

## License

MIT
