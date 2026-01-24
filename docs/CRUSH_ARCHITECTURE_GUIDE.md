# Crush/ZorkAgent 架构学习指南

> 本文档总结了 Crush/ZorkAgent 项目的核心架构、配置机制和设计理念。

---

## 目录

1. [系统架构概览](#1-系统架构概览)
2. [Agent 协作机制](#2-agent-协作机制)
3. [权限控制系统](#3-权限控制系统)
4. [系统提示词机制](#4-系统提示词机制)
5. [Provider 配置系统](#5-provider-配置系统)
6. [配置文件详解](#6-配置文件详解)
7. [最佳实践](#7-最佳实践)

---

## 1. 系统架构概览

### 1.1 核心组件

```
┌─────────────────────────────────────────────────────────────┐
│                        用户输入                               │
└─────────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────────┐
│                      API Handler                             │
│  (api/handlers/messages.go:155)                             │
│  - 接收用户请求                                               │
│  - 调用 coordinator.Run()                                    │
└─────────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────────┐
│                   Coordinator (协调器)                        │
│  (internal/agent/coordinator.go)                             │
│  - 管理 Agent 生命周期                                        │
│  - 构建 Agent 和工具                                          │
│  - 路由用户请求到对应的 Agent                                  │
└─────────────────────────────────────────────────────────────┘
                          ↓
        ┌─────────────────┴─────────────────┐
        ↓                                     ↓
┌──────────────────────┐          ┌──────────────────────┐
│   Coder Agent        │          │   Task Agent         │
│   (主要 Agent)        │          │   (子 Agent)          │
│                       │          │                       │
│ - 全部工具            │          │ - 只读工具            │
│ - Large Model         │          │ - Large Model         │
│ - 代码修改            │          │ - 代码搜索            │
└──────────────────────┘          └──────────────────────┘
        ↓                                     ↓
┌─────────────────────────────────────────────────────────────┐
│                    工具层 (Tools)                             │
│  bash, edit, view, grep, glob, ls, sourcegraph, etc.        │
└─────────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────────┐
│                   权限服务 (Permission)                       │
│  - 白名单检查                                                │
│  - 用户批准/拒绝                                             │
│  - YOLO 模式                                                 │
└─────────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────────┐
│                   会话管理 (Session)                          │
│  - 父子 Session 关系                                         │
│  - 消息历史                                                  │
│  - 成本追踪                                                  │
└─────────────────────────────────────────────────────────────┘
```

### 1.2 关键设计原则

1. **Coder 中心制**：用户始终与 Coder Agent 交互
2. **按需创建子 Agent**：Task/Fetch Agent 动态创建
3. **权限分层控制**：工具级白名单 + 运行时权限检查
4. **智能配置合并**：用户配置 > 内置配置
5. **系统提示词优化**：禁用工具自动移除描述

---

## 2. Agent 协作机制

### 2.1 Agent 类型对比

| 特性 | **Coder Agent** | **Task Agent** | **Fetch Agent** |
|------|----------------|----------------|----------------|
| **用途** | 编程任务 | 代码搜索 | 网页分析 |
| **提示词** | 380+ 行详细规则 | 15 行简洁规则 | 77 行网页分析规则 |
| **工具** | 全部工具 (14+) | 只读工具 (5) | 网络工具 (6) |
| **编辑** | ✅ edit, multiedit | ❌ | ❌ |
| **执行** | ✅ bash | ❌ | ❌ |
| **网络** | ❌ | ❌ | ✅ web_search, fetch |
| **模型** | Large | Large | Small |
| **MCP/LSP** | ✅ | ❌ | ❌ |
| **触发方式** | 默认启动 | Coder 调用 | Coder 调用 |
| **权限** | 需要批准 | 自动批准子 Session | 自动批准子 Session |

### 2.2 协作流程

```
用户: "帮我实现 JWT 认证功能，参考最佳实践"
   ↓
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Coder Agent 开始工作
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
   ↓
Coder: "我需要先了解 JWT 最佳实践"
   ↓
调用 agentic_fetch 工具 → 创建子 Session
   ↓
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Fetch Agent (Small Model) 启动
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
   ↓
工具: web_search, web_fetch, glob, grep, view
   ↓
返回: "JWT 最佳实践：使用 RS256，设置合理过期时间..."
   ↓
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Coder Agent 继续工作
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
   ↓
Coder: "现在让我查看现有的认证代码"
   ↓
调用 agent 工具 → 创建子 Session
   ↓
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Task Agent 启动
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
   ↓
工具: glob, grep, ls, view, sourcegraph
   ↓
返回: "Found auth code in: internal/auth/jwt.go:45"
   ↓
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Coder Agent 实现功能
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
   ↓
工具: view, edit, multiedit, bash
   ↓
返回: "✅ 已实现用户认证功能"
```

### 2.3 子 Agent 调用机制

**agent 工具**（[agent_tool.go:27-105](internal/agent/agent_tool.go)）：

```go
func (c *coordinator) agentTool(ctx context.Context) (fantasy.AgentTool, error) {
    // 1. 获取 Task Agent 配置
    agentCfg, ok := c.cfg.Agents[config.AgentTask]

    // 2. 创建 Task Agent (isSubAgent=true)
    agent, err := c.buildAgent(ctx, prompt, agentCfg, true)

    // 3. 返回工具包装器
    return fantasy.NewParallelAgentTool(
        "agent",
        "Launch a new agent with Glob, Grep, LS, View tools...",
        func(ctx context.Context, params AgentParams, call fantasy.ToolCall) {
            // 4. 创建子 Session
            agentToolSessionID := c.sessions.CreateAgentToolSessionID(
                agentMessageID, call.ID
            )
            session, err := c.sessions.CreateTaskSession(
                ctx,
                agentToolSessionID,
                sessionID,  // 父 Session ID
                "New Agent Session"
            )

            // 5. 运行 Task Agent
            result, err := agent.Run(ctx, SessionAgentCall{...})

            // 6. 累加成本到父 Session
            parentSession.Cost += updatedSession.Cost

            // 7. 返回结果给 Coder
            return fantasy.NewTextResponse(result.Response.Content.Text())
        }
    ), nil
}
```

### 2.4 Session 父子关系

**Session 结构**：
```go
type Session struct {
    ID              string
    ParentSessionID string  // ← 父 Session ID
    Cost            float64
    // ...
}
```

**Session 树**：
```
主 Session (用户对话)
  Cost: $0.15
  |
  ├─ 消息 1: "帮我看下代码结构"
  │   └─ 子 Session (Task Agent)
  │       Cost: $0.02
  │       → grep, glob, view 调用
  │
  ├─ 消息 2: "实现新功能"
  │   └─ 子 Session (Fetch Agent)
  │       Cost: $0.03
  │       → web_search, web_fetch 调用
  │
  └─ 消息 3: "修复 bug"

总成本: $0.15 + $0.02 + $0.03 = $0.20
```

---

## 3. 权限控制系统

### 3.1 两层权限机制

```
┌─────────────────────────────────────────────────────────────┐
│  第一层：工具可用性 (Agent.AllowedTools)                    │
│                                                             │
│  配置: disabled_tools, AllowedTools                         │
│  ↓                                                           │
│  buildTools() 过滤工具                                       │
│  ↓                                                           │
│  效果: 禁用的工具根本不会被创建                              │
└─────────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────────┐
│  第二层：运行时权限检查 (PermissionService)                 │
│                                                             │
│  配置: permissions.allowed_tools                             │
│  ↓                                                           │
│  PermissionService.Request()                                │
│  ↓                                                           │
│  效果: 白名单自动批准，其他需要用户批准                      │
└─────────────────────────────────────────────────────────────┘
```

### 3.2 权限配置流程

**代码位置**：[config.go:731-755](internal/config/config.go#L731-L755)

```go
func (c *Config) SetupAgents() {
    // 1. 从全局工具列表中移除 disabled_tools
    allowedTools := resolveAllowedTools(allToolNames(), c.Options.DisabledTools)

    agents := map[string]Agent{
        AgentCoder: {
            AllowedTools: allowedTools,  // Coder 获得过滤后的工具
        },

        AgentTask: {
            AllowedTools: resolveReadOnlyTools(allowedTools),  // Task 再过滤只读工具
            AllowedMCP: map[string][]string{},  // Task 禁用 MCP
        },
    }
}
```

**过滤流程**：
```
allToolNames() (14 个工具)
   ↓
resolveAllowedTools(排除 disabled_tools: ["bash", "edit"])
   ↓
allowedTools (12 个工具)
   ↓
   ├─ Coder.AllowedTools = 12 个工具
   └─ Task.AllowedTools = resolveReadOnlyTools(12) = 5 个只读工具
```

### 3.3 工具构建和过滤

**代码位置**：[coordinator.go:360-439](internal/agent/coordinator.go#L360-L439)

```go
func (c *coordinator) buildTools(ctx context.Context, agent config.Agent) ([]fantasy.AgentTool, error) {
    var allTools []fantasy.AgentTool

    // 1. 添加所有可能的工具
    allTools = append(allTools,
        tools.NewBashTool(...),
        tools.NewEditTool(...),
        tools.NewGrepTool(...),
        // ... 等等
    )

    // 2. 根据 Agent.AllowedTools 过滤
    var filteredTools []fantasy.AgentTool
    for _, tool := range allTools {
        if slices.Contains(agent.AllowedTools, tool.Info().Name) {
            filteredTools = append(filteredTools, tool)
        }
    }

    return filteredTools, nil
}
```

### 3.4 运行时权限检查

**代码位置**：[permission.go:132-217](internal/permission/permission.go#L132-L217)

```go
func (s *permissionService) Request(ctx context.Context, opts CreatePermissionRequest) (bool, error) {
    // 1. YOLO 模式：全部跳过
    if s.skip {
        return true, nil
    }

    // 2. 检查白名单
    commandKey := opts.ToolName + ":" + opts.Action
    if slices.Contains(s.allowedTools, commandKey) ||
       slices.Contains(s.allowedTools, opts.ToolName) {
        return true, nil  // 自动批准
    }

    // 3. 检查会话自动批准（子 Agent）
    if s.autoApproveSessions[opts.SessionID] {
        return true, nil
    }

    // 4. 检查持久化的权限
    for _, p := range s.sessionPermissions {
        if p.ToolName == permission.ToolName &&
           p.SessionID == permission.SessionID {
            return true, nil  // 之前批准过
        }
    }

    // 5. 发布权限请求事件，等待用户响应
    s.Publish(pubsub.CreatedEvent, permission)

    select {
    case <-ctx.Done():
        return false, ctx.Err()
    case granted := <-respCh:
        return granted, nil
    }
}
```

---

## 4. 系统提示词机制

### 4.1 系统提示词组成

```
系统提示词 = Provider 前缀 + Agent 模板 + 上下文文件 + 环境信息
```

**组成结构**：
```
┌─────────────────────────────────────────────────────────────┐
│  1. Provider system_prompt_prefix (可选)                     │
│     "You are a senior Go developer.\n\n"                    │
└─────────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────────┐
│  2. Agent 模板内容                                          │
│     - Coder: 380+ 行 (coder.md.tpl)                         │
│     - Task: 15 行 (task.md.tpl)                             │
│     - Fetch: 77 行 (agentic_fetch_prompt.md.tpl)            │
└─────────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────────┐
│  3. 工具描述 (动态生成)                                      │
│     - 每个工具的详细说明 (100-200 tokens/工具)                │
│     - 禁用的工具不会包含                                      │
└─────────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────────┐
│  4. 上下文文件内容                                          │
│     - .cursorrules, AGENTS.md, etc.                         │
└─────────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────────┐
│  5. 环境信息                                                │
│     - Working directory, Platform, Date, Git status         │
└─────────────────────────────────────────────────────────────┘
```

### 4.2 提示词优化机制

**关键发现**：禁用工具会减少系统提示词大小！

**代码流程**：
```
buildTools()
   ↓
根据 AllowedTools 过滤工具
   ↓
返回过滤后的工具列表
   ↓
fantasy.WithTools(agentTools...)
   ↓
fantasy 框架将工具描述添加到系统提示词
```

**效果对比**：

| 配置 | 工具数量 | 提示词大小 | 减少 |
|------|---------|-----------|-----|
| 全部工具 | 14 | ~8000 tokens | - |
| 禁用 bash, edit | 11 | ~5000 tokens | 37% |
| 只读模式 | 5 | ~2000 tokens | 75% |

**优化建议**：
```json
{
  "options": {
    "disabled_tools": ["bash", "edit", "multiedit", "write"]
  },
  "permissions": {
    "allowed_tools": ["view", "ls", "glob", "grep"]
  }
}
```

### 4.3 Provider 级别的前缀

**配置**：
```json
{
  "providers": {
    "anthropic": {
      "system_prompt_prefix": "你是一个专业的中文 AI 助手。\n\n"
    },
    "openai": {
      "system_prompt_prefix": "You are a senior Go developer.\n\n"
    }
  }
}
```

**注入位置**（[agent.go:270-272](internal/agent/agent.go#L270-L272)）：
```go
if promptPrefix != "" {
    prepared.Messages = append(
        []fantasy.Message{fantasy.NewSystemMessage(promptPrefix)},
        prepared.Messages...,
    )
}
```

---

## 5. Provider 配置系统

### 5.1 Provider 来源

```
Provider 来源（优先级）：
   ↓
1. 内置 Provider (embedded)
   - 编译时打包在代码中
   - 位置: github.com/charmbracelet/catwalk/pkg/embedded
   ↓
2. 缓存 Provider
   - 本地缓存文件
   - Linux: ~/.local/share/crush/providers.json
   - Windows: %LOCALAPPDATA%\crush\providers.json
   ↓
3. 在线 Provider
   - 从 Catwalk API 获取
   - URL: https://catwalk.charm.sh/v2/providers
```

### 5.2 Provider 加载流程

**代码位置**：[provider.go:142-188](internal/config/provider.go#L142-L188)

```go
func Providers(cfg *Config) ([]catwalk.Provider, error) {
    autoupdate := !cfg.Options.DisableProviderAutoUpdate

    // 1. 获取 Catwalk Providers
    wg.Go(func() {
        catwalkSyncer.Init(client, cachePath, autoupdate)
        items, err := catwalkSyncer.Get(ctx)
        // ↓ 尝试顺序：
        // 1. 缓存文件
        // 2. 在线 API (如果 autoupdate=true)
        // 3. 内置 embedded (fallback)
    })

    // 2. 获取 Hyper Provider
    wg.Go(func() {
        if hyper.Enabled() {
            hyperSyncer.Get(ctx)
        }
    })

    return providerList, nil
}
```

### 5.3 配置合并逻辑

**代码位置**：[load.go:120-330](internal/config/load.go#L120-L330)

**规则**：用户配置 > 内置配置

**示例**：

**内置 Provider**：
```go
Provider{
    ID: "anthropic",
    APIKey: "",
    BaseURL: "https://api.anthropic.com",
    Models: [
        {ID: "claude-sonnet-4", Name: "Claude Sonnet 4"},
        {ID: "claude-haiku-4", Name: "Claude Haiku 4"},
    ]
}
```

**用户配置**：
```json
{
  "providers": {
    "anthropic": {
      "api_key": "$ANTHROPIC_API_KEY",
      "system_prompt_prefix": "你是一个专业的中文 AI 助手。\n\n",
      "models": [
        {"id": "claude-opus-4", "name": "Claude Opus 4"}
      ]
    }
  }
}
```

**合并结果**：
```go
Provider{
    ID: "anthropic",
    APIKey: "sk-ant-xxxxx",           // ← 用户覆盖
    BaseURL: "https://api.anthropic.com",  // ← 内置保留
    SystemPromptPrefix: "你是一个专业的中文 AI 助手。\n\n",  // ← 用户添加
    Models: [  // ← 合并去重
        {ID: "claude-opus-4"},      // 用户
        {ID: "claude-sonnet-4"},    // 内置
        {ID: "claude-haiku-4"},     // 内置
    ]
}
```

---

## 6. 配置文件详解

### 6.1 配置文件位置（优先级从低到高）

```
1. 全局配置
   Linux/macOS: ~/.config/crush/crush.json
   Windows: %LOCALAPPDATA%\crush\crush.json

2. 全局数据（运行时修改）
   Linux/macOS: ~/.local/share/crush/crush.json
   Windows: %LOCALAPPDATA%\crush\crush.json

3. 项目配置（优先级最高）
   .crush.json
   crush.json
```

**注意**：后面的配置会覆盖前面的配置（通过 `jsons.Merge` 合并）

### 6.2 完整配置示例

```json
{
  "$schema": "https://charm.sh/crush.json",

  // ============ 模型配置 ============
  "models": {
    "large": {
      "model": "claude-sonnet-4-20250514",
      "provider": "anthropic",
      "max_tokens": 8192,
      "temperature": 0.7
    },
    "small": {
      "model": "claude-haiku-4-20250514",
      "provider": "anthropic"
    }
  },

  // ============ Provider 配置 ============
  "providers": {
    "anthropic": {
      "api_key": "$ANTHROPIC_API_KEY",
      "system_prompt_prefix": "你是一个专业的中文 AI 助手。\n\n"
    },
    "ollama": {
      "id": "ollama",
      "name": "Ollama",
      "base_url": "http://localhost:11434/v1",
      "api_key": "sk-xxx",
      "type": "openai-compat",
      "models": [
        {"id": "llama3.2", "name": "Llama 3.2"}
      ]
    }
  },

  // ============ 权限配置 ============
  "permissions": {
    "allowed_tools": ["view", "ls", "glob", "grep"]
  },

  // ============ 全局选项 ============
  "options": {
    "disabled_tools": ["bash", "edit", "multiedit"],
    "context_paths": [".cursorrules", "AGENTS.md"],
    "debug": false,
    "disable_provider_auto_update": false,
    "tui": {
      "compact_mode": false,
      "diff_mode": "unified"
    }
  },

  // ============ LSP 配置 ============
  "lsp": {
    "gopls": {
      "command": "gopls",
      "filetypes": ["go", "mod"]
    }
  },

  // ============ MCP 配置 ============
  "mcp": {
    "filesystem": {
      "type": "stdio",
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-filesystem"]
    }
  }
}
```

### 6.3 关键配置选项

| 选项 | 说明 | 推荐值 |
|------|------|--------|
| `disabled_tools` | 完全禁用的工具 | `["bash"]` (安全) |
| `permissions.allowed_tools` | 白名单工具（自动批准） | `["view", "ls", "glob", "grep"]` |
| `disable_provider_auto_update` | 禁用 Provider 在线更新 | `true` (离线环境) |
| `disable_default_providers` | 禁用所有内置 Provider | `true` (完全自定义) |
| `context_paths` | 上下文文件路径 | `[".cursorrules"]` |

---

## 7. 最佳实践

### 7.1 推荐配置（个人开发）

**`~/.config/crush/crush.json`**
```json
{
  "$schema": "https://charm.sh/crush.json",

  "models": {
    "large": {
      "model": "claude-sonnet-4-20250514",
      "provider": "anthropic"
    },
    "small": {
      "model": "claude-haiku-4-20250514",
      "provider": "anthropic"
    }
  },

  "providers": {
    "anthropic": {
      "api_key": "$ANTHROPIC_API_KEY",
      "system_prompt_prefix": "你是一个专业的中文 AI 助手，擅长 Go 开发。\n\n"
    }
  },

  "permissions": {
    "allowed_tools": ["view", "ls", "glob", "grep"]
  },

  "options": {
    "context_paths": [".cursorrules", "AGENTS.md"]
  }
}
```

### 7.2 团队项目配置

**`.crush.json`**
```json
{
  "options": {
    "disabled_tools": ["bash"],  // 禁止执行命令
    "context_paths": ["docs/CONVENTIONS.md"],
    "initialize_as": "docs/LLMs.md"
  },

  "permissions": {
    "allowed_tools": ["view", "ls", "glob", "grep", "edit", "multiedit"]
  }
}
```

### 7.3 只读模式（代码审查）

**`.crush.json`**
```json
{
  "options": {
    "disabled_tools": ["bash", "edit", "multiedit", "write", "download"]
  },
  "permissions": {
    "allowed_tools": ["view", "ls", "glob", "grep", "sourcegraph"]
  }
}
```

### 7.4 性能优化建议

1. **禁用不需要的工具**
   ```json
   {"options": {"disabled_tools": ["bash", "download"]}}
   ```
   - 减少 30-40% 的 tokens
   - 提升 AI 响应速度

2. **使用只读工具白名单**
   ```json
   {"permissions": {"allowed_tools": ["view", "ls", "glob", "grep"]}}
   ```
   - 跳过权限提示
   - 提升开发体验

3. **禁用 Provider 自动更新**
   ```json
   {"options": {"disable_provider_auto_update": true}}
   ```
   - 更快的启动速度
   - 减少网络请求

---

## 8. 关键代码位置索引

### 8.1 核心文件

| 文件 | 位置 | 功能 |
|------|------|------|
| **配置加载** | `internal/config/load.go` | 配置文件读取和合并 |
| **配置管理** | `internal/config/config.go` | 配置结构和 Agent 设置 |
| **Provider** | `internal/config/provider.go` | Provider 加载和更新 |
| **协调器** | `internal/agent/coordinator.go` | Agent 管理和工具构建 |
| **Agent** | `internal/agent/agent.go` | Agent 核心逻辑 |
| **权限** | `internal/permission/permission.go` | 权限检查和批准 |

### 8.2 关键函数

| 函数 | 位置 | 功能 |
|------|------|------|
| `Load()` | `load.go:33` | 加载配置文件 |
| `Providers()` | `provider.go:142` | 加载 Provider 列表 |
| `SetupAgents()` | `config.go:731` | 初始化 Agent 配置 |
| `NewCoordinator()` | `coordinator.go:75` | 创建协调器 |
| `buildAgent()` | `coordinator.go:319` | 构建 Agent |
| `buildTools()` | `coordinator.go:360` | 构建工具列表 |
| `Request()` | `permission.go:132` | 权限检查 |

### 8.3 提示词模板

| 模板 | 位置 | Agent |
|------|------|-------|
| `coder.md.tpl` | `internal/agent/templates/` | Coder Agent |
| `task.md.tpl` | `internal/agent/templates/` | Task Agent |
| `agentic_fetch_prompt.md.tpl` | `internal/agent/templates/` | Fetch Agent |

---

## 9. 常见问题

### Q1: 为什么不配置也能看到 Provider？

**A**: 因为代码中有**内置 Provider**（embedded），编译时打包在二进制文件中。

### Q2: 同时配置内置和自定义 Provider，会怎样？

**A**: **合并**，用户配置优先级更高：
- 用户配置的参数覆盖内置的
- 用户未配置的参数使用内置的

### Q3: 禁用工具会减少系统提示词吗？

**A**: **是的**！禁用工具会完全移除其描述，减少 30-80% 的 tokens。

### Q4: Task Agent 的工具有什么限制？

**A**:
- 只有只读工具：`glob`, `grep`, `ls`, `sourcegraph`, `view`
- 不能编辑文件或执行命令
- 禁用 MCP 和 LSP

### Q5: 如何实现只读模式？

**A**:
```json
{
  "options": {
    "disabled_tools": ["bash", "edit", "multiedit", "write"]
  },
  "permissions": {
    "allowed_tools": ["view", "ls", "glob", "grep"]
  }
}
```

---

## 10. 总结

### 核心设计理念

1. **Coder 中心制**：用户始终与 Coder Agent 交互
2. **按需创建子 Agent**：Task/Fetch Agent 动态创建和销毁
3. **权限分层控制**：工具级过滤 + 运行时权限检查
4. **智能配置合并**：用户配置 > 内置配置
5. **系统提示词优化**：禁用工具自动移除描述

### 关键优势

- ✅ **灵活的权限控制**：从 YOLO 模式到严格只读
- ✅ **高效的系统提示词**：禁用工具减少 tokens
- ✅ **智能的配置合并**：用户配置覆盖内置配置
- ✅ **强大的 Agent 协作**：Coder 委派给 Task/Fetch
- ✅ **完善的成本追踪**：父子 Session 关系

### 学习收获

通过本项目学习到：
1. 如何设计多 Agent 协作系统
2. 如何实现灵活的权限控制
3. 如何优化系统提示词大小
4. 如何设计配置合并机制
5. 如何管理会话和成本追踪

---

**文档版本**: v1.0
**最后更新**: 2025-01-24
**项目**: ZorkAgent (基于 Crush)
