#!/bin/bash
# Claude Agent System Demo Script

set -e

echo "🤖 Claude Long-Running Agent System Demo"
echo "========================================"

# Check prerequisites
echo "🔍 Checking prerequisites..."

if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go first."
    exit 1
fi

if [ -z "$OPENAI_API_KEY" ]; then
    echo "⚠️  Warning: OPENAI_API_KEY environment variable not set"
    echo "   Some features may not work properly"
fi

echo "✅ Prerequisites check passed"

# Build the agent
echo "🔨 Building agent..."
make build

# Create demo directory
DEMO_DIR="./demo-project"
echo "📁 Creating demo project in $DEMO_DIR"

# Initialize demo project
echo "🚀 Initializing demo project..."
./bin/agent -project-dir="$DEMO_DIR" -init -project-type=web-chat-app

echo "📋 Project initialized with the following structure:"
tree "$DEMO_DIR" || find "$DEMO_DIR" -print | sed -e 's/[^-][^\/]*\//--/g;s/^/   /'

# Show initial feature list
echo ""
echo "🎯 Initial feature list:"
cat "$DEMO_DIR/feature_list.json" | jq '.' 2>/dev/null || cat "$DEMO_DIR/feature_list.json"

# Show progress file
echo ""
echo "📝 Initial progress tracking:"
cat "$DEMO_DIR/claude-progress.txt"

echo ""
echo "🎉 Demo setup complete!"
echo ""
echo "Next steps:"
echo "1. cd $DEMO_DIR"
echo "2. Review the feature_list.json"
echo "3. Run the agent: ../bin/agent -project-dir=. -sessions=2"
echo "4. Check progress: cat claude-progress.txt"
echo ""
echo "💡 Tips:"
echo "- Run multiple sessions to see incremental development"
echo "- Check git history to see commit progression"
echo "- Review the generated code and tests"