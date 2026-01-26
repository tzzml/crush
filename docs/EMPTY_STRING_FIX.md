# GET /system-prompt 空字符串问题修复

## 问题描述

GET /system-prompt 端点返回 200 OK，但 `system_prompt` 字段为空字符串 `""`。

## 根本原因

`sessionAgent` 的实际类型是 `*sessionAgent`（指向结构体的指针），而不是 `sessionAgent`（结构体本身）。

### 代码证据

在 [internal/agent/agent.go:127-144](../internal/agent/agent.go#L127-L144) 中：

```go
func NewSessionAgent(
	opts SessionAgentOptions,
) SessionAgent {
	return &sessionAgent{  // ← 返回指针 &sessionAgent{}
		largeModel:           csync.NewValue(opts.LargeModel),
		smallModel:           csync.NewValue(opts.SmallModel),
		systemPromptPrefix:   csync.NewValue(opts.SystemPromptPrefix),
		systemPrompt:         csync.NewValue(opts.SystemPrompt),
		// ...
	}
}
```

- `NewSessionAgent` 返回 `&sessionAgent{...}` （指针）
- 接口 `SessionAgent` 持有的是 `*sessionAgent` 类型的值
- 当我们通过反射访问时，`reflect.ValueOf(sessionAgent)` 返回的是指针类型

## 修复方案

在调用 `FieldByName("systemPrompt")` 之前，先检查并解引用指针：

```go
sessionAgentValue := reflect.ValueOf(sessionAgent)

// 如果是指针，需要解引用
if sessionAgentValue.Kind() == reflect.Ptr {
    sessionAgentValue = sessionAgentValue.Elem()
}

// 现在可以安全地访问字段
systemPromptField := sessionAgentValue.FieldByName("systemPrompt")
```

## 修改文件

- [api/handlers/coordinator_accessor.go](../api/handlers/coordinator_accessor.go#L103-L108)

## 测试方法

### 1. 重新编译

```bash
go build -o /tmp/zorkagent-test .
```

### 2. 重启服务

```bash
./zorkagent server --port 8080
```

### 3. 测试 GET 端点

```bash
curl -X GET "http://localhost:8080/system-prompt?directory=/path/to/project"
```

### 4. 预期结果

```json
{
  "system_prompt": "You are an expert Go developer...",
  "length": 1234,
  "is_custom": true
}
```

其中 `system_prompt` 应该包含实际的提示词内容，`length` 应该大于 0。

## 调试日志

修复后的代码会输出以下调试日志：

```
DEBUG Got sessionAgent from getSessionAgent
  type=*agent.sessionAgent
  kind=ptr
DEBUG Dereferenced pointer
  type=agent.sessionAgent
  kind=struct
DEBUG Found systemPrompt field
  type=*csync.Value[string]
  kind=ptr
DEBUG Successfully retrieved system prompt
  prompt_length=1234
  prompt_preview=You are an expert Go developer...
```

这些日志可以帮助验证：
1. ✅ 正确识别了是指针类型 (`kind=ptr`)
2. ✅ 成功解引用 (`Dereferenced pointer`)
3. ✅ 找到了 `systemPrompt` 字段
4. ✅ 成功调用 `Get()` 方法获取值

## 相关知识点

### Go 反射：指针 vs 值

- `reflect.ValueOf(ptr)` 返回一个 `reflect.Value`，其 `Kind()` 是 `reflect.Ptr`
- 必须调用 `.Elem()` 来解引用指针，获取指向的值
- 解引用后才能调用 `.FieldByName()` 访问结构体字段

### 接口中存储的指针

当一个接口持有指针类型的值时：
- 接口的动态类型是 `*T`（指向 T 的指针）
- 通过反射获取时，`reflect.Value.Interface()` 返回的值仍然是指针
- 必须先解引用才能访问结构体字段

## 总结

这个修复解决了一个常见的 Go 反射陷阱：**接口中存储的指针需要先解引用才能访问其字段**。

修复后的代码能够正确处理 `*sessionAgent` 类型，成功提取出 `systemPrompt` 字段的值。
