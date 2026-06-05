# agy-acp-bridge Development Skill

Learnings, caveats, gotchas, and pitfalls from developing and testing the agy-acp-bridge (ACP stdio bridge for agy Antigravity CLI).

## Project Overview

**agy-acp-bridge** is a Go binary that wraps the `agy` CLI using a pseudo-TTY to provide an ACP (Agent Client Protocol) stdio interface. It enables agy to work with any ACP-compatible client without modifications.

## Architecture

```
ACP Client → agy-acp-bridge (JSON-RPC 2.0 stdio) → agy (via pty)
```

**Key Components**:
- `main.go` - CLI entrypoint
- `acp.go` - ACP JSON-RPC server implementation
- `bridge.go` - agy pty runner with model selection and fallback
- `session.go` - Single-session store with model tracking

## Critical Learnings

### 1. ACP Message Format Compatibility

**Pitfall**: Different ACP clients send different message formats.

**Two formats exist**:
- **Standard ACP**: `{"messages": [{"role": "user", "content": [{"type": "text", "text": "prompt"}]}]}`
- **Simplified format**: `{"prompt": [{"type": "text", "text": "prompt"}]}`

**Solution**: Support both formats in `session/prompt` handler:
```go
type sessionPromptParams struct {
    SessionID string         `json:"sessionId"`
    Messages  []promptMessage `json:"messages,omitempty"`
    Prompt    []contentBlock `json:"prompt,omitempty"`  // Support simplified format
}
```

**Priority**: Try `prompt` first, fall back to `messages`.

### 2. Thinking Models and --print Mode

**Pitfall**: Some "Thinking" models (Claude Opus 4.6, Claude Sonnet 4.6) return empty output when using agy's `--print` mode.

**Symptoms**:
- Empty response from model
- No error, just blank output
- Model appears to work but produces no text

**Solution**: Implement automatic fallback to `--prompt-interactive`:
```go
func RunPrompt(sess *Session, prompt string) RunResult {
    // First attempt with --print mode
    result := runPromptWithMode(sess, prompt, false)
    
    // If empty output and Thinking model, retry with --prompt-interactive
    if result.Output == "" && strings.Contains(sess.Model, "Thinking") {
        result = runPromptWithMode(sess, prompt, true)
    }
    
    return result
}
```

**Model detection**: Check if model name contains "Thinking" substring.

### 3. Session Continuity

**Pitfall**: Each pipe command creates a new bridge process, losing session context.

**Symptoms**:
- Session ID changes on each prompt
- Conversation history lost
- `--continue` flag not working

**Solution**: Use a persistent Go test program or single-process client (acpx, universe-agent-acp-client) to maintain session continuity.

**Testing approach**: Create a Go program that:
1. Starts bridge process once
2. Sends initialize
3. Sends session/new
4. Sends multiple session/prompt requests
5. Closes bridge

### 4. Model Selection Implementation

**Pitfall**: agy's `--model` flag must be placed before other flags.

**Wrong order**:
```go
args := []string{"--print", "--model", "claude", prompt}  // Fails
```

**Correct order**:
```go
args := []string{"--model", "claude", "--print", prompt}  // Works
```

**Implementation**: Prepend `--model` flag before other flags in `buildArgs()`.

### 5. ACP Client Testing

**Pitfall**: Shell-based testing causes session loss.

**Working clients**:
- `acpx` - Headless CLI client (recommended for testing)
- `universe-agent-acp-client` - Simple debugging client

**Testing workflow**:
```bash
# Install acpx
npm install -g acpx

# Configure in ~/.acpx/config.json
{
  "defaultAgent": "agy",
  "agents": {
    "agy": {
      "command": "/path/to/agy-acp-bridge",
      "args": ["acp"]
    }
  }
}

# Test
acpx sessions new
acpx "What is 2+2?"
```

### 6. Zed Editor Configuration

**Pitfall**: Zed requires specific configuration format in `settings.json`.

**Configuration**:
```json
{
  "agent_servers": {
    "agy": {
      "type": "custom",
      "command": "/path/to/agy-acp-bridge",
      "args": ["acp"],
      "env": {},
      "default_model": "Claude Sonnet 4.6 (Thinking)"
    }
  }
}
```

**Key points**:
- Use `"type": "custom"` for custom agents
- Binary must be executable
- `default_model` is optional but recommended
- Path must be absolute

## Common Pitfalls

### 1. Empty Responses from Models

**Cause**: 
- Thinking models in --print mode
- Model not responding
- agy internal error

**Debugging**:
1. Check if model name contains "Thinking"
2. Test with `agy --print "test"` directly
3. Verify model name is exact (case-sensitive)
4. Check agy logs for errors

### 2. Session/Load Not Supported

**Pitfall**: agy doesn't expose conversation IDs, so `session/load` cannot be implemented.

