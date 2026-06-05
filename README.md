# agy-acp-bridge

> **⚠️ UNOFFICIAL IMPLEMENTATION — USE AT YOUR OWN RISK**
>
> This is an unofficial ACP implementation that wraps agy via a pseudo-TTY. It is not endorsed by or affiliated with Google or the Antigravity CLI team. Use only in trusted environments. The bridge may break if agy's behavior changes in future versions.
>
> **How it works**: This bridge uses the official `agy` binary with standard CLI flags (`--print`, `--continue`, `--dangerously-skip-permissions`). No reverse engineering, exploits, or protocol manipulation. It simply provides an ACP stdio interface to agy's existing print mode.

ACP (Agent Client Protocol) stdio bridge for [agy](https://github.com/google-antigravity/antigravity-cli) (Antigravity CLI).

**Problem**: agy does not support ACP natively ([Issue #31](https://github.com/google-antigravity/antigravity-cli/issues/31)) and silently drops stdout in non-TTY contexts ([Issue #76](https://github.com/google-antigravity/antigravity-cli/issues/76)).

**Solution**: `agy-acp-bridge` wraps `agy --print` via a pseudo-TTY (pty) and exposes it as an ACP-compliant stdio server, enabling use with any ACP client (acpc, acp-ui, VS Code extensions, etc.).

---

## Features

- ✅ **ACP stdio server** — JSON-RPC 2.0 over stdin/stdout
- ✅ **Pseudo-TTY wrapper** — Solves agy's non-TTY output issue
- ✅ **Single-session support** — Conversation continuity via `agy --continue`
- ✅ **Daemon management** — Start/stop/status with PID file
- ✅ **No external dependencies** — Only `creack/pty` (well-maintained, 3.4k stars)
- ✅ **Binary size** — 2.3 MB optimized build

---

## Architecture

```
ACP Client (acpc, acp-ui, VS Code)
       │  JSON-RPC 2.0 over stdin/stdout
       ▼
agy-acp-bridge (Go binary)
       │  exec: agy --print --dangerously-skip-permissions "<prompt>"
       ▼
agy (via pty for TTY emulation)
```

### Key Design Decisions

| Decision | Rationale |
|---|---|
| **pty wrapper** | Solves Issue #76 — agy sees a real terminal and outputs correctly |
| **Single chunk per response** | agy `--print` is not streaming; emits one `agent_message_chunk` per response |
| **`--continue` for continuity** | Resumes last conversation from second prompt onward |
| **Single session only** | agy `--continue` only supports one active conversation per process |
| **Automatic fallback for Thinking models** | Some "Thinking" models don't work with `--print`; automatically retries with `--prompt-interactive` |

---

## Installation

### Build from source

```bash
cd ~/ai/agy-acp-bridge
./build.sh
```

Binary output: `agy-acp-bridge` (2.3 MB)

---

## Usage

### CLI Commands

```bash
# Show help
agy-acp-bridge help

# Show version
agy-acp-bridge version

# Start ACP stdio server (for ACP clients)
agy-acp-bridge acp

# Start as background daemon
agy-acp-bridge start --daemon

# Stop daemon
agy-acp-bridge stop

# Check daemon status
agy-acp-bridge status
```

### ACP Client Configuration

Configure your ACP client (acpc, acp-ui, VS Code, etc.) to launch:

```json
{
  "command": "/home/jarancibia/ai/agy-acp-bridge/agy-acp-bridge",
  "args": ["acp"]
}
```

#### acpc example

```bash
acpc models agy
acpc prompt "list files in current directory" --agent agy
```

#### acp-ui / VS Code example

Add to agent configuration:

```json
{
  "agy": {
    "command": "/path/to/agy-acp-bridge",
    "args": ["acp"],
    "env": {}
  }
}
```

#### acpx (Headless ACP Client)

Install acpx:
```bash
npm install -g acpx
```

Configure acpx for agy-acp-bridge (`~/.acpx/config.json`):
```json
{
  "defaultAgent": "agy",
  "defaultPermissions": "approve-all",
  "nonInteractivePermissions": "deny",
  "authPolicy": "skip",
  "agents": {
    "agy": {
      "command": "/path/to/agy-acp-bridge/agy-acp-bridge",
      "args": ["acp"]
    }
  }
}
```

Usage:
```bash
acpx sessions new
acpx "what is 2+2?"
```

#### universe-agent-acp-client

Install:
```bash
npm install -g @universe-agent/acp-client
```

Usage:
```bash
universe-agent-acp-client --command "/path/to/agy-acp-bridge/agy-acp-bridge" --args "acp" "what is 2+2?"
```

#### Zed Editor

Zed has native ACP support. Add to `~/.config/zed/settings.json`:

```json
{
  "agent_servers": {
    "agy": {
      "type": "custom",
      "command": "/path/to/agy-acp-bridge/agy-acp-bridge",
      "args": ["acp"],
      "env": {}
    }
  }
}
```

Usage:
1. Open Zed Agent Panel: `Cmd-?` (macOS) or `Ctrl-?` (Windows/Linux)
2. Click `+` to create new thread
3. Select `agy` from agent dropdown

### Model Selection

The bridge supports custom model selection via the `session/new` `model` parameter. Available models can be listed with `agy models`:

```bash
agy models
```

Example models:
- `Claude Sonnet 4.6 (Thinking)`
- `Claude Opus 4.6 (Thinking)`
- `Gemini 3.5 Flash (Medium)`
- `GPT-OSS 120B (Medium)`

To use a specific model, include it in the `session/new` request:

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "session/new",
  "params": {
    "cwd": "/path/to/project",
    "model": "Claude Sonnet 4.6 (Thinking)"
  }
}
```

If no model is specified, agy uses its default model.

---

## ACP Methods Implemented

| Method | Supported | Notes |
|---|---|---|
| `initialize` | ✅ | Returns capabilities, no auth required |
| `session/new` | ✅ | Creates new session, stores `cwd` and optional `model` |
| `session/load` | ❌ | Not supported (advertised as `false`) |
| `session/prompt` | ✅ | Runs `agy --print` via pty, streams `agent_message_chunk` |
| `session/cancel` | ⚠️ | Best-effort (agy `--print` is synchronous) |

---

## Performance Stats

Smoke tests on Ubuntu Linux, agy v1.0.2:

| Test | Total Time | agy Time | Output Size |
|---|---|---|---|
| Simple echo | 5.37s | 5.37s | 9 chars |
| Short answer | 4.43s | 4.43s | 86 chars |
| Code generation | 14.46s | 14.46s | 1449 chars |
| File context | 9.88s | 9.88s | 755 chars |
| Explanation | 4.12s | 4.11s | 223 chars |
| **AVERAGE** | **7.65s** | **7.65s** | **504 chars** |

**Notes**:
- Bridge overhead is negligible (~1-2ms for `initialize` and `session/new`)
- Response time is dominated by agy itself (model inference)
- No streaming — full response emitted in single `agent_message_chunk`

---

## Project Structure

```
agy-acp-bridge/
├── main.go      # CLI entrypoint (94 LOC)
├── acp.go       # ACP JSON-RPC stdio server (303 LOC)
├── bridge.go    # agy pty runner (133 LOC)
├── session.go   # Single-session store (65 LOC)
├── daemon.go    # PID-file daemon management (130 LOC)
├── build.sh     # Optimized Go build (8 LOC)
├── go.mod       # Go module definition
└── README.md    # This file
```

**Total**: 733 LOC (excluding README)

---

## Limitations & Known Issues

1. **No streaming** — agy `--print` returns full response at once; cannot emit token-by-token
2. **Single session** — Only one active conversation per bridge process
3. **Tool approval** — Uses `--dangerously-skip-permissions` (auto-approves all tool calls)
4. **Session loading** — Does not support `session/load` (agy conversation IDs not exposed)
5. **Cancellation** — `session/cancel` is best-effort (agy `--print` is synchronous)
6. **Thinking models** — Some "Thinking" models (e.g., Claude Opus 4.6) don't work with `--print` mode; bridge automatically retries with `--prompt-interactive` for these models

---

## Security Considerations

- ⚠️ **UNOFFICIAL IMPLEMENTATION** — This is not an official ACP implementation. May break if agy changes.
- ⚠️ **Auto-approves tool calls** — `--dangerously-skip-permissions` bypasses agy's permission prompts
- ⚠️ **No authentication** — ACP `initialize` returns empty `authMethods`
- ✅ **No reverse engineering** — Uses official agy binary with standard CLI flags only
- ✅ **No protocol manipulation** — Simply wraps agy's existing `--print` mode with ACP stdio interface
- ✅ **No credential leakage** — pty wrapper prevents agy from reading user input
- ✅ **Process isolation** — Each bridge process is isolated with its own pty

**Recommendation**: Use in trusted environments only. Do not expose to untrusted networks. This is a temporary workaround until agy adds official ACP support.

---

## Requirements

- Go 1.21+
- agy (Antigravity CLI) v1.0.0+ installed in PATH
- Linux/macOS (pty support)

---

## Development

### Build

```bash
./build.sh
```

### Run tests

```bash
go test -v -timeout 120s
```

### Dependencies

- `github.com/creack/pty` — Pseudo-terminal operations for Linux/macOS

---

## Troubleshooting

### "pty start failed"

Ensure `agy` is installed and accessible in PATH:

```bash
which agy
agy --version
```

### "read error: input/output error"

Normal pty termination signal when agy exits. Not an error.

### "Unknown sessionId"

Each bridge process has its own session store. Use a single long-running process for conversation continuity.

---

## Alternatives

- **Wait for official ACP support** — agy Issue #31 is open and actively discussed (recommended)
- **professional-ALFIE fork** — Has `--json` support but unofficial
- **acpc + other agents** — Use acpc with ACP-compatible agents (Claude Code, OpenCode, etc.)

**Note**: This bridge is a temporary workaround. Consider switching to official ACP support when available.

---

## License

MIT

---

## Contributing

Contributions welcome! Areas for improvement:
- Multi-session support (if agy exposes conversation IDs)
- Streaming output (if agy adds ACP mode)
- Better cancellation support (if agy adds async mode)

---

## Acknowledgments

- [Antigravity CLI](https://github.com/google-antigravity/antigravity-cli) — The underlying AI agent
- [Agent Client Protocol](https://agentclientprotocol.com/) — The standard for agent-client communication
- [creack/pty](https://github.com/creack/pty) — Pseudo-terminal operations library