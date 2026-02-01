# 系统提示词动态更新 API

## 概述

本 API 允许您在运行时动态修改系统提示词（system prompt），无需重启服务。更新后立即对后续 AI 对话生效。

**重要说明**: 获取/设置的是**纯的 coder agent 提示词**（来自模板文件），不包含 provider 的 `system_prompt_prefix`。Provider prefix 会在运行时自动添加。

## API 端点

### 1. 获取系统提示词

```
GET /system-prompt?directory=<project_path>
```

**参数:**
- `directory` (query, required): 项目路径

**成功响应** (200 OK):
```json
{
  "system_prompt": "You are an expert Go developer...",
  "length": 1234,
  "is_custom": true
}
```

**字段说明:**
- `system_prompt`: 当前的系统提示词内容
- `length`: 提示词字符数
- `is_custom`: 是否为自定义提示词（非空表示已自定义）

**使用示例:**
```bash
curl -X GET "http://localhost:8080/system-prompt?directory=/path/to/project"
```

### 2. 更新系统提示词

```
PUT /system-prompt?directory=<project_path>
```

**参数:**
- `directory` (query, required): 项目路径

**请求体:**
```json
{
  "system_prompt": "You are an expert Go developer. Always write clean code."
}
```

**成功响应** (200 OK):
```json
{
  "success": true,
  "system_prompt": "You are an expert Go developer...",
  "message": "System prompt updated successfully"
}
```

**错误响应示例:**

缺少 directory 参数 (400):
```json
{
  "error_code": "MISSING_DIRECTORY_PARAM",
  "message": "Directory query parameter is required"
}
```

项目不存在 (404):
```json
{
  "error_code": "PROJECT_NOT_FOUND",
  "message": "Project not found: /invalid/path"
}
```

空提示词 (400):
```json
{
  "error_code": "EMPTY_SYSTEM_PROMPT",
  "message": "System prompt cannot be empty or whitespace only"
}
```

## 使用示例

### 获取当前提示词

```bash
curl -X GET "http://localhost:8080/system-prompt?directory=/path/to/your/project"
```

### 更新提示词

```bash
curl -X PUT "http://localhost:8080/system-prompt?directory=/path/to/your/project" \
  -H "Content-Type: application/json" \
  -d '{
    "system_prompt": "You are a helpful assistant. Always respond in Chinese."
  }'
```

### 先获取再更新

```bash
# 1. 获取当前提示词
CURRENT=$(curl -s "http://localhost:8080/system-prompt?directory=/my/project" | jq -r '.system_prompt')

# 2. 查看当前提示词
echo "当前提示词: $CURRENT"

# 3. 更新为新提示词
curl -X PUT "http://localhost:8080/system-prompt?directory=/my/project" \
  -H "Content-Type: application/json" \
  -d '{
    "system_prompt": "You are an expert Go developer. Focus on code quality."
  }'
```

### 使用 Go

```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
)

// 获取系统提示词
func getSystemPrompt(projectPath string) (string, error) {
    resp, err := http.Get(
        fmt.Sprintf("http://localhost:8080/system-prompt?directory=%s", projectPath),
    )
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    var result struct {
        SystemPrompt string `json:"system_prompt"`
        Length       int    `json:"length"`
        IsCustom     bool   `json:"is_custom"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return "", err
    }

    return result.SystemPrompt, nil
}

// 更新系统提示词
func updateSystemPrompt(projectPath, prompt string) error {
    data := map[string]string{"system_prompt": prompt}
    jsonData, _ := json.Marshal(data)

    req, _ := http.NewRequest(
        "PUT",
        fmt.Sprintf("http://localhost:8080/system-prompt?directory=%s", projectPath),
        bytes.NewBuffer(jsonData),
    )
    req.Header.Set("Content-Type", "application/json")

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
    }

    return nil
}

func main() {
    projectPath := "/path/to/project"

    // 获取当前提示词
    current, _ := getSystemPrompt(projectPath)
    fmt.Printf("当前提示词: %s\n", current)

    // 更新提示词
    newPrompt := "You are an expert Go developer."
    updateSystemPrompt(projectPath, newPrompt)

    // 再次获取验证
    updated, _ := getSystemPrompt(projectPath)
    fmt.Printf("更新后提示词: %s\n", updated)
}
```

### 使用 JavaScript

```javascript
// 获取系统提示词
async function getSystemPrompt(projectPath) {
  const response = await fetch(
    `http://localhost:8080/system-prompt?directory=${projectPath}`
  );
  const data = await response.json();
  console.log('当前提示词:', data.system_prompt);
  console.log('长度:', data.length);
  console.log('是否自定义:', data.is_custom);
  return data.system_prompt;
}