**Solution**: Advertise as `false` in capabilities:
```go
writeResult(req.ID, agentInfo{
    AgentCapabilities: map[string]bool{
        "loadSession": false,
    },
})
```

### 3. Cancellation is Best-Effort

**Pitfall**: agy's `--print` mode is synchronous, so `session/cancel` cannot truly cancel.

**Solution**: Implement best-effort cancellation by killing the pty process, but document as best-effort.

### 4. Tool Approval

**Pitfall**: agy requires tool approval prompts in interactive mode.

**Solution**: Use `--dangerously-skip-permissions` flag to auto-approve all tool calls (document security implications).

## Testing Best Practices

### 1. Smoke Test All Models

Create a test program that:
1. Tests each model with a simple prompt ("2+2")
2. Measures response time
3. Verifies output is not empty
4. Tests fallback mechanism for Thinking models

**Expected results**:
- All 8 models should respond
- Average response time: 6-10 seconds
- Thinking models should use fallback

### 2. Test ACP Client Compatibility

Test with multiple clients:
- acpx (headless CLI)
- universe-agent-acp-client (debugging)
- Zed editor (if applicable)

**Verify**:
- Session creation works
- Prompt/response cycle works
- Model selection works
- Session continuity works

### 3. Test Error Conditions

Test error scenarios:
- Invalid session ID
- Empty prompt
- Invalid model name
- Missing required parameters

**Expected behavior**: Proper JSON-RPC error responses with meaningful error codes.

## Performance Considerations

### 1. Bridge Overhead

**Measurement**: ~1-2ms for initialize/session/new operations

**Optimization**: Minimal overhead is acceptable; focus on agy's model inference time (dominant factor).

### 2. Binary Size

**Current**: 2.3 MB (optimized Go build)

**Optimization**: Use Go build flags:
```bash
go build -ldflags="-s -w" -o agy-acp-bridge
```

### 3. Memory Usage

**Current**: Minimal (single-session store)

**Optimization**: Not a concern for single-session use case.

## Security Considerations

### 1. Tool Permissions

**Risk**: `--dangerously-skip-permissions` auto-approves all tool calls.

**Mitigation**: Document this clearly in README. Users must trust the environment.

### 2. Session Isolation

**Risk**: Single-session store means only one session at a time.

**Mitigation**: Document limitation. Use daemon mode for multi-session support.

### 3. Model Access

**Risk**: agy requires API keys for some models.

**Mitigation**: agy handles authentication; bridge doesn't see credentials.

## Documentation Best Practices

### 1. README Structure

Include:
- Project overview
- Architecture diagram
- Installation instructions
- ACP client configuration examples
- Model selection guide
- Troubleshooting section
- Performance benchmarks

### 2. AGENTS.md

Use for:
- Memory system documentation
- Agent workflow guidelines
- Relevant memory tags
- Cross-session continuity patterns

### 3. Code Comments

Keep comments minimal but explain:
- Why certain design decisions were made
- Workarounds for known issues
- Model-specific behavior
- ACP protocol quirks

## Deployment Patterns

### 1. Daemon Mode

For production use, run as daemon:
```bash
agy-acp-bridge start
agy-acp-bridge status
agy-acp-bridge stop
```

### 2. Direct Invocation

For testing:
```bash
./agy-acp-bridge acp
```

### 3. ACP Client Integration

Configure in client settings:
```json
{
  "command": "/path/to/agy-acp-bridge",
  "args": ["acp"]
}
```

## Troubleshooting Guide

### Issue: "No text content found in messages"

**Cause**: Client sending unsupported message format.

**Solution**: Ensure bridge supports both `messages` and `prompt` formats.

### Issue: Empty response from model

**Cause**: Thinking model in --print mode.

**Solution**: Verify fallback mechanism is implemented and working.

### Issue: Session ID not found

**Cause**: New bridge process started for each prompt.

**Solution**: Use persistent client (acpx) or single-process test program.

### Issue: Model not recognized

**Cause**: Model name mismatch or typo.

**Solution**: Verify exact model name with `agy models`.

## Future Improvements

### 1. Multi-Session Support

Implement session store with multiple concurrent sessions.

### 2. MCP Integration

Add Model Context Protocol support for tool integration.

### 3. Streaming Support

Implement streaming responses instead of single-chunk output.

### 4. Better Cancellation

Implement true cancellation if agy adds support.

## References

- ACP Protocol: https://agentclientprotocol.com
- agy CLI: Antigravity CLI documentation
- creack/pty: https://github.com/creack/pty
- acpx: https://github.com/yourusername/acpx

## Memory System Tags

Use these tags for related memories:
- `agy-acp-bridge` - All project-specific memories
- `acp` - Agent Client Protocol
- `models` - Model selection and behavior
- `testing` - Test results and procedures
- `pitfalls` - Common issues and solutions
- `configuration` - Client setup and configuration