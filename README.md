# ApexCode — The Ultimate AI Coding Agent

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8.svg)](https://golang.org/)

**ApexCode** combines the best features from Claude Code, OpenCode, Aider, Codex CLI, Gemini CLI, and more into a single, superior tool that outperforms all of them.

## 🚀 Features

### Core Capabilities
- **Multi-Agent Swarms** — Spawn sub-agents with isolated contexts for parallel task execution
- **MemPalace Memory System** — Revolutionary memory architecture using the method of loci (Wings/Rooms/Halls/Drawers)
- **MCP Integration** — Full Model Context Protocol support with 300+ existing integrations
- **LSP Code Intelligence** — Language Server Protocol bridge for completions, diagnostics, and symbol navigation
- **Tree-Sitter Repo Map** — AST-based codebase analysis with graph ranking
- **Git Safety System** — Auto-stash, branch protection, pre-commit checks
- **Background Daemon** — Persistent service with memory consolidation and proactive suggestions

### LLM Provider Support
- **Cloud**: OpenAI, Anthropic, Google, Groq, AWS Bedrock, Azure, OpenRouter
- **Local**: LM Studio (priority), Ollama, any OpenAI-compatible server
- **Model Routing**: Auto-select model based on task complexity
- **Fallback**: Automatic provider failover

## 📦 Installation

### Quick Install
```bash
curl -fsSL https://apexcode.dev/install.sh | sh
```

### Build from Source
```bash
git clone https://github.com/apexcode/apexcode.git
cd apexcode
go mod download
go build -o apex ./cmd/apex
sudo mv apex /usr/local/bin/
```

### Homebrew (macOS/Linux)
```bash
brew tap apexcode/apexcode
brew install apexcode
```

## 🎯 Usage

### Start Interactive Mode
```bash
apex
```

### Run a Single Task
```bash
apex "fix the authentication bug in auth.py"
```

### Initialize Project
```bash
apex --init
```

This creates a `APEX.md` file for project-specific context.

## ⌨️ Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `Enter` | Send message |
| `Tab` | Toggle Plan/Build mode |
| `Ctrl+C` / `Ctrl+Q` | Quit |
| `?` | Toggle help |

### Plan Mode (Read-Only)
- Analyze code without making changes
- Ask questions about the codebase
- Safe for exploring unfamiliar code

### Build Mode (Read/Write)
- AI can read and modify files
- Execute commands and fix bugs
- Full agent capabilities

## 🔧 Configuration

Configuration is stored in `~/.config/apexcode/config.json`:

```json
{
  "provider": "openai",
  "providers": {
    "openai": {
      "api_key": "sk-...",
      "model": "gpt-4o"
    },
    "anthropic": {
      "api_key": "sk-ant-...",
      "model": "claude-sonnet-4-20250514"
    },
    "lmstudio": {
      "base_url": "http://localhost:1234/v1",
      "model": "local-model"
    }
  }
}
```

Or use environment variables:
```bash
export OPENAI_API_KEY=sk-...
export ANTHROPIC_API_KEY=sk-ant-...
export GOOGLE_API_KEY=...
```

## 🧠 MemPalace Memory System

ApexCode uses **MemPalace** (by Milla Jovovich) for persistent memory:

- **Wings** → Top-level categories (project, personal, topics)
- **Rooms** → Subtopics within wings
- **Halls** → Category metadata (facts, events)
- **Drawers** → Actual content chunks
- **Tunnels** → Cross-domain bridges between rooms
- **AAAK Compression** → Deterministic abbreviation scheme
- **Knowledge Graph** → Temporal fact tracking in SQLite
- **170-Token Wake-Up** → Minimal context loading on startup

## 🏗️ Architecture

```
┌─────────────────────────────────────────────┐
│           Terminal UI (Bubble Tea)          │
├─────────────────────────────────────────────┤
│          Go Core (Agent Orchestrator)       │
│  ┌─────────────┬──────────────┬──────────┐  │
│  │ Agent Loop  │ Tool Router  │ Security │  │
│  └─────────────┴──────────────┴──────────┘  │
├─────────────────────────────────────────────┤
│      TypeScript Plugin Runtime (Node)       │
│  ┌─────────────┬──────────────┬──────────┐  │
│  │ MCP Servers │ LSP Bridge   │ Plugins  │  │
│  └─────────────┴──────────────┴──────────┘  │
├─────────────────────────────────────────────┤
│          LLM Provider Abstraction           │
│  (OpenAI, Anthropic, Google, LM Studio,     │
│   Groq, Ollama, Bedrock, Azure, OpenRouter) │
└─────────────────────────────────────────────┘
```

## 📊 Comparison with Other Tools

| Feature | ApexCode | Claude Code | OpenCode | Aider |
|---------|----------|-------------|----------|-------|
| Multi-agent swarms | ✅ | ✅ (hidden) | ❌ | ❌ |
| MemPalace memory | ✅ | ❌ | ❌ | ❌ |
| MCP integration | ✅ | ✅ | ✅ | ❌ |
| LSP support | ✅ | ✅ | ✅ | ❌ |
| Tree-sitter map | ✅ | ❌ | ❌ | ✅ |
| Local models (LM Studio) | ✅ (priority) | ❌ | ✅ | ✅ |
| Git safety | ✅ | ✅ | ❌ | ✅ |
| Background daemon | ✅ | ✅ (hidden) | ❌ | ❌ |
| Plugin system | ✅ | Limited | Limited | ❌ |
| Multi-provider | ✅ All | Anthropic only | ✅ 75+ | ✅ Many |

## 🤝 Contributing

Contributions are welcome! Please read our contributing guidelines and submit pull requests.

## 📝 License

This project is currently closed-source. Future licensing plans are under consideration.

## 🙏 Acknowledgments

- **MemPalace** by Milla Jovovich & Ben Sigman — Revolutionary memory architecture
- **Aider** — Pioneer CLI coding assistant with tree-sitter repo maps
- **OpenCode** — Open-source AI coding agent with excellent TUI
- **Claude Code** — Inspired many features from the leaked source
- **Charmbracelet** — Beautiful terminal UI libraries
- **Tree-sitter** — Incremental parsing library

---

Built with ❤️ by the ApexCode team
