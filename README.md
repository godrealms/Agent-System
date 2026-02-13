# Claude Long-Running Agent System

这是一个基于Anthropic工程实践的长运行Agent系统，专门设计用于处理需要跨多个上下文窗口的复杂任务。

## 系统架构

### 核心组件

1. **Agent Core** (`pkg/agent`) - Agent框架和类型定义
2. **Environment Manager** (`pkg/environment`) - 环境设置和Git集成
3. **Feature Manager** (`pkg/features`) - 功能列表和状态管理
4. **Test Manager** (`pkg/testing`) - 测试和验证模块
5. **Harness** (`pkg/harness`) - 主要编排器

### 设计模式

系统采用双Agent模式：
- **Initializer Agent**: 负责首次项目设置
- **Coding Agent**: 负责增量开发

## 快速开始

### 安装依赖

```bash
go mod tidy
```

### 设置环境变量

```bash
export OPENAI_API_KEY="your-openai-api-key"
```

### 初始化新项目

```bash
go run cmd/agent/main.go -project-dir=/path/to/project -init -project-type=web-chat-app
```

### 运行Agent会话

```bash
# 运行单个会话
go run cmd/agent/main.go -project-dir=/path/to/project -sessions=1

# 运行多个会话
go run cmd/agent/main.go -project-dir=/path/to/project -sessions=5

# 生成项目报告
go run cmd/agent/main.go -project-dir=/path/to/project -report
```

## 系统特性

### 1. 上下文窗口管理
- 自动上下文压缩
- 跨会话状态保持
- 渐进式开发模式

### 2. 环境管理
- 自动Git集成
- 进度跟踪文件
- 环境状态同步

### 3. 功能驱动开发
- 结构化功能列表
- 优先级排序
- 自动状态更新

### 4. 测试和验证
- 端到端测试
- 单元测试集成
- 自动验证

## 配置选项

在 `internal/config/config.go` 中可以调整：

```go
type Config struct {
    OpenAIKey      string  // OpenAI API密钥
    ProjectDir     string  // 项目目录
    MaxContextSize int     // 最大上下文大小
    SessionTimeout int     // 会话超时时间
    GitEnabled     bool    // 是否启用Git
    TestEnabled    bool    // 是否启用测试
    BrowserTool    string  // 浏览器工具选择
}
```

## 使用示例

### 创建聊天应用

```bash
# 1. 初始化项目
go run cmd/agent/main.go -project-dir=./my-chat-app -init -project-type=web-chat-app

# 2. 运行开发会话
go run cmd/agent/main.go -project-dir=./my-chat-app -sessions=10

# 3. 查看进度
cat ./my-chat-app/claude-progress.txt
```

### 项目结构示例

初始化后会创建以下文件：
```
my-chat-app/
├── init.sh                 # 开发服务器启动脚本
├── claude-progress.txt     # 进度跟踪文件
├── feature_list.json       # 功能列表
├── README.md              # 项目说明
└── .git/                  # Git仓库
```

## 最佳实践

### 1. 会话管理
- 每次只实现一个功能
- 确保代码处于可合并状态
- 充分测试后再提交

### 2. 功能列表
- 功能描述要具体明确
- 包含详细的测试步骤
- 设置合理的优先级

### 3. 测试策略
- 实现端到端测试
- 验证现有功能不受影响
- 截图记录测试结果

## 故障排除

### 常见问题

1. **API密钥错误**
   ```
   Error: OPENAI_API_KEY environment variable is required
   ```
   解决：确保设置了有效的OpenAI API密钥

2. **Git初始化失败**
   ```
   failed to initialize git: exit status 128
   ```
   解决：检查Git是否正确安装并配置

3. **上下文长度超限**
   ```
   context length exceeded
   ```
   解决：调整 `MaxContextSize` 或优化提示词

### 调试技巧

1. 查看详细日志输出
2. 检查进度文件内容
3. 验证Git提交历史
4. 手动测试功能实现

## 扩展开发

### 添加新的项目类型

在 `pkg/features/manager.go` 中添加：

```go
func (fm *FeatureManager) generateMyProjectFeatures() []Feature {
    return []Feature{
        {
            ID: "my-feature-001",
            Category: "core",
            Description: "My feature description",
            Steps: []string{"step 1", "step 2"},
            Passes: false,
            Priority: 1,
        },
    }
}
```

### 自定义测试逻辑

在 `pkg/testing/manager.go` 中扩展：

```go
func (tm *TestManager) runCustomTests() []TestResult {
    // 实现自定义测试逻辑
}
```

## 许可证

MIT License

## 致谢

灵感来源于Anthropic的《Effective harnesses for long-running agents》工程实践。