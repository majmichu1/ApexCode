# ApexCode Plugin for OpenCode

**MemPalace memory · Repomap · Sentinel analysis · LM Studio auto-discovery · Multi-agent swarms**

ApexCode transforms OpenCode into the most capable AI coding agent by adding:

- **MemPalace Memory** — Cross-session memory using the method of loci. The agent remembers your codebase patterns, past decisions, and preferences.
- **Repomap + PageRank** — Repository mapping that identifies the most relevant files for any task using network analysis of code dependencies.
- **Sentinel Proactive Analysis** — Continuous code scanning for TODOs, security issues, long functions, deep nesting, and hardcoded values.
- **LM Studio Auto-Discovery** — Automatically discovers ALL models from your local LM Studio instance. No manual configuration needed.
- **Multi-Agent Swarms** — Spawn 6 specialized agents (Planner, Architect, Coder, Reviewer, Tester, Documenter) to collaborate on complex tasks.

## Installation

### Option 1: Install from GitHub (Recommended)

```bash
opencode plugin github:YOUR_USERNAME/apexcode
```

### Option 2: Install from a local path

```bash
opencode plugin /path/to/apexcode-plugin
```

### Option 3: Install from npm (when published)

```bash
opencode plugin apexcode-plugin
```

## Go Backend

The plugin auto-starts the Go backend (`apex --serve`) when OpenCode launches. The `apex` binary must be in your PATH:

```bash
# Build from source
git clone https://github.com/YOUR_USERNAME/apexcode
cd apexcode
go build -o apex ./cmd/apex
cp apex ~/.local/bin/apex   # or any directory in your PATH
```

If the binary is not found, the plugin still works — you just won't get MemPalace/Repomap enhancements.

## Features

### MemPalace + Repomap (Automatic)

Every time you send a prompt, the plugin fetches relevant context from the Go backend and injects it into the system prompt. You'll see:

```
<apexcode_context>
You are augmented with ApexCode's codebase intelligence.
Token savings from context injection: 60-80% vs full file contents
<repository_map>
... PageRank-ranked files and their relationships ...
</repository_map>
<memory_context>
... relevant knowledge from previous sessions ...
</memory_context>
</apexcode_context>
```

### LM Studio Auto-Discovery

Connect to LM Studio via `/connect` and all your local models appear automatically. No manual configuration. No stale cache files.

### Sentinel Analysis

View proactive code analysis results in the sidebar. The plugin scans for:
- TODO/FIXME/HACK comments
- Long functions (>50 lines)
- Deep nesting (>3 levels)
- Empty catch blocks
- Hardcoded URLs, passwords, API keys
- Security issues (SQL injection, eval, innerHTML)

### Multi-Agent Swarms

Use the `/swarm` command or the `apexcode_swarm` tool to execute tasks across specialized agents:

```
/swarm

# Or via the tool
apexcode_swarm task="Refactor the auth module" agents=["planner", "architect", "coder", "reviewer"] mode="parallel"
```

Available agents:
| Agent | Role |
|-------|------|
| `planner` | Break down tasks into actionable steps |
| `architect` | Design architecture and define interfaces |
| `coder` | Write clean, efficient code |
| `reviewer` | Code review for correctness and security |
| `tester` | Write comprehensive tests |
| `documenter` | Create documentation and guides |

## Tools

The plugin registers these tools with OpenCode:

| Tool | Description |
|------|-------------|
| `apexcode_enhance` | Refresh MemPalace + Repomap context manually |
| `apexcode_swarm` | Execute multi-agent swarm |
| `apexcode_health` | Check Go backend connectivity |

## Slash Commands

| Command | Aliases | Description |
|---------|---------|-------------|
| `/sentinel` | `/suggest`, `/issues` | View proactive analysis issues |
| `/swarm` | `/agents` | Execute multi-agent swarm |
| `/apex` | `/apexcode` | Health check for Go backend |

## Architecture

```
┌─────────────────────────────────┐
│         OpenCode TUI            │
│  (your existing OpenCode UI)    │
└──────────────┬──────────────────┘
               │
    ┌──────────┴──────────┐
    │                     │
┌───┴──────┐      ┌──────┴────────┐
│ OpenCode │      │ ApexCode      │
│ Backend  │      │ Plugin        │
│ (agent)  │      │ (server+tui)  │
└──────────┘      └──────┬────────┘
                         │
                  ┌──────┴────────┐
                  │ Go Backend    │
                  │ :7777         │
                  │               │
                  │ MemPalace     │
                  │ Repomap       │
                  │ Sentinel        │
                  │ Swarms        │
                  └───────────────┘
```

## Configuration

The Go backend reads config from `~/.config/apexcode/config.json`. Available options:

```json
{
  "provider": "openai",
  "max_turns": 100,
  "work_dir": "/path/to/project"
}
```

API keys can be set via environment variables:
- `OPENAI_API_KEY`
- `ANTHROPIC_API_KEY`
- `GOOGLE_API_KEY`
- `GROQ_API_KEY`

## Troubleshooting

### Go backend not responding
```bash
# Check if it's running
curl http://localhost:7777/health

# Start it
apex --serve
```

### LM Studio models not appearing
- Ensure LM Studio is running on `http://127.0.0.1:1234`
- The plugin auto-discovers models at startup
- No manual configuration needed

### Plugin not loading
```bash
# Check plugin is installed
cat ~/.config/apexcode/apexcode.json
# Should include "apexcode-plugin" in the plugin array
```

## License

MIT
