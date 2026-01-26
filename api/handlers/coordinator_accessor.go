package handlers

import (
	"fmt"
	"log/slog"
	"reflect"
	"unsafe"

	"github.com/charmbracelet/crush/internal/agent"
)

// coordinatorAccessor 封装访问 coordinator 私有字段的逻辑。
// 使用反射和 unsafe 包来访问 internal/agent/coordinator 的私有字段。
//
// 警告：此实现依赖于 internal/agent/coordinator 的内部实现细节。
// 如果 coordinator 结构体的字段名或类型改变，此代码可能失效。
type coordinatorAccessor struct{}

// newCoordinatorAccessor 创建一个新的 coordinatorAccessor 实例
func newCoordinatorAccessor() *coordinatorAccessor {
	return &coordinatorAccessor{}
}

// getSessionAgent 通过反射和 unsafe 访问 coordinator 的 currentAgent 字段
//
// 参数:
//   - coord: agent.Coordinator 接口实例（实际是 *coordinator 类型）
//
// 返回:
//   - agent.SessionAgent: coordinator 内部的 currentAgent
//   - error: 如果访问失败，返回详细错误信息
func (a *coordinatorAccessor) getSessionAgent(coord agent.Coordinator) (agent.SessionAgent, error) {
	// 获取 coordinator 的反射值
	v := reflect.ValueOf(coord)
	if v.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("coordinator is not a pointer, got: %v", v.Kind())
	}

	// 解引用指针
	v = v.Elem()

	// 查找 currentAgent 字段
	currentAgentField := v.FieldByName("currentAgent")
	if !currentAgentField.IsValid() {
		return nil, fmt.Errorf("currentAgent field not found in coordinator struct. " +
			"This may indicate that the internal implementation has changed")
	}

	// 检查字段是否可以被导出（通常私有字段不能直接访问）
	if !currentAgentField.CanInterface() {
		// 使用 unsafe 获取未导出字段的值
		currentAgentPtr := unsafe.Pointer(currentAgentField.UnsafeAddr())
		currentAgentValue := reflect.NewAt(currentAgentField.Type(), currentAgentPtr).Elem()

		sessionAgent, ok := currentAgentValue.Interface().(agent.SessionAgent)
		if !ok {
			return nil, fmt.Errorf("currentAgent field does not implement SessionAgent interface, actual type: %T",
				currentAgentValue.Interface())
		}

		slog.Debug("Successfully accessed private currentAgent field using unsafe",
			"coordinator_type", v.Type(),
			"agent_type", currentAgentValue.Type())

		return sessionAgent, nil
	}

	// 如果字段可以被导出（不太可能，但处理这种情况）
	sessionAgent, ok := currentAgentField.Interface().(agent.SessionAgent)
	if !ok {
		return nil, fmt.Errorf("currentAgent field does not implement SessionAgent interface, actual type: %T",
			currentAgentField.Interface())
	}

	return sessionAgent, nil
}

// getSystemPrompt 获取 coordinator 的当前系统提示词
//
// 参数:
//   - coord: agent.Coordinator 接口实例（实际是 *coordinator 类型）
//
// 返回:
//   - string: 当前的系统提示词
//   - error: 如果访问失败，返回详细错误信息
func (a *coordinatorAccessor) getSystemPrompt(coord agent.Coordinator) (string, error) {
	// 获取 coordinator 的反射值
	v := reflect.ValueOf(coord)
	if v.Kind() != reflect.Ptr {
		return "", fmt.Errorf("coordinator is not a pointer, got: %v", v.Kind())
	}

	// 解引用指针
	v = v.Elem()

	// 查找 currentAgent 字段
	currentAgentField := v.FieldByName("currentAgent")
	if !currentAgentField.IsValid() {
		return "", fmt.Errorf("currentAgent field not found in coordinator struct. " +
			"This may indicate that the internal implementation has changed")
	}

	// currentAgent 是一个接口类型，我们需要获取其指向的实际值
	// 使用 unsafe 获取未导出字段的值
	currentAgentPtr := unsafe.Pointer(currentAgentField.UnsafeAddr())

	// 创建一个新的 reflect.Value 来指向实际的对象
	// 这里 currentAgent 是 *sessionAgent 类型的接口
	currentAgentValue := reflect.NewAt(currentAgentField.Type(), currentAgentPtr).Elem()

	// 现在当前 currentAgentValue 是具体的 *sessionAgent 类型
	// 我们需要再次解引用来获取 sessionAgent 本身
	if currentAgentValue.Kind() == reflect.Ptr {
		currentAgentValue = currentAgentValue.Elem()
	}

	// 查找 systemPrompt 字段
	systemPromptField := currentAgentValue.FieldByName("systemPrompt")
	if !systemPromptField.IsValid() {
		return "", fmt.Errorf("systemPrompt field not found in sessionAgent struct. " +
			"This may indicate that the internal implementation has changed")
	}

	// systemPrompt 是 *csync.Value[string] 类型
	// 我们需要调用它的 Get() 方法
	if !systemPromptField.CanInterface() {
		// 使用 unsafe 获取未导出字段的值
		systemPromptPtr := unsafe.Pointer(systemPromptField.UnsafeAddr())
		systemPromptValue := reflect.NewAt(systemPromptField.Type(), systemPromptPtr).Elem()

		// 调用 Get() 方法
		getMethod := systemPromptValue.MethodByName("Get")
		if !getMethod.IsValid() {
			return "", fmt.Errorf("Get() method not found on systemPrompt field")
		}

		results := getMethod.Call(nil)
		if len(results) != 1 {
			return "", fmt.Errorf("Get() method returned unexpected number of results")
		}

		systemPrompt, ok := results[0].Interface().(string)
		if !ok {
			return "", fmt.Errorf("Get() method did not return a string")
		}

		slog.Debug("Successfully retrieved system prompt using unsafe",
			"prompt_length", len(systemPrompt))

		return systemPrompt, nil
	}

	// 如果字段可以被导出（不太可能）
	systemPromptValue := systemPromptField.Interface()
	return "", fmt.Errorf("unexpected: systemPrompt field is exported, type: %T", systemPromptValue)
}
