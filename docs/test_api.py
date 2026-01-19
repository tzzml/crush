#!/usr/bin/env python3
"""Crush REST API 测试脚本"""

import argparse
import json
import sys
import threading
import time
import urllib.parse
from typing import Dict, Any, Optional

try:
    import requests
    import sseclient
except ImportError:
    print("错误: 需要安装 requests 和 sseclient-py")
    print("请运行: pip install requests sseclient-py")
    sys.exit(1)


class CrushAPIClient:
    """Crush API 客户端"""

    def __init__(self, base_url: str = "http://localhost:8080/api/v1"):
        self.base_url = base_url.rstrip("/")
        self.session = requests.Session()
        self.session.headers.update({"Content-Type": "application/json"})

    def _request(self, method: str, endpoint: str, **kwargs) -> requests.Response:
        return self.session.request(method, f"{self.base_url}{endpoint}", **kwargs)

    def _handle_response(self, response: requests.Response) -> Dict[str, Any]:
        if response.status_code >= 400:
            try:
                error = response.json().get("error", {})
                print(f"❌ [{error.get('code', 'UNKNOWN')}]: {error.get('message', 'Unknown error')}")
            except:
                print(f"❌ HTTP {response.status_code}")
            return {}
        return response.json()

    def list_projects(self) -> list:
        print("\n[1] 获取项目列表...")
        data = self._handle_response(self._request("GET", "/projects"))
        projects = data.get("projects", [])
        print(f"   找到 {len(projects)} 个项目")
        for i, proj in enumerate(projects, 1):
            print(f"   [{i}] 项目路径: {proj.get('path', 'N/A')}")
            print(f"       数据目录: {proj.get('data_dir', 'N/A')}")
            print(f"       最后访问: {proj.get('last_accessed', 'N/A')}")
        return projects

    def create_project(self, path: str) -> Optional[Dict]:
        print(f"\n[2] 创建项目: {path}")
        data = self._handle_response(self._request("POST", "/projects", json={"path": path}))
        project = data.get("project")
        if project:
            print(f"   ✅ 项目已创建")
            print(f"      路径: {project.get('path', 'N/A')}")
            print(f"      数据目录: {project.get('data_dir', 'N/A')}")
            print(f"      最后访问: {project.get('last_accessed', 'N/A')}")
        return project

    def open_project(self, project_path: str) -> Optional[Dict]:
        print(f"\n[3] 打开项目...")
        encoded = urllib.parse.quote(project_path, safe="")
        data = self._handle_response(self._request("POST", f"/projects/{encoded}/open", json={}))
        if data.get("status") == "opened":
            print(f"   ✅ 项目已打开")
        return data

    def list_sessions(self, project_path: str) -> list:
        print(f"\n[5] 获取会话列表...")
        encoded = urllib.parse.quote(project_path, safe="")
        data = self._handle_response(self._request("GET", f"/projects/{encoded}/sessions"))
        sessions = data.get("sessions", [])
        total = data.get("total", len(sessions))
        print(f"   找到 {len(sessions)} 个会话 (总计: {total})")
        for i, sess in enumerate(sessions, 1):
            print(f"   [{i}] 会话 ID: {sess.get('id', 'N/A')[:16]}...")
            print(f"       标题: {sess.get('title', 'N/A')}")
            print(f"       消息数: {sess.get('message_count', 0)}")
            print(f"       Token: {sess.get('prompt_tokens', 0)} prompt + {sess.get('completion_tokens', 0)} completion")
            print(f"       成本: ${sess.get('cost', 0):.6f}")
            print(f"       创建时间: {sess.get('created_at', 'N/A')}")
        return sessions

    def create_session(self, project_path: str, title: str) -> Optional[Dict]:
        print(f"\n[6] 创建会话: {title}")
        encoded = urllib.parse.quote(project_path, safe="")
        data = self._handle_response(self._request("POST", f"/projects/{encoded}/sessions", json={"title": title}))
        session = data.get("session")
        if session:
            print(f"   ✅ 会话已创建")
            print(f"      ID: {session.get('id', 'N/A')}")
            print(f"      标题: {session.get('title', 'N/A')}")
            print(f"      消息数: {session.get('message_count', 0)}")
            print(f"      创建时间: {session.get('created_at', 'N/A')}")
        return session

    def update_session(self, project_path: str, session_id: str, title: str) -> Optional[Dict]:
        print(f"\n[6.5] 更新会话: {title}")
        encoded = urllib.parse.quote(project_path, safe="")
        data = self._handle_response(self._request("PUT", f"/projects/{encoded}/sessions/{session_id}", json={"title": title}))
        session = data.get("session")
        if session:
            print(f"   ✅ 会话已更新")
            print(f"      ID: {session.get('id', 'N/A')}")
            print(f"      新标题: {session.get('title', 'N/A')}")
            print(f"      更新时间: {session.get('updated_at', 'N/A')}")
        return session

    def send_message_sync(self, project_path: str, session_id: str, prompt: str) -> Optional[Dict]:
        print(f"\n[7] 发送消息: {prompt[:50]}...")
        encoded = urllib.parse.quote(project_path, safe="")
        start = time.time()
        data = self._handle_response(self._request("POST", f"/projects/{encoded}/sessions/{session_id}/messages",
                                                   json={"prompt": prompt, "stream": False}))
        elapsed = time.time() - start
        if data.get("message"):
            msg = data.get("message", {})
            sess = data.get("session", {})
            print(f"   ✅ 响应完成 ({elapsed:.1f}秒)")
            print(f"      消息 ID: {msg.get('id', 'N/A')}")
            print(f"      角色: {msg.get('role', 'N/A')}")
            print(f"      内容预览: {msg.get('content', '')[:100]}...")
            print(f"      模型: {msg.get('model', 'N/A')}")
            print(f"      提供商: {msg.get('provider', 'N/A')}")
            if msg.get('finish_reason'):
                print(f"      完成原因: {msg.get('finish_reason', 'N/A')}")
            if msg.get('parts'):
                parts_count = len(msg.get('parts', []))
                print(f"      部分数: {parts_count}")
            print(f"      创建时间: {msg.get('created_at', 'N/A')}")
            if sess:
                print(f"      会话 Token: {sess.get('prompt_tokens', 0)} prompt + {sess.get('completion_tokens', 0)} completion")
                print(f"      会话成本: ${sess.get('cost', 0):.6f}")
        return data

    def get_config(self, project_path: str) -> Optional[Dict]:
        print(f"\n[8] 获取配置信息...")
        encoded = urllib.parse.quote(project_path, safe="")
        data = self._handle_response(self._request("GET", f"/projects/{encoded}/config"))
        if data:
            print(f"   ✅ 配置已获取")
            print(f"      工作目录: {data.get('working_dir', 'N/A')}")
            print(f"      数据目录: {data.get('data_dir', 'N/A')}")
            print(f"      调试模式: {data.get('debug', False)}")
            print(f"      已配置: {data.get('configured', False)}")
            providers = data.get('providers', [])
            if providers:
                print(f"      提供商: {len(providers)} 个")
                for p in providers:
                    status = "✅" if p.get('configured') else "❌"
                    print(f"        {status} {p.get('name', 'N/A')} ({p.get('type', 'N/A')})")
        return data

    def get_permissions(self, project_path: str) -> Optional[Dict]:
        print(f"\n[9] 获取权限状态...")
        encoded = urllib.parse.quote(project_path, safe="")
        data = self._handle_response(self._request("GET", f"/projects/{encoded}/permissions"))
        if data:
            print(f"   ✅ 权限状态已获取")
            print(f"      跳过请求: {data.get('skip_requests', False)}")
            pending = data.get('pending', [])
            print(f"      待处理请求: {len(pending)} 个")
        return data

    def abort_session(self, project_path: str, session_id: str) -> Optional[Dict]:
        print(f"\n[10] 中止会话处理...")
        encoded = urllib.parse.quote(project_path, safe="")
        data = self._handle_response(self._request("POST", f"/projects/{encoded}/sessions/{session_id}/abort"))
        if data:
            print(f"   ✅ 会话已中止")
            print(f"      状态: {data.get('status', 'N/A')}")
        return data

    def get_session_status(self, project_path: str) -> Optional[Dict]:
        print(f"\n[11] 获取会话状态...")
        encoded = urllib.parse.quote(project_path, safe="")
        data = self._handle_response(self._request("GET", f"/projects/{encoded}/sessions/status"))
        if data:
            print(f"   ✅ 状态已获取")
            print(f"      总会话数: {data.get('total_sessions', 0)}")
            print(f"      应用已配置: {data.get('app_configured', False)}")
            print(f"      Agent 就绪: {data.get('agent_ready', False)}")
        return data

    def subscribe_events(self, project_path: str, callback=None, duration: int = 5):
        """订阅 SSE 事件"""
        encoded = urllib.parse.quote(project_path, safe="")
        sse_url = f"{self.base_url}/projects/{encoded}/events"

        try:
            response = requests.get(sse_url, stream=True, headers={
                'Accept': 'text/event-stream',
                'Cache-Control': 'no-cache',
            })

            if response.status_code != 200:
                if callback:
                    callback("error", {"message": f"SSE 连接失败: HTTP {response.status_code}"})
                return

            client = sseclient.SSEClient(response)
            start_time = time.time()
            event_count = 0

            for event in client.events():
                event_count += 1
                try:
                    data = json.loads(event.data)
                    if callback:
                        callback(event.event or "unknown", data)
                except:
                    pass

                if time.time() - start_time >= duration:
                    break

            if callback:
                callback("done", {"count": event_count})

        except (requests.RequestException, KeyboardInterrupt):
            pass


