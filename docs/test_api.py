#!/usr/bin/env python3
"""
Crush REST API 测试脚本

使用方法:
    python3 test_api.py [--base-url BASE_URL] [--project-path PROJECT_PATH]

示例:
    # 使用默认配置
    python3 test_api.py

    # 指定 API 服务器地址
    python3 test_api.py --base-url http://localhost:3000

    # 指定项目路径
    python3 test_api.py --project-path /path/to/my/project
"""

import argparse
import json
import sys
import time
import urllib.parse
from typing import Dict, Any, Optional, Generator

try:
    import requests
    import sseclient
except ImportError as e:
    missing_lib = str(e).split("'")[1] if "'" in str(e) else "requests or sseclient"
    print(f"错误: 需要安装 {missing_lib} 库")
    print("请运行: pip install requests sseclient-py")
    sys.exit(1)


class CrushAPIClient:
    """Crush API 客户端"""

    def __init__(self, base_url: str = "http://localhost:8080/api/v1"):
        self.base_url = base_url.rstrip("/")
        self.session = requests.Session()
        self.session.headers.update({"Content-Type": "application/json"})

    def _request(
        self, method: str, endpoint: str, **kwargs
    ) -> requests.Response:
        """发送 HTTP 请求"""
        url = f"{self.base_url}{endpoint}"
        response = self.session.request(method, url, **kwargs)
        return response

    def _handle_response(self, response: requests.Response) -> Dict[str, Any]:
        """处理响应，检查错误"""
        try:
            data = response.json()
        except json.JSONDecodeError:
            print(f"错误: 无法解析 JSON 响应")
            print(f"状态码: {response.status_code}")
            print(f"响应内容: {response.text[:200]}")
            return {}

        if response.status_code >= 400:
            error = data.get("error", {})
            print(f"错误 [{error.get('code', 'UNKNOWN')}]: {error.get('message', 'Unknown error')}")
            return {}

        return data

    # Projects API

    def list_projects(self) -> list:
        """获取所有项目"""
        print("\n[1] 获取所有项目...")
        response = self._request("GET", "/projects")
        data = self._handle_response(response)
        projects = data.get("projects", [])
        print(f"找到 {len(projects)} 个项目")
        for project in projects:
            print(f"  - {project['path']} (数据目录: {project['data_dir']})")
        return projects

    def create_project(self, path: str, data_dir: Optional[str] = None) -> Optional[Dict]:
        """创建/注册项目"""
        print(f"\n[2] 创建项目: {path}")
        payload = {"path": path}
        if data_dir:
            payload["data_dir"] = data_dir

        response = self._request("POST", "/projects", json=payload)
        data = self._handle_response(response)
        project = data.get("project")
        if project:
            print(f"项目已创建/更新: {project['path']}")
            print(f"  数据目录: {project['data_dir']}")
            print(f"  最后访问: {project['last_accessed']}")
        return project

    def open_project(self, project_path: str) -> Optional[Dict]:
        """打开项目的 app 实例"""
        print(f"\n[2.5] 打开项目: {project_path}")
        encoded_path = urllib.parse.quote(project_path, safe="")
        response = self._request("POST", f"/projects/{encoded_path}/open", json={})
        data = self._handle_response(response)
        if data.get("status") == "opened":
            print(f"项目已打开: {data.get('project_path')}")
        return data

    def close_project(self, project_path: str) -> Optional[Dict]:
        """关闭项目的 app 实例"""
        print(f"\n[关闭] 关闭项目: {project_path}")
        encoded_path = urllib.parse.quote(project_path, safe="")
        response = self._request("POST", f"/projects/{encoded_path}/close", json={})
        data = self._handle_response(response)
        if data.get("status") == "closed":
            print(f"项目已关闭: {data.get('project_path')}")
        return data

    def connect_project(self, project_path: str) -> Optional[Dict]:
        """检查项目连接状态"""
        print(f"\n[连接] 检查项目状态: {project_path}")
        encoded_path = urllib.parse.quote(project_path, safe="")
        response = self._request("GET", f"/projects/{encoded_path}/connect")
        data = self._handle_response(response)
        is_open = data.get("is_open", False)
        print(f"项目状态: {'已打开' if is_open else '未打开'}")
        return data

    # Sessions API

    def list_sessions(self, project_path: str) -> list:
        """获取项目下的所有会话"""
        print(f"\n[3] 获取项目会话: {project_path}")
        encoded_path = urllib.parse.quote(project_path, safe="")
        response = self._request("GET", f"/projects/{encoded_path}/sessions")
        data = self._handle_response(response)
        sessions = data.get("sessions", [])
        print(f"找到 {len(sessions)} 个会话")
        for session in sessions[:5]:  # 只显示前5个
            print(f"  - [{session['id'][:8]}...] {session['title']} ({session['message_count']} 条消息)")
        return sessions

    def create_session(self, project_path: str, title: str) -> Optional[Dict]:
        """创建新会话"""
        print(f"\n[4] 创建会话: {title}")
        encoded_path = urllib.parse.quote(project_path, safe="")
        response = self._request(
            "POST",
            f"/projects/{encoded_path}/sessions",
            json={"title": title},
        )
        data = self._handle_response(response)
        session = data.get("session")
        if session:
            print(f"会话已创建: {session['id']}")
            print(f"  标题: {session['title']}")
        return session

    def get_session(self, project_path: str, session_id: str) -> Optional[Dict]:
        """获取单个会话详情"""
        print(f"\n[5] 获取会话详情: {session_id[:8]}...")
        encoded_path = urllib.parse.quote(project_path, safe="")
        response = self._request("GET", f"/projects/{encoded_path}/sessions/{session_id}")
        data = self._handle_response(response)
        session = data.get("session")
        if session:
            print(f"会话信息:")
            print(f"  标题: {session['title']}")
            print(f"  消息数: {session['message_count']}")
            print(f"  Token 使用: {session['prompt_tokens']} prompt + {session['completion_tokens']} completion")
            print(f"  成本: ${session['cost']:.6f}")
        return session

    def delete_session(self, project_path: str, session_id: str) -> bool:
        """删除会话"""
        print(f"\n[6] 删除会话: {session_id[:8]}...")
        encoded_path = urllib.parse.quote(project_path, safe="")
        response = self._request("DELETE", f"/projects/{encoded_path}/sessions/{session_id}")
        data = self._handle_response(response)
        if response.status_code == 200:
            print("会话已删除")
            return True
        return False

    # Messages API

    def list_messages(self, project_path: str, session_id: str) -> list:
        """获取会话的所有消息"""
        print(f"\n[7] 获取会话消息: {session_id[:8]}...")
        encoded_path = urllib.parse.quote(project_path, safe="")
        response = self._request("GET", f"/projects/{encoded_path}/sessions/{session_id}/messages")
        data = self._handle_response(response)
        messages = data.get("messages", [])
        print(f"找到 {len(messages)} 条消息")
        for msg in messages:
            role_icon = "👤" if msg["role"] == "user" else "🤖"
            content_preview = msg["content"][:50] + "..." if len(msg["content"]) > 50 else msg["content"]
            print(f"  {role_icon} [{msg['role']}]: {content_preview}")
        return messages

    def send_message_sync(self, project_path: str, session_id: str, prompt: str) -> Optional[Dict]:
        """发送消息（同步模式）"""
        print(f"\n[8] 发送消息（同步）: {prompt[:50]}...")
        print("等待 AI 响应...")
        start_time = time.time()

        encoded_path = urllib.parse.quote(project_path, safe="")
        response = self._request(
            "POST",
            f"/projects/{encoded_path}/sessions/{session_id}/messages",
            json={"prompt": prompt, "stream": False},
        )
        elapsed = time.time() - start_time
        data = self._handle_response(response)

        message = data.get("message")
        session = data.get("session")
        if message:
            print(f"✓ 响应完成 (耗时: {elapsed:.2f}秒)")
            print(f"消息内容: {message['content'][:200]}...")
            if session:
                print(f"Token 使用: {session['prompt_tokens']} prompt + {session['completion_tokens']} completion")
        return data

    def send_message_stream(
        self, project_path: str, session_id: str, prompt: str
    ) -> Generator[str, None, None]:
        """发送消息（流式模式）"""
        print(f"\n[9] 发送消息（流式）: {prompt[:50]}...")
        print("接收流式响应:")

        encoded_path = urllib.parse.quote(project_path, safe="")
        response = self._request(
            "POST",
            f"/projects/{encoded_path}/sessions/{session_id}/messages",
            json={"prompt": prompt, "stream": True},
            stream=True,
        )

        if response.status_code != 200:
            self._handle_response(response)
            return

        buffer = ""
        for line in response.iter_lines():
            if not line:
                continue

            line_str = line.decode("utf-8")
            buffer += line_str + "\n"

            # 处理 SSE 格式
            if line_str.startswith("data: "):
                try:
                    data = json.loads(line_str[6:])
                    event_type = data.get("type")

                    if event_type == "start":
                        print(f"开始生成消息: {data.get('message_id', '')[:8]}...")
                    elif event_type == "chunk":
                        content = data.get("content", "")
                        print(content, end="", flush=True)
                        yield content
                    elif event_type == "done":
                        print("\n✓ 消息生成完成")
                        message = data.get("message")
                        if message:
                            print(f"消息 ID: {message['id'][:8]}...")
                        session = data.get("session")
                        if session:
                            print(f"Token 使用: {session['prompt_tokens']} prompt + {session['completion_tokens']} completion")
                    elif event_type == "error":
                        error = data.get("error", {})
                        print(f"\n✗ 错误: {error.get('message', 'Unknown error')}")
                except json.JSONDecodeError:
                    continue

    def get_message(self, project_path: str, message_id: str) -> Optional[Dict]:
        """获取单个消息"""
        print(f"\n[10] 获取消息: {message_id[:8]}...")
        encoded_path = urllib.parse.quote(project_path, safe="")
        response = self._request("GET", f"/projects/{encoded_path}/messages/{message_id}")
        data = self._handle_response(response)
        message = data.get("message")
        if message:
            print(f"消息内容: {message['content'][:200]}...")
        return message

    def subscribe_events(self, project_path: str) -> Generator[Dict[str, Any], None, None]:
        """订阅项目的实时事件 (SSE)"""
        print(f"\n[11] 订阅项目实时事件 (SSE): {project_path}")

        # SSE URL: /api/v1/projects/{project_path}/events
        encoded_path = urllib.parse.quote(project_path, safe="")
        sse_url = f"{self.base_url}/projects/{encoded_path}/events"

        try:
            response = requests.get(sse_url, stream=True, headers={
                'Accept': 'text/event-stream',
                'Cache-Control': 'no-cache',
            })

            if response.status_code != 200:
                print(f"✗ SSE 连接失败: HTTP {response.status_code}")
                return

            print("✓ SSE 连接成功，开始接收事件...")

            client = sseclient.SSEClient(response)

            for event in client.events():
                try:
                    data = json.loads(event.data)
                    event_type = event.event if event.event else "unknown"

                    print(f"📡 收到事件 [{event_type}]: {json.dumps(data, ensure_ascii=False, indent=2)[:200]}...")

                    yield {
                        "event_type": event_type,
                        "data": data,
                        "timestamp": time.time()
                    }

                except json.JSONDecodeError as e:
                    print(f"⚠️ 无法解析事件数据: {e}")
                    continue
                except KeyboardInterrupt:
                    print("\n✓ SSE 连接已断开")
                    break

        except requests.RequestException as e:
            print(f"✗ SSE 连接错误: {e}")
        except KeyboardInterrupt:
            print("\n✓ SSE 连接已断开")