// 更新系统提示词
async function updateSystemPrompt(projectPath, prompt) {
  const response = await fetch(
    `http://localhost:8080/system-prompt?directory=${projectPath}`,
    {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ system_prompt: prompt })
    }
  );
  const data = await response.json();
  console.log('更新成功:', data.success);
  return data;
}

// 使用示例
(async () => {
  const projectPath = '/path/to/project';

  // 获取当前提示词
  const current = await getSystemPrompt(projectPath);

  // 更新提示词
  await updateSystemPrompt(
    projectPath,
    'You are a helpful assistant. Always respond in Chinese.'
  );

  // 验证更新
  const updated = await getSystemPrompt(projectPath);
  console.log('已更新');
})();
```

## 功能特性

### ✅ 运行时生效
- 更新后立即对后续对话生效
- 无需重启服务
- 不影响正在进行的任务

### ✅ 项目隔离
- 每个项目有独立的系统提示词
- 通过 `directory` 参数指定项目
- 不同项目的提示词互不干扰

### ✅ 线程安全
- 使用线程安全的存储机制
- 支持并发读取和更新
- 无需担心竞态条件

### ✅ 读取功能
- 可以获取当前提示词内容
- 查看提示词长度和是否自定义
- 方便验证更新是否成功

## 使用场景

### 场景 1: 临时修改 AI 行为

```bash
# 让 AI 临时只输出代码，不做解释
curl -X PUT "http://localhost:8080/system-prompt?directory=/my/project" \
  -H "Content-Type: application/json" \
  -d '{"system_prompt": "Output only code. No explanations."}'
```

### 场景 2: 切换语言

```bash
# 让 AI 使用中文回复
curl -X PUT "http://localhost:8080/system-prompt?directory=/my/project" \
  -H "Content-Type: application/json" \
  -d '{"system_prompt": "请始终使用中文回复。"}'
```

### 场景 3: 特定任务定制

```bash
# 让 AI 专注于代码审查
curl -X PUT "http://localhost:8080/system-prompt?directory=/my/project" \
  -H "Content-Type: application/json" \
  -d '{
    "system_prompt": "You are a code reviewer. Focus on: 1) Security, 2) Performance, 3) Code quality"
  }'
```

### 场景 4: 查看当前配置

```bash
# 先查看当前提示词
curl -X GET "http://localhost:8080/system-prompt?directory=/my/project"

# 再决定是否需要修改
```

## 提示词说明

### 纯的 Coder Agent 提示词

获取和设置的是**纯的 coder agent 模板提示词**，来自 `internal/agent/templates/coder.md.tpl`。

**运行时组合**:
在实际使用时，系统会自动将以下两部分组合：
1. **Provider Prefix** - 从 provider 配置的 `system_prompt_prefix`
2. **Coder Agent Prompt** - 我们通过 API 获取/设置的部分

**组合顺序**:
```go
// 伪代码
finalPrompt = providerPrefix + coderAgentPrompt
messages = [
  {role: "system", content: finalPrompt},
  // ... 其他消息
]
```

**示例**:
```json
// Provider Prefix (配置文件中)
"You are using OpenAI GPT-4 model. "

// Coder Agent Prompt (通过 API 获取/设置)
"You are an expert Go developer. Always write clean code."

// 最终发送给模型的消息
"You are using OpenAI GPT-4 model. You are an expert Go developer. Always write clean code."
```

### 为什么这样设计？

1. **灵活性** - 可以为不同 provider 设置不同的 prefix
2. **隔离性** - Coder agent 提示词保持纯粹，不依赖 provider
3. **可控性** - 用户可以完全控制 coder agent 的行为
4. **兼容性** - 保持与现有配置系统的兼容

## 测试

项目提供了完整的测试脚本：

```bash
# 1. 启动 API 服务器
./zorkagent server --port 8080

