# Crush API 测试方案

本通过档描述了 Crush API 的测试策略、环境准备及执行步骤，旨在确保 API 的功能完整性和稳定性。

## 1. 测试目标

验证 Crush API (Flat Structure) 的以下核心功能：
- **项目管理**：项目注册、配置读取、权限检查。
- **会话管理**：会话的创建、列表获取、元数据更新。
- **消息交互**：发送消息、接收响应、流式传输支持。
- **事件系统**：SSE (Server-Sent Events) 的连接与实时推送。
- **系统状态**：健康检查、服务状态。

## 2. 环境准备

### 2.1 系统要求
- **OS**: macOS (当前环境)
- **Runtime**: Python 3.8+
- **Server**: Crush (Go implementation)

### 2.2 依赖安装
测试脚本依赖 `requests` 和 `sseclient-py` 库：

```bash
pip install requests sseclient-py
```

### 2.3 测试工具
位于 `docs/` 目录下的测试脚本：
- `test_api.py`: 完整的 API 功能自动化测试脚本。
- `test_sse.py`: 专门针对 SSE 事件推送的独立测试脚本。
- `test_client.py`: 模拟真实客户端行为的演示脚本。
- `test_all.sh`: 一键运行所有测试的编排脚本。

## 3. 测试范围与用例

### 3.1 基础功能测试 (`test_api.py`)
| 模块 | 测试点 | 预期结果 |
|------|--------|----------|
| **项目** | `GET /project` | 返回项目列表 |
| **项目** | `POST /project` | 成功注册新项目 |
| **会话** | `GET /session` | 返回指定项目的会话列表 |
| **会话** | `POST /session` | 成功创建新会话 |
| **会话** | `PUT /session/{id}` | 成功更新会话标题 |
| **消息** | `POST /session/{id}/message` | 消息发送成功，返回模型响应 |
| **配置** | `GET /project/config` | 返回工作目录及配置状态 |
| **权限** | `GET /project/permissions` | 返回权限状态 |

### 3.2 实时流测试 (`test_sse.py`)
| 模块 | 测试点 | 预期结果 |
|------|--------|----------|
| **SSE** | `GET /event` | 建立长连接，并在操作触发时接收到 JSON 格式事件 |

## 4. 执行步骤

### 步骤 1: 启动服务器
在项目根目录下编译并启动服务器：

```bash
go build -o crush .
./crush serve
# 或者使用默认端口 8080
```

### 步骤 2: 运行自动化回归测试
运行综合测试脚本，它会自动检测环境、启动/检查服务器（如果集成在脚本中）并运行 Python 测试用例：

```bash
# 推荐方式
cd docs
./test_all.sh
```

或者单独运行：

```bash
python3 docs/test_api.py
```

### 步骤 3: 验证 SSE 实时性
在一个终端窗口运行 SSE 监听：

```bash
python3 docs/test_sse.py
```

### 步骤 4: 模拟客户端交互
运行演示脚本以验证从用户角度的 API 体验：

```bash
python3 docs/test_client.py
```

## 5. 验收标准

1.  **自动化测试通过**：`test_api.py` 所有断言通过，无 HTTP 4xx/5xx 错误（预期外的）。
2.  **SSE 连接稳定**：在测试期间 SSE 连接不中断，且能准确收到 `message`、`session` 等事件。
3.  **文档一致性**：测试行为与 `docs/API.md` 描述的接口定义保持一致。

## 6. 问题排查

如果测试失败，请检查：
1.  **端口冲突**：确保 8080 端口未被占用。
2.  **服务器日志**：查看 `crush serve` 的输出日志，寻找 Panic 或 Error 信息。
3.  **依赖版本**：确保 `sseclient-py` 已正确安装。
