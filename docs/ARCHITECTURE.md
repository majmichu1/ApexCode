# ApexCode Architecture Reference

## Project Structure
```
apexcode/
├── cmd/apex/main.go              # CLI entry point
├── internal/
│   ├── agent/agent.go            # Core agent loop & orchestration
│   ├── swarm/swarm.go            # Multi-agent swarm coordination
│   ├── tools/tools.go            # Tool implementations (bash, file ops, grep, glob, web fetch)
│   ├── providers/
│   │   ├── provider.go           # Provider interface definition
│   │   ├── openai.go             # OpenAI/LM Studio/Ollama/Groq providers
│   │   ├── anthropic.go          # Anthropic Claude provider
│   │   ├── google.go             # Google Gemini provider
│   │   └── router.go             # Model auto-routing & failover
│   ├── memory/mempalace.go       # MemPalace memory system (wings/rooms/halls/drawers)
│   ├── git/safety.go             # Git safety manager
│   ├── lsp/lsp.go                # LSP bridge for code intelligence
│   ├── mcp/mcp.go                # MCP server integration
│   └── config/config.go          # Configuration management
├── pkg/
│   ├── repomap/repomap.go        # Tree-sitter repository map generator
│   └── security/permissions.go   # Permission & security system
├── tui/app.go                    # Bubble Tea terminal UI
├── daemon/daemon.go              # Background daemon (KAIROS-like)
├── plugins/
│   ├── package.json              # Node.js dependencies
│   └── runtime.js                # TypeScript plugin runtime
├── scripts/install.sh            # Installation script
├── go.mod                        # Go module definition
├── README.md                     # Project documentation
└── APEX.md.template              # Project context template
```

## Key Components

### 1. Agent Loop (internal/agent/agent.go)
- Multi-turn agent with parallel tool execution
- Sync.WaitGroup for concurrent tool calls
- Configurable max turns
- Memory palace context injection

### 2. MemPalace Memory (internal/memory/mempalace.go)
- **Wings**: Top-level categories (code, docs, config)
- **Rooms**: Subtopics within wings
- **Halls**: Category metadata (facts, events)
- **Drawers**: Content chunks with importance scoring
- **AAAK Compression**: Deterministic abbreviation
- **Knowledge Graph**: SQLite with entities/triples/diary
- **Progressive Loading**: L0 (170 tokens) → L3 (full search)
- **Tunnels**: Cross-wing room connections

### 3. LLM Providers (internal/providers/)
- OpenAI (GPT-4o, etc.)
- Anthropic (Claude Sonnet/Opus)
- Google (Gemini 2.5 Pro)
- LM Studio (local, priority)
- Ollama (local)
- Groq (fast inference)
- Router with auto-failover

### 4. Tool System (internal/tools/tools.go)
- `bash` - Command execution with timeout & safety
- `read_file` - File reading with offset/limit
- `write_file` - File creation/writing
- `edit_file` - String replacement editing
- `grep` - Pattern search via ripgrep
- `glob` - File pattern matching
- `web_fetch` - URL content retrieval

### 5. Multi-Agent Swarms (internal/swarm/swarm.go)
- Parent spawns child agents
- Isolated contexts per child
- Parallel execution
- Result merging
- Status tracking

### 6. TUI (tui/app.go)
- Bubble Tea framework
- Plan mode (read-only) / Build mode (read/write)
- Streaming output
- Keyboard shortcuts (Tab, Ctrl+C, ?)
- Lipgloss styling

### 7. MCP Integration (internal/mcp/mcp.go)
- JSON-RPC 2.0 client
- Multiple server connections
- Tool discovery
- Execution routing

### 8. LSP Bridge (internal/lsp/lsp.go)
- Server management per language
- Completions
- Diagnostics
- Symbol definition lookup

### 9. Git Safety (internal/git/safety.go)
- Auto-stash before risky ops
- Branch creation
- Pre-commit checks
- Safe push (with force warnings)
- Diff/log access

### 10. Background Daemon (daemon/daemon.go)
- Tick-based loop (30s default)
- Memory consolidation
- Idle detection
- Append-only logging
- Brief mode support

### 11. Security System (pkg/security/permissions.go)
- Permission levels: ReadOnly, ReadWrite, FullAccess
- Tool allowlisting
- Path blocking
- Untrusted repo mode
- User approval gates

### 12. Model Router (internal/providers/router.go)
- Auto-select by task complexity
- Failover chain
- Provider registration
- Status reporting

## Next Steps for Development

1. **Run `go mod tidy`** to resolve all dependencies
2. **Build the project**: `go build -o apex ./cmd/apex`
3. **Test individual packages** with `go test ./...`
4. **Add real tree-sitter bindings** (currently simplified parser)
5. **Implement full ChromaDB integration** for MemPalace vector store
6. **Add streaming for Anthropic/Google** providers
7. **Implement LSP protocol** (currently uses linting fallback)
8. **Add WebSocket support for MCP** (currently HTTP placeholder)
9. **Write comprehensive tests** for all components
10. **Create benchmark suite** for comparison with other tools
