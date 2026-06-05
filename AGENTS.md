# Agent Memory System for agy-acp-bridge

This document describes the memory system used for AI agent work on the agy-acp-bridge project.

## Memory System: mem

We use the `mem` CLI tool for storing and retrieving memories with full-text search. Data is stored locally in `~/.mem/mem.db`.

### Installation

```bash
# Clone the repository
git clone https://github.com/runablehq/memory.git
cd memory

# Build the CLI (requires bun)
bun build --compile src/cli.ts --outfile dist/mem

# Install to PATH
cp dist/mem ~/.local/bin/mem
```

### Usage

#### Recall (search, list, get)
```bash
mem                             # list recent memories
mem "deploy"                    # full-text search
mem "database" --tag db         # search filtered by tag
mem <id>                        # get full content by ID
mem --tag prefs                 # list filtered by tag
mem --full                      # show full content for all
```

#### Remember
```bash
mem + "user prefers dark mode" --tag prefs
mem + "deploy: bun build --compile" --tag deploy
echo "long content" | mem + --tag notes
```

#### Forget
```bash
mem - <id>                      # delete one memory
mem - id1 id2 id3               # delete multiple
```

## Stored Memories for agy-acp-bridge

### ACP Client Compatibility
- **ID**: `FupNjHLA4GF8`
- **Tags**: `acp`, `clients`, `agy-acp-bridge`
- **Content**: ACP clients that work with agy-acp-bridge: 1) acpx (headless CLI client) - install via 'npm install -g acpx', configure in ~/.acpx/config.json with agent command and args, usage: 'acpx sessions new' then 'acpx "prompt"' 2) universe-agent-acp-client - install via 'npm install -g @universe-agent/acp-client', usage: 'universe-agent-acp-client --command /path/to/agy-acp-bridge --args acp "prompt"' 3) Bridge updated to support both standard ACP format (messages array with role/content) and simplified format (prompt array) for compatibility with different clients

### Model Customization & Claude Opus Fix
- **ID**: `vbyihWU1KhC_`
- **Tags**: `agy-acp-bridge`, `models`, `fix`
- **Content**: agy-acp-bridge model customization: Added 'model' parameter to session/new request to select agy models. Available models: Gemini 3.5 Flash (Medium/High/Low), Gemini 3.1 Pro (High/Low), Claude Sonnet 4.6 (Thinking), Claude Opus 4.6 (Thinking), GPT-OSS 120B (Medium). Usage: include model in session/new params. Claude Opus 4.6 fix: auto-fallback from --print to --prompt-interactive for Thinking models that return empty output in print mode. All 8 models tested successfully with 2+2 question.

## Memory Best Practices

1. **Tag consistently** — Use lowercase, descriptive tags like `prefs`, `api`, `deploy`, `db`
2. **Search before asking** — Check if you've stored relevant information before asking the user
3. **Store decisions** — When making architectural or design decisions, store the reasoning
4. **Keep memories atomic** — One concept per memory for better searchability

## Agent Workflow for This Project

When working on agy-acp-bridge:

1. **Before starting work**: Search existing memories
   ```bash
   mem "agy-acp-bridge"
   mem "acp" --tag clients
   ```

2. **During development**: Store important findings
   ```bash
   mem + "discovered that agy --print mode fails for Claude Opus 4.6" --tag bug,agy
   ```

3. **After testing**: Store test results
   ```bash
   mem + "smoke test results: all 8 models passed, average response time 7.4s" --tag testing,performance
   ```

4. **When implementing features**: Store architectural decisions
   ```bash
   mem + "chose to support both ACP message formats for broader client compatibility" --tag architecture,acp
   ```

## Relevant Memory Tags

- `agy-acp-bridge` - All project-specific memories
- `acp` - Agent Client Protocol related
- `clients` - ACP client compatibility
- `models` - Model selection and behavior
- `fix` - Bug fixes and workarounds
- `testing` - Test results and procedures
- `architecture` - Design decisions

## Accessing Memories

To recall stored information about this project:
```bash
# List all agy-acp-bridge memories
mem --tag agy-acp-bridge

# Search for ACP client information
mem "acp client" --tag clients

# Get specific memory by ID
mem FupNjHLA4GF8

# Full-text search
mem "Claude Opus"
```

## Integration with Development Workflow

The mem system helps maintain context across sessions:

- **Cross-session continuity**: Important decisions and findings persist between different agent sessions
- **Quick recall**: No need to re-discover known issues or solutions
- **Knowledge sharing**: Different agents can access the same knowledge base
- **Decision tracking**: Architectural decisions and their rationale are preserved

## Future Memory Topics

Consider storing memories for:
- Performance benchmarks and optimization results
- User feedback and feature requests
- Deployment procedures and configurations
- Integration patterns with other tools
- Troubleshooting guides and common issues
