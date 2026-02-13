#!/bin/bash
# 示例：Claude.ai 克隆项目初始化脚本

set -e

echo "🚀 初始化 Claude.ai 克隆项目..."

# 项目名称
PROJECT_NAME=${1:-"claude-clone"}
PROJECT_DIR="./$PROJECT_NAME"

# 创建项目目录
mkdir -p "$PROJECT_DIR"
cd "$PROJECT_DIR"

echo "📁 创建项目目录结构..."

# 创建基本目录结构
mkdir -p src/{components,pages,styles,utils,api}
mkdir -p public/assets
mkdir -p tests/e2e

# 创建基础文件
echo "📝 创建基础文件..."

# package.json
cat > package.json << EOF
{
  "name": "$PROJECT_NAME",
  "version": "1.0.0",
  "description": "Claude.ai clone built by AI agents",
  "main": "src/index.js",
  "scripts": {
    "dev": "vite",
    "build": "vite build",
    "preview": "vite preview",
    "test": "vitest"
  },
  "dependencies": {
    "react": "^18.2.0",
    "react-dom": "^18.2.0"
  },
  "devDependencies": {
    "@vitejs/plugin-react": "^4.0.0",
    "vite": "^4.4.0",
    "vitest": "^0.34.0"
  }
}
EOF

# vite.config.js
cat > vite.config.js << EOF
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 3000,
    open: true
  }
})
EOF

# index.html
cat > index.html << EOF
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Claude Clone</title>
</head>
<body>
    <div id="root"></div>
    <script type="module" src="/src/main.jsx"></script>
</body>
</html>
EOF

# src/main.jsx
cat > src/main.jsx << EOF
import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App'
import './styles/global.css'

ReactDOM.createRoot(document.getElementById('root')).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
)
EOF

# src/App.jsx
cat > src/App.jsx << EOF
import React from 'react'
import ChatInterface from './components/ChatInterface'
import './styles/App.css'

function App() {
  return (
    <div className="App">
      <header className="App-header">
        <h1>Claude Chat</h1>
      </header>
      <main className="App-main">
        <ChatInterface />
      </main>
    </div>
  )
}

export default App
EOF

# 创建样式文件
mkdir -p src/styles

cat > src/styles/global.css << EOF
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

body {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  background-color: #f5f5f5;
}

#root {
  height: 100vh;
}
EOF

cat > src/styles/App.css << EOF
.App {
  height: 100vh;
  display: flex;
  flex-direction: column;
}

.App-header {
  background-color: #2d2d2d;
  color: white;
  padding: 1rem;
  text-align: center;
}

.App-main {
  flex: 1;
  display: flex;
  overflow: hidden;
}
EOF

# 创建组件目录和基础组件
mkdir -p src/components

cat > src/components/ChatInterface.jsx << EOF
import React, { useState } from 'react'
import ChatHistory from './ChatHistory'
import MessageInput from './MessageInput'
import './ChatInterface.css'

function ChatInterface() {
  const [messages, setMessages] = useState([
    {
      id: 1,
      text: "Hello! I'm Claude. How can I help you today?",
      sender: 'ai',
      timestamp: new Date()
    }
  ])

  const handleSendMessage = (text) => {
    const newMessage = {
      id: messages.length + 1,
      text: text,
      sender: 'user',
      timestamp: new Date()
    }
    
    setMessages(prev => [...prev, newMessage])
    
    // Simulate AI response
    setTimeout(() => {
      const aiResponse = {
        id: messages.length + 2,
        text: "I understand your message. This is a simulated response.",
        sender: 'ai',
        timestamp: new Date()
      }
      setMessages(prev => [...prev, aiResponse])
    }, 1000)
  }

  return (
    <div className="chat-interface">
      <ChatHistory messages={messages} />
      <MessageInput onSendMessage={handleSendMessage} />
    </div>
  )
}

export default ChatInterface
EOF

cat > src/components/ChatInterface.css << EOF
.chat-interface {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  background-color: white;
}
EOF

cat > src/components/ChatHistory.jsx << EOF
import React from 'react'
import Message from './Message'
import './ChatHistory.css'

function ChatHistory({ messages }) {
  return (
    <div className="chat-history">
      {messages.map(message => (
        <Message key={message.id} message={message} />
      ))}
    </div>
  )
}