# 2. 在另一个终端运行测试
./docs/test_system_prompt_api.sh
```

测试脚本会验证：
- ✅ 获取系统提示词
- ✅ 更新系统提示词
- ✅ 验证更新成功
- ✅ 错误处理（缺少参数、空提示词、无效路径）
- ✅ 项目隔离
- ✅ 多次更新功能

## 限制与注意事项

### ⚠️ 功能限制

1. **无持久化**
   - 重启服务后会恢复为默认提示词
   - 默认提示词来自模板文件 `internal/agent/templates/coder.md.tpl`
   - 需要通过脚本或配置管理工具自动化

2. **无历史记录**
   - 不保存提示词的修改历史
   - 无法回滚到之前的提示词
   - 建议在外部维护提示词版本

3. **无内容验证**
   - 不验证提示词的合法性
   - 请确保提示词格式正确
   - 注意模型的上下文窗口限制

### ⚠️ 使用注意事项

1. **提示词长度**
   - 注意模型的上下文窗口限制
   - 过长的提示词可能影响性能
   - 建议保持在合理范围内

2. **格式要求**
   - 确保提示词符合模型要求
   - 避免特殊字符导致的解析问题
   - 使用清晰的指令

3. **生效范围**
   - 只影响指定项目
   - 不影响其他项目
   - 需要通过 directory 参数指定

4. **重启失效**
   - 服务重启后需要重新设置
   - 建议通过脚本或配置管理工具自动化
   - 考虑使用数据库或配置文件持久化

### ⚠️ 维护注意事项

1. **版本升级**
   - 升级 internal 包时需要测试此功能
   - 依赖内部实现细节
   - 检查字段名是否改变

2. **错误监控**
   - 监控 `COORDINATOR_ACCESS_FAILED` 错误
   - 及时发现内部实现变更
   - 设置告警机制

3. **实现依赖**
   - 使用反射和 unsafe 包访问私有字段
   - 如果 `coordinator` 内部实现改变，可能需要更新代码
   - 关注 internal/agent 的更新

## 技术实现

### 获取提示词路径

```
HTTP Request (GET)
  ↓
HandleGetSystemPrompt (Handler)
  ↓
GetAppForProject (获取 App 实例)
  ↓
appInstance.AgentCoordinator (获取 Coordinator)
  ↓
coordinatorAccessor.getSystemPrompt (反射访问私有字段)
  ↓
systemPrompt.Get() (获取值)
```

### 更新提示词路径

```
HTTP Request (PUT)
  ↓
HandleUpdateSystemPrompt (Handler)
  ↓
GetAppForProject (获取 App 实例)
  ↓
appInstance.AgentCoordinator (获取 Coordinator)
  ↓
coordinatorAccessor.getSessionAgent (反射获取 SessionAgent)
  ↓
sessionAgent.SetSystemPrompt (设置新提示词)
  ↓
csync.Value.Set (线程安全存储)
```

### 下次对话执行路径

```
AgentCoordinator.Run
  ↓
sessionAgent.Run
  ↓
获取 promptPrefix (from systemPromptPrefix)
  ↓
获取 systemPrompt (from systemPrompt)
  ↓
组合: prefix + systemPrompt
  ↓
fantasy.NewAgent(WithSystemPrompt)
```

### 关键文件

- `api/handlers/system_prompt.go` - HTTP Handlers (GET/PUT)
- `api/handlers/coordinator_accessor.go` - 反射访问器
- `api/models/system_prompt.go` - 数据模型
- `api/server.go` - 路由注册

## 故障排除

### 问题：更新后提示词没有生效

**解决方案：**
1. 确认是否在正确的项目中更新
2. 检查是否有缓存导致使用了旧提示词
3. 尝试重新创建会话
4. 通过 GET 端点验证当前提示词

### 问题：返回 COORDINATOR_ACCESS_FAILED 错误

**解决方案：**
1. 检查 internal 包版本是否更新
2. 查看日志了解具体错误信息
3. 可能需要更新反射访问器代码
4. 验证字段名是否改变

### 问题：获取的提示词与预期不符

**解决方案：**
1. 确认获取的是 coder agent 提示词（不包含 provider prefix）
2. Provider prefix 会在运行时自动添加
3. 检查 provider 配置中的 `system_prompt_prefix`
4. 查看完整的组合提示词（需要查看日志或消息）

### 问题：编译失败

**解决方案：**
1. 确保所有新文件都已创建
2. 检查 import 路径是否正确
3. 运行 `go mod tidy` 更新依赖
4. 检查 Go 版本是否兼容

## 开发计划

### 未来可能的改进

- [x] 添加 GET 端点获取当前提示词
- [ ] 添加提示词持久化功能
- [ ] 添加提示词版本管理
- [ ] 添加提示词模板库
- [ ] 添加验证规则（长度、格式等）
- [ ] 支持批量更新多个项目
- [ ] 支持提示词 A/B 测试
- [ ] 提供 Web UI 管理界面

## 贡献

如果您发现任何问题或有改进建议，欢迎：
1. 提交 Issue
2. 创建 Pull Request
3. 联系维护者

## 许可证

本项目遵循主项目的许可证。