def run_full_test(base_url: str, project_path: str):
    """运行完整的 API 测试流程"""
    print("=" * 60)
    print("Crush REST API 测试")
    print("=" * 60)
    print(f"API 地址: {base_url}")
    print(f"项目路径: {project_path}")
    print("=" * 60)

    client = CrushAPIClient(base_url)

    try:
        # 1. 获取所有项目
        projects = client.list_projects()

        # 2. 创建/注册项目
        project = client.create_project(project_path)
        if not project:
            print("错误: 无法创建项目，测试终止")
            return

        # 2.5. 打开项目
        open_result = client.open_project(project_path)
        if not open_result or open_result.get("status") != "opened":
            print("错误: 无法打开项目，测试终止")
            return

        # 3. 获取项目会话
        sessions = client.list_sessions(project_path)

        # 4. 创建新会话
        new_session = client.create_session(
            project_path, f"API 测试会话 - {time.strftime('%Y-%m-%d %H:%M:%S')}"
        )
        if not new_session:
            print("错误: 无法创建会话，测试终止")
            return

        session_id = new_session["id"]

        # 5. 获取会话详情
        client.get_session(project_path, session_id)

        # 6. 发送消息（同步）
        sync_response = client.send_message_sync(
            project_path, session_id, "请用一句话介绍 Go 语言"
        )
        if sync_response:
            message_id = sync_response.get("message", {}).get("id")
            if message_id:
                # 7. 获取消息列表
                client.list_messages(project_path, session_id)

                # 8. 获取单个消息
                client.get_message(project_path, message_id)

        # 9. 发送消息（流式）
        print("\n" + "-" * 60)
        chunks = []
        for chunk in client.send_message_stream(project_path, session_id, "请用一句话介绍 Python 语言"):
            chunks.append(chunk)

        # 10. 再次获取消息列表
        client.list_messages(project_path, session_id)

        # 11. 获取更新后的会话信息
        updated_session = client.get_session(project_path, session_id)

        # 12. 测试 SSE 实时事件（运行一段时间后停止）
        print("\n" + "-" * 60)
        print("12. 测试 SSE 实时事件 (运行 10 秒)...")

        event_count = 0
        start_time = time.time()

        try:
            for event in client.subscribe_events(project_path):
                event_count += 1
                print(f"📊 已收到 {event_count} 个事件")

                # 运行 10 秒后停止
                if time.time() - start_time > 10:
                    print("✓ SSE 测试完成 (10 秒)")
                    break

        except Exception as e:
            print(f"⚠️ SSE 测试异常: {e}")

        print("\n" + "=" * 60)
        print("测试完成!")
        print("=" * 60)
        print(f"创建的会话 ID: {session_id}")
        print(f"会话包含 {updated_session.get('message_count', 0) if updated_session else 0} 条消息")
        print(f"SSE 事件接收: {event_count} 个")

    except requests.exceptions.ConnectionError:
        print("\n错误: 无法连接到 API 服务器")
        print(f"请确保 API 服务器正在运行: crush --server")
        sys.exit(1)
    except KeyboardInterrupt:
        print("\n\n测试被用户中断")
        sys.exit(1)
    except Exception as e:
        print(f"\n错误: {type(e).__name__}: {e}")
        import traceback
        traceback.print_exc()
        sys.exit(1)


