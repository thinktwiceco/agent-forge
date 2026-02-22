#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "Agent Forge Install"
echo "==================="
read -r -p "Agent name: " AGENT_NAME
if [ -z "$AGENT_NAME" ]; then
  echo "Agent name cannot be empty."
  exit 1
fi

INSTALL_DIR="$PROJECT_ROOT/$AGENT_NAME"
BIN_DIR="$INSTALL_DIR/bin"

if [ -d "$INSTALL_DIR" ]; then
  echo "Directory '$INSTALL_DIR' already exists."
  read -r -p "Overwrite? [y/N] " confirm
  if [[ ! "$confirm" =~ ^[yY]$ ]]; then
    exit 1
  fi
  rm -rf "$INSTALL_DIR"
fi

mkdir -p "$BIN_DIR"

# Generate config.yaml
cat > "$INSTALL_DIR/config.yaml" << EOF
# Copy to config.yaml and fill in your values.
# Use dollar-brace syntax for env var interpolation, e.g. REPO_ROOT, DATABASE_URL.
agent:
  name: "$AGENT_NAME"
  system_prompt: |
    You are a helpful assistant. Customize this prompt for your use case.
  model: "togetherai::moonshotai/Kimi-K2.5"
  working_dir: "\${AGENT_WORKING_DIR}"  # start.sh exports this
  persistence: "json"
  tools:
    - name: fs
    - name: web
  subagents:
    reasoning: "deepseek::deepseek-reasoner"
  plugins:
    - "todo"
    - "vault"
EOF

# Build binary
echo "Building binary..."
cd "$PROJECT_ROOT"
go build -o "$BIN_DIR/localforge" ./cmd/localforge/src

# Create start.sh
cat > "$INSTALL_DIR/start.sh" << 'STARTEOF'
#!/bin/bash
set -euo pipefail

INSTALL_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
export AGENT_WORKING_DIR="$INSTALL_DIR"
cd "$INSTALL_DIR"
exec ./bin/localforge -config config.yaml -port 8080
STARTEOF
chmod +x "$INSTALL_DIR/start.sh"

echo ""
echo "Install complete: $INSTALL_DIR"
echo "  bin/localforge  - binary"
echo "  config.yaml    - agent config (edit model, prompt, etc.)"
echo "  start.sh       - run ./start.sh to start the agent"
echo ""
echo "Start with: cd $INSTALL_DIR && ./start.sh"
