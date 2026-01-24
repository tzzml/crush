# OpsAgent 使用说明

## 配置文件位置

OpsAgent 配置已经创建完成，使用**项目级配置**方式：

- **项目配置**：[`.crush.json`](.crush.json)（仅影响 zorkagent 项目）
- **全局配置**：`~/.config/crush/crush.json`（默认配置，影响所有项目）
- **Ops 规则**：`~/.config/crush/ops-rules`（补充规则）
- **Kafka Skill**：`~/.config/crush/skills/kafka-ops/SKILL.md`

## 使用方式

### 方式 1：在 zorkagent 项目中使用 OpsAgent（当前配置）

```bash
# 在 zorkagent 项目目录中
cd /Users/zhuminglei/Projects/zorkagent
crush

# Agent 会自动使用 Ops 模式（因为项目根目录有 .crush.json）
```

**特点**：
- ✅ 只影响 zorkagent 项目
- ✅ 其他项目使用默认 CoderAgent
- ✅ 删除 `.crush.json` 即可恢复默认模式

### 方式 2：在其他项目中使用 OpsAgent

有两种方式：

#### 方式 2a：复制项目配置

```bash
# 复制 Ops 配置到目标项目
cp /Users/zhuminglei/Projects/zorkagent/.crush.json /path/to/other-project/.crush.json

# 在目标项目目录中启动
cd /path/to/other-project
crush
```

#### 方式 2b：切换到全局 Ops 模式

```bash
# 备份当前全局配置
cp ~/.config/crush/crush.json ~/.config/crush/crush.json.bak

# 使用 Ops 配置作为全局配置
cp /Users/zhuminglei/Projects/zorkagent/.crush.json ~/.config/crush/crush.json

# 在任何项目中启动（都会使用 Ops 模式）
crush

# 恢复默认配置
mv ~/.config/crush/crush.json.bak ~/.config/crush/crush.json
```

## OpsAgent 功能特性

### ✅ 可用功能

- **查看文件**：`view`, `ls`, `grep`, `glob`
- **执行脚本**：`bash`（需要用户确认）
- **后台任务**：`job_output`, `job_kill`
- **下载文件**：`download`
- **使用 Skills**：自动发现并使用运维 Skills

### ❌ 不可用功能

- **编辑代码**：`edit`, `multiedit`, `write`
- **子 Agent**：`agent`, `agentic_fetch`

## 权限配置

自动批准（无需用户确认）：
- `view` - 查看文件内容
- `ls` - 列出目录
- `grep` - 搜索文件内容
- `glob` - 查找文件

需要用户确认：
- `bash` - 执行命令（危险操作）

## 测试 OpsAgent

### 测试 1：查看日志（应该成功）

```bash
cd /Users/zhuminglei/Projects/zorkagent
crush

# 在 Agent 中输入：
# "查看最近的 git commit 日志"
```

预期结果：Agent 应该能使用 `grep` 和 `view` 工具查看日志。

### 测试 2：使用 Kafka Skill（应该成功）

```bash
crush

# 在 Agent 中输入：
# "如何查看 Kafka consumer lag？"
```

预期结果：Agent 应该：
1. 发现 `kafka-ops` Skill
2. 读取 Skill 工作流程
3. 执行相应的 Kafka 命令

### 测试 3：尝试编辑代码（应该失败）

```bash
crush

# 在 Agent 中输入：
# "修复这个 bug"
```

预期结果：Agent 应该报错，说明编辑工具不可用。

## 切换回默认模式

如果需要在 zorkagent 项目中临时使用默认 CoderAgent：

```bash
# 方式 1：重命名项目配置
mv .crush.json .crush.json.ops
crush  # 使用默认模式
mv .crush.json.ops .crush.json  # 恢复 Ops 模式

# 方式 2：在项目目录外启动
cd /tmp
crush -c /Users/zhuminglei/Projects/zorkagent
```

## 配置文件内容说明

### `.crush.json`（项目配置）

```json
{
  "$schema": "https://charm.sh/crush.json",

  // Provider 配置：注入 Ops 专用提示词
  "providers": {
    "anthropic": {
      "api_key": "$ANTHROPIC_API_KEY",
      "system_prompt_prefix": "你现在是运维 Agent（Ops），专注于运维任务...\n"
    }
  },

  // 禁用编辑工具
  "options": {
    "disabled_tools": ["edit", "multiedit", "write", "agent", "agentic_fetch"],
    "context_paths": [".ops-rules", "AGENTS.md", ".cursorrules"],
    "skills_paths": ["~/.config/crush/skills", "./skills"]
  },

  // 自动批准安全工具
  "permissions": {
    "allowed_tools": ["view", "ls", "grep", "glob"]
  },

  // 模型配置
  "models": {
    "large": {
      "model": "claude-sonnet-4-20250514",
      "provider": "anthropic"
    }
  }
}
```

## 添加更多 Skills

参考 Kafka Skill 示例，创建更多运维 Skills：

```bash
# 1. 创建 Skill 目录
mkdir -p ~/.config/crush/skills/<skill-name>

# 2. 创建 SKILL.md
cat > ~/.config/crush/skills/<skill-name>/SKILL.md << 'EOF'
---
name: <skill-name>
description: <简短描述>
---

# <技能名称> 工作流程

## 概述
<技能用途说明>

## 工作流程
<详细步骤>

## 常用命令
```bash
<命令示例>
```
EOF
```

已有的 Skills：
- `~/.config/crush/skills/elasticsearch-analyzer/` - Elasticsearch 分析
- `~/.config/crush/skills/kafka-ops/` - Kafka 运维（新增）

## 故障排查

### 问题：Agent 没有使用 Ops 模式

**检查**：
```bash
# 确认项目根目录有 .crush.json
ls -la .crush.json

# 确认配置内容正确
cat .crush.json | jq .
```

### 问题：Skills 没有被加载

**检查**：
```bash
# 确认 Skills 目录存在
ls -la ~/.config/crush/skills/

# 确认 SKILL.md 文件存在
ls -la ~/.config/crush/skills/*/SKILL.md
```

### 问题：编辑工具仍然可用

**检查**：
```bash
# 确认 disabled_tools 配置正确
cat .crush.json | jq .options.disabled_tools
```

应该输出：`["edit", "multiedit", "write", "agent", "agentic_fetch"]`

## 总结

OpsAgent 通过**配置文件**实现，无需修改代码：

1. ✅ **项目级配置**：`.crush.json`（仅影响当前项目）
2. ✅ **全局配置**：`~/.config/crush/crush.json`（影响所有项目）
3. ✅ **Skills 系统**：自动发现并使用运维 Skills
4. ✅ **工具权限控制**：精确控制可用工具
5. ✅ **灵活切换**：通过重命名或删除配置文件即可切换模式