def main():
    parser = argparse.ArgumentParser(
        description="Crush REST API 测试脚本",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
示例:
  # 使用默认配置
  python3 test_api.py

  # 指定 API 服务器地址
  python3 test_api.py --base-url http://localhost:3000

  # 指定项目路径
  python3 test_api.py --project-path /path/to/my/project

  # 完整配置
  python3 test_api.py --base-url http://localhost:8080 --project-path /tmp/test-project
        """,
    )
    parser.add_argument(
        "--base-url",
        default="http://localhost:8080/api/v1",
        help="API 服务器基础 URL (默认: http://localhost:8080/api/v1)",
    )
    parser.add_argument(
        "--project-path",
        default="/tmp/crush-test-project",
        help="测试项目路径 (默认: /tmp/crush-test-project)",
    )

    args = parser.parse_args()

    # 确保 base_url 不包含 /api/v1（如果用户提供了完整 URL）
    if args.base_url.endswith("/api/v1"):
        base_url = args.base_url
    elif args.base_url.endswith("/api/v1/"):
        base_url = args.base_url.rstrip("/")
    else:
        # 如果用户只提供了基础 URL，添加 /api/v1
        base_url = f"{args.base_url.rstrip('/')}/api/v1"

    run_full_test(base_url, args.project_path)


if __name__ == "__main__":
    main()
