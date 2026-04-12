#!/bin/bash
set -e

echo "🚀 Installing ApexCode..."

# Check for Go
if ! command -v go &> /dev/null; then
    echo "❌ Go is required but not installed."
    echo "   Install Go from https://golang.org/dl/"
    exit 1
fi

# Check for ripgrep
if ! command -v rg &> /dev/null; then
    echo "⚠️  ripgrep not found. Installing recommended for grep tool."
    echo "   Install from https://github.com/BurntSushi/ripgrep#installation"
fi

# Determine install location
INSTALL_DIR="/usr/local/bin"
if [ ! -w "$INSTALL_DIR" ]; then
    echo "⚠️  Cannot write to $INSTALL_DIR, using ~/.local/bin"
    INSTALL_DIR="$HOME/.local/bin"
    mkdir -p "$INSTALL_DIR"
fi

# Build
echo "📦 Building ApexCode..."
cd "$(dirname "$0")"
go build -o apex ./cmd/apex

# Install
cp apex "$INSTALL_DIR/apex"
chmod +x "$INSTALL_DIR/apex"

echo "✅ ApexCode installed successfully!"
echo ""
echo "Usage:"
echo "  apex              Start interactive mode"
echo "  apex --init       Initialize project"
echo "  apex \"task\"       Run a single task"
echo ""
echo "Set your API key:"
echo "  export OPENAI_API_KEY=sk-..."
echo "  export ANTHROPIC_API_KEY=sk-ant-..."
echo "  export GOOGLE_API_KEY=..."
echo ""
echo "For local models (LM Studio):"
echo "  - Start LM Studio server on port 1234"
echo "  - ApexCode will auto-detect it"