def run_test(base_url: str, project_path: str):
    """运行测试"""
    print("=" * 50)
    print("Crush REST API 测试")
    print("=" * 50)
    print(f"API: {base_url}")
    print(f"项目: {project_path}")
    print("=" * 50)

    client = CrushAPIClient(base_url)

    try:
        # 1. 列出项目
        client.list_projects()

        # 2. 创建项目
        project = client.create_project(project_path)
        if not project:
            print("❌ 无法创建项目")
            return

        # 3. 打开项目
        if not client.open_project(project_path):
            print("❌ 无法打开项目")
            return

        # 4. 立即订阅 SSE 事件（在后台运行，捕获所有后续事件）
        print(f"\n[4] 订阅 SSE 事件（后台运行，捕获后续操作的事件）...")
        event_count = [0]
        event_log = []

        def handle_event(event_type, data):
            event_count[0] += 1
            print(f"   📡 [{event_type}] 事件 #{event_count[0]}:")
            
            # 尝试提取和显示消息内容
            if isinstance(data, dict):
                # 检查是否是消息事件
                if "id" in data and "role" in data:
                    # 这是消息事件
                    msg_id = data.get("id", "N/A")[:16]
                    role = data.get("role", "N/A")
                    content = data.get("content", "")
                    if not content and "parts" in data:
                        # 尝试从 parts 中提取文本内容
                        parts = data.get("parts", [])
                        for part in parts:
                            if isinstance(part, dict) and part.get("type") == "text":
                                # 新的 parts 格式：{"type": "text", "text": "..."}
                                content = part.get("text", "") or part.get("data", {}).get("text", "")
                                break
                    
                    print(f"      消息 ID: {msg_id}...")
                    print(f"      角色: {role}")
                    if content:
                        preview = content[:200] + "..." if len(content) > 200 else content
                        print(f"      内容: {preview}")
                    if "session_id" in data:
                        print(f"      会话 ID: {data['session_id'][:16]}...")
                    if "model" in data:
                        print(f"      模型: {data.get('model', 'N/A')}")
                    if "provider" in data:
                        print(f"      提供商: {data.get('provider', 'N/A')}")
                
                # 检查是否是会话事件
                elif "title" in data and "id" in data:
                    # 这是会话事件
                    sess_id = data.get("id", "N/A")[:16]
                    title = data.get("title", "N/A")
                    msg_count = data.get("message_count", 0)
                    print(f"      会话 ID: {sess_id}...")
                    print(f"      标题: {title}")
                    print(f"      消息数: {msg_count}")
                    if "prompt_tokens" in data:
                        print(f"      Token: {data.get('prompt_tokens', 0)} prompt + {data.get('completion_tokens', 0)} completion")
                
                # 检查是否是 LSP 事件
                elif "Name" in data and "State" in data:
                    # 这是 LSP 事件
                    name = data.get("Name", "N/A")
                    state = data.get("State", "N/A")
                    print(f"      LSP 客户端: {name}")
                    print(f"      状态: {state}")
                    if "DiagnosticCount" in data:
                        print(f"      诊断数: {data.get('DiagnosticCount', 0)}")
                
                # 其他事件，显示完整 JSON
                else:
                    print(f"      {json.dumps(data, ensure_ascii=False, indent=6)}")
            else:
                print(f"      {json.dumps(data, ensure_ascii=False, indent=6)}")

        sse_thread = threading.Thread(
            target=client.subscribe_events,
            args=(project_path, handle_event, 15),  # 运行15秒，覆盖后续所有操作
            daemon=True
        )
        sse_thread.start()
        time.sleep(1)  # 等待 SSE 连接建立

        # 5. 列出会话
        sessions = client.list_sessions(project_path)

        # 6. 创建会话（会触发事件）
        session = client.create_session(project_path, f"测试会话 - {time.strftime('%H:%M:%S')}")
        if not session:
            print("❌ 无法创建会话")
            return

        # 6.5. 更新会话（测试新功能）
        client.update_session(project_path, session["id"], f"更新后的会话标题 - {time.strftime('%H:%M:%S')}")

        # 7. 发送消息（会触发更多事件）
        message_response = client.send_message_sync(project_path, session["id"], "请用一句话介绍 Go 语言")

        # 8. 获取配置信息
        client.get_config(project_path)

        # 9. 获取权限状态
        client.get_permissions(project_path)

        # 10. 获取会话状态
        client.get_session_status(project_path)

        # 11. 测试中止会话（如果有正在进行的任务）
        # client.abort_session(project_path, session["id"])

        # 等待一段时间让事件处理完成
        time.sleep(2)
        print(f"\n   ✅ SSE 事件订阅完成 (共收到 {event_count[0]} 个事件)")

        print("\n" + "=" * 50)
        print("✅ 测试完成")
        print("=" * 50)

    except requests.exceptions.ConnectionError:
        print("\n❌ 无法连接到服务器")
        print("请运行: crush serve")
        sys.exit(1)
    except KeyboardInterrupt:
        print("\n\n测试被中断")
        sys.exit(1)
    except Exception as e:
        print(f"\n❌ 错误: {e}")
        sys.exit(1)


def main():
    parser = argparse.ArgumentParser(description="Crush REST API 测试脚本")
    parser.add_argument("--base-url", default="http://localhost:8080/api/v1",
                       help="API 服务器地址")
    parser.add_argument("--project-path", default="/tmp/crush-test-project",
                       help="测试项目路径")

    args = parser.parse_args()

    # 处理 base_url
    if args.base_url.endswith("/api/v1"):
        base_url = args.base_url
    elif args.base_url.endswith("/api/v1/"):
        base_url = args.base_url.rstrip("/")
    else:
        base_url = f"{args.base_url.rstrip('/')}/api/v1"

    run_test(base_url, args.project_path)


if __name__ == "__main__":
    main()