export default ChatHistory
EOF

cat > src/components/ChatHistory.css << EOF
.chat-history {
  flex: 1;
  overflow-y: auto;
  padding: 1rem;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}
EOF

cat > src/components/Message.jsx << EOF
import React from 'react'
import './Message.css'

function Message({ message }) {
  const isUser = message.sender === 'user'
  
  return (
    <div className={`message ${isUser ? 'user-message' : 'ai-message'}`}>
      <div className="message-content">
        {message.text}
      </div>
      <div className="message-timestamp">
        {message.timestamp.toLocaleTimeString()}
      </div>
    </div>
  )
}

export default Message
EOF

cat > src/components/Message.css << EOF
.message {
  max-width: 80%;
  padding: 0.75rem 1rem;
  border-radius: 12px;
  animation: fadeIn 0.3s ease-in;
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
}

.user-message {
  align-self: flex-end;
  background-color: #10a37f;
  color: white;
  border-bottom-right-radius: 4px;
}

.ai-message {
  align-self: flex-start;
  background-color: #f0f0f0;
  color: #333;
  border-bottom-left-radius: 4px;
}

.message-content {
  margin-bottom: 0.25rem;
  line-height: 1.4;
}

.message-timestamp {
  font-size: 0.75rem;
  opacity: 0.7;
}
EOF

cat > src/components/MessageInput.jsx << EOF
import React, { useState } from 'react'
import './MessageInput.css'

function MessageInput({ onSendMessage }) {
  const [inputText, setInputText] = useState('')

  const handleSubmit = (e) => {
    e.preventDefault()
    if (inputText.trim()) {
      onSendMessage(inputText.trim())
      setInputText('')
    }
  }

  return (
    <form className="message-input" onSubmit={handleSubmit}>
      <input
        type="text"
        value={inputText}
        onChange={(e) => setInputText(e.target.value)}
        placeholder="Message Claude..."
        className="message-input-field"
      />
      <button type="submit" className="send-button" disabled={!inputText.trim()}>
        Send
      </button>
    </form>
  )
}

export default MessageInput
EOF

cat > src/components/MessageInput.css << EOF
.message-input {
  display: flex;
  padding: 1rem;
  background-color: white;
  border-top: 1px solid #eee;
}

.message-input-field {
  flex: 1;
  padding: 0.75rem 1rem;
  border: 1px solid #ddd;
  border-radius: 20px;
  outline: none;
  font-size: 1rem;
  margin-right: 0.5rem;
}

.message-input-field:focus {
  border-color: #10a37f;
}

.send-button {
  padding: 0.75rem 1.5rem;
  background-color: #10a37f;
  color: white;
  border: none;
  border-radius: 20px;
  cursor: pointer;
  font-weight: 500;
}

.send-button:disabled {
  background-color: #ccc;
  cursor: not-allowed;
}

.send-button:hover:not(:disabled) {
  background-color: #0d8a6a;
}
EOF

# 创建测试文件
mkdir -p tests/unit

cat > tests/unit/chat.test.js << EOF
import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import App from '../../src/App'

describe('Chat Interface', () => {
  it('renders chat header', () => {
    render(<App />)
    expect(screen.getByText('Claude Chat')).toBeInTheDocument()
  })

  it('renders message input', () => {
    render(<App />)
    expect(screen.getByPlaceholderText('Message Claude...')).toBeInTheDocument()
  })
})
EOF

# 创建开发脚本
cat > dev.sh << 'EOF'
#!/bin/bash
# Development script for Claude clone

echo "🚀 Starting Claude Clone Development Server..."

# Install dependencies if node_modules doesn't exist
if [ ! -d "node_modules" ]; then
    echo "📦 Installing dependencies..."
    npm install
fi

# Start development server
echo "🔧 Starting development server on http://localhost:3000"
npm run dev
EOF

chmod +x dev.sh

echo "✅ 项目初始化完成!"
echo "📁 项目位置: $PROJECT_DIR"
echo ""
echo "下一步:"
echo "1. cd $PROJECT_DIR"
echo "2. chmod +x dev.sh"
echo "3. ./dev.sh"
echo ""
echo "或者使用AI Agent进行开发:"
echo "go run ../cmd/agent/main.go -project-dir=$PROJECT_DIR -sessions=5"