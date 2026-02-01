# 系统提示词动态更新功能实现总结

## 实现概述

成功实现了通过 API 在运行时动态修改系统提示词的功能，**无需修改 `internal/` 包下的任何代码**。

## 实现方案

采用 **Reflection + Unsafe** 方案，通过反射访问 `coordinator` 的私有字段 `currentAgent`，然后调用其 `SetSystemPrompt` 方法。

## 新增文件

### 1. API Handler - 反射访问器
**文件**: `api/handlers/coordinator_accessor.go`

封装使用反射和 unsafe 包访问 coordinator 私有字段的逻辑。

关键方法:
- `getSessionAgent(coord agent.Coordinator) (agent.SessionAgent, error)`

### 2. 数据模型
**文件**: `api/models/system_prompt.go`

定义请求和响应结构:
- `UpdateSystemPromptRequest` - 请求体
- `UpdateSystemPromptResponse` - 响应体

### 3. HTTP Handler
**文件**: `api/handlers/system_prompt.go`

处理系统提示词更新请求:
- 验证项目路径和请求参数
- 通过反射访问器获取 SessionAgent
- 调用 SetSystemPrompt 更新提示词
- 返回操作结果

### 4. 文档和测试
- `docs/SYSTEM_PROMPT_API.md` - 完整的 API 使用文档
- `docs/test_system_prompt_api.sh` - 自动化测试脚本

## 修改文件

### API Server - 路由注册
**文件**: `api/server.go`

在路由注册部分添加:
```go
// 系统提示词管理
s.PUT("/system-prompt", s.handlers.HandleUpdateSystemPrompt)
```

## API 使用示例

```bash
# 更新系统提示词
curl -X PUT "http://localhost:8080/system-prompt?directory=/path/to/project" \
  -H "Content-Type: application/json" \
  -d '{
    "system_prompt": "You are an expert Go developer. Always write clean code."
  }'
```

## 功能特性

✅ **不修改 internal 包** - 所有代码在 `api/` 目录
✅ **运行时生效** - 更新后立即对后续对话生效
✅ **线程安全** - 利用现有的 `csync.Value` 机制
✅ **项目隔离** - 不同项目的提示词互不干扰
✅ **错误处理** - 完善的参数验证和错误响应
✅ **文档完善** - 包含使用文档和测试脚本

## 测试验证

运行测试脚本验证功能:

```bash
# 1. 启动 API 服务器
./zorkagent server --port 8080

# 2. 运行测试脚本
./docs/test_system_prompt_api.sh
```

测试覆盖:
- ✅ 正常更新系统提示词
- ✅ 缺少 directory 参数（400 错误）
- ✅ 空提示词（400 错误）
- ✅ 无效项目路径（404 错误）
- ✅ 多次更新功能

## 技术亮点

### 1. 反射访问私有字段

使用 Go 的反射和 unsafe 包访问 `coordinator` 的私有字段:

```go
v := reflect.ValueOf(coord).Elem()
currentAgentField := v.FieldByName("currentAgent")
currentAgentPtr := unsafe.Pointer(currentAgentField.UnsafeAddr())
sessionAgent := reflect.NewAt(currentAgentField.Type(), currentAgentPtr).Elem()
```

### 2. 线程安全存储

`systemPrompt` 使用 `*csync.Value[string]` 存储，确保线程安全:

```go
func (a *sessionAgent) SetSystemPrompt(systemPrompt string) {
    a.systemPrompt.Set(systemPrompt)
}
```

### 3. 运行时生效

每次调用 `Agent.Run()` 时都会获取最新的 systemPrompt:

```go
systemPrompt := a.systemPrompt.Get()  // 每次运行时获取最新值
agent := fantasy.NewAgent(
    largeModel.Model,
    fantasy.WithSystemPrompt(systemPrompt),
)
```

## 已知限制

1. **无法读取当前提示词** - 只能设置，不能获取
2. **无持久化** - 重启服务后恢复为默认提示词
3. **实现依赖** - 依赖 `coordinator` 内部字段名不变

## 维护建议

1. **版本升级测试** - 升级 internal 包时测试此功能
2. **错误监控** - 监控 `COORDINATOR_ACCESS_FAILED` 错误
3. **文档更新** - 内部实现改变时更新相关文档

## 关键代码路径

```
HTTP Request
  → api/handlers/system_prompt.go:HandleUpdateSystemPrompt
    → api/handlers/app_manager.go:GetAppForProject
      → internal/app/app.go:AgentCoordinator
        → api/handlers/coordinator_accessor.go:getSessionAgent
          → internal/agent/agent.go:SetSystemPrompt
            → internal/csync:value.Set
```

## 下次对话执行路径

```
AgentCoordinator.Run
  → sessionAgent.Run
    → systemPrompt.Get() (获取最新值)
      → fantasy.NewAgent(WithSystemPrompt)
```

## 总结

此实现成功满足了用户需求：
- ✅ 不修改 internal 包代码
- ✅ 通过 API 访问
- ✅ 运行时生效
- ✅ 线程安全
- ✅ 向下兼容

通过巧妙使用反射和 unsafe 包，我们在不修改核心代码的前提下，实现了灵活的系统提示词动态更新功能。
