#!/usr/bin/env python3
"""Crush API 客户端演示"""

import json
import requests
import sseclient
import time
from typing import Optional, Dict, Any


class CrushClient:
    """Crush API 客户端"""

    def __init__(self, base_url: str = "http://localhost:8080/api/v1"):
        self.base_url = base_url.rstrip("/")
        self.session = requests.Session()
        self.session.headers.update({"Content-Type": "application/json"})

    def _post(self, endpoint: str, data: Dict[str, Any] = None) -> Dict[str, Any]:
        response = self.session.post(f"{self.base_url}{endpoint}", json=data or {})
        response.raise_for_status()
        return response.json()

    def _get(self, endpoint: str) -> Dict[str, Any]:
        response = self.session.get(f"{self.base_url}{endpoint}")
        response.raise_for_status()
        return response.json()

    def open_project(self, project_path: str) -> Dict[str, Any]:
        """打开项目"""
        encoded = requests.utils.quote(project_path, safe="")
        return self._post(f"/projects/{encoded}/open")

    def close_project(self, project_path: str) -> Dict[str, Any]:
        """关闭项目"""
        encoded = requests.utils.quote(project_path, safe="")
        return self._post(f"/projects/{encoded}/close")

    def create_session(self, project_path: str, title: str) -> Dict[str, Any]:
        """创建会话"""
        encoded = requests.utils.quote(project_path, safe="")
        return self._post(f"/projects/{encoded}/sessions", {"title": title})

    def send_message(self, project_path: str, session_id: str, prompt: str) -> Dict[str, Any]:
        """发送消息"""
        encoded = requests.utils.quote(project_path, safe="")
        return self._post(f"/projects/{encoded}/sessions/{session_id}/messages", {"prompt": prompt})

    def get_config(self, project_path: str) -> Dict[str, Any]:
        """获取配置"""
        encoded = requests.utils.quote(project_path, safe="")
        return self._get(f"/projects/{encoded}/config")

    def get_permissions(self, project_path: str) -> Dict[str, Any]:
        """获取权限状态"""
        encoded = requests.utils.quote(project_path, safe="")
        return self._get(f"/projects/{encoded}/permissions")

    def abort_session(self, project_path: str, session_id: str) -> Dict[str, Any]:
        """中止会话"""
        encoded = requests.utils.quote(project_path, safe="")
        return self._post(f"/projects/{encoded}/sessions/{session_id}/abort")

    def get_session_status(self, project_path: str) -> Dict[str, Any]:
        """获取会话状态"""
        encoded = requests.utils.quote(project_path, safe="")
        return self._get(f"/projects/{encoded}/sessions/status")

    def subscribe_events(self, project_path: str, callback=None):
        """订阅 SSE 事件"""
        encoded = requests.utils.quote(project_path, safe="")
        sse_url = f"{self.base_url}/projects/{encoded}/events"

        try:
            response = requests.get(sse_url, stream=True, headers={
                'Accept': 'text/event-stream',
                'Cache-Control': 'no-cache',
            })

            if response.status_code != 200:
                print(f"❌ SSE 连接失败: HTTP {response.status_code}")
                return

            client = sseclient.SSEClient(response)
            for event in client.events():
                try:
                    data = json.loads(event.data)
                    if callback:
                        callback(event.event or "unknown", data)
                except (json.JSONDecodeError, KeyboardInterrupt):
                    break
        except (requests.RequestException, KeyboardInterrupt):
            pass


def main():
    print("🎪 Crush API 客户端演示\n")

    client = CrushClient()
    project_path = "/tmp/demo-project"

    try:
        # 创建并打开项目
        print("📁 创建项目...")
        project_resp = requests.post("http://localhost:8080/api/v1/projects", json={"path": project_path})
        if project_resp.status_code == 200:
            project_data = project_resp.json().get("project", {})
            print(f"   ✅ 项目已创建")
            print(f"      路径: {project_data.get('path', 'N/A')}")
            print(f"      数据目录: {project_data.get('data_dir', 'N/A')}")

        print("🔓 打开项目...")
        open_result = client.open_project(project_path)
        print(f"   ✅ 项目已打开: {open_result.get('status', 'N/A')}")
        time.sleep(1)  # 等待 LSP 初始化

        # 立即订阅事件（在后台运行，捕获后续所有操作的事件）
        print("📡 订阅事件 (后台运行)...")
        event_count = [0]

        def handle_event(event_type, data):
            event_count[0] += 1
            print(f"   📡 [{event_type}] 事件 #{event_count[0]}:")
            
            # 尝试提取和显示消息内容
            if isinstance(data, dict):
                # 检查是否是消息事件
                if "id" in data and "role" in data:
                    msg_id = data.get("id", "N/A")[:16]
                    role = data.get("role", "N/A")
                    content = data.get("content", "")
                    if not content and "parts" in data:
                        parts = data.get("parts", [])
                        for part in parts:
                            if isinstance(part, dict) and part.get("type") == "text":
                                # 新的 parts 格式：{"type": "text", "text": "..."}
                                content = part.get("text", "") or part.get("data", {}).get("text", "")
                                break
                    
                    print(f"      消息 ID: {msg_id}...")
                    print(f"      角色: {role}")
                    if content:
                        preview = content[:150] + "..." if len(content) > 150 else content
                        print(f"      内容: {preview}")
                
                # 检查是否是会话事件
                elif "title" in data and "id" in data:
                    print(f"      会话: {data.get('title', 'N/A')} ({data.get('message_count', 0)} 条消息)")
                
                # 检查是否是 LSP 事件
                elif "Name" in data and "State" in data:
                    print(f"      LSP {data.get('Name', 'N/A')}: {data.get('State', 'N/A')}")
                
                # 其他事件
                else:
                    print(f"      {json.dumps(data, ensure_ascii=False, indent=6)}")
            else:
                print(f"      {json.dumps(data, ensure_ascii=False, indent=6)}")

        import threading
        thread = threading.Thread(target=client.subscribe_events, args=(project_path, handle_event), daemon=True)
        thread.start()
        time.sleep(1)  # 等待 SSE 连接建立

        # 创建会话（会触发事件）
        print("💬 创建会话...")
        session = client.create_session(project_path, "演示会话")
        session_data = session.get('session', {})
        print(f"   ✅ 会话已创建")
        print(f"      ID: {session_data.get('id', 'N/A')}")
        print(f"      标题: {session_data.get('title', 'N/A')}")
        print(f"      消息数: {session_data.get('message_count', 0)}")

        # 发送消息（会触发更多事件）
        print("📤 发送消息...")
        message = client.send_message(project_path, session_data['id'], "你好")
        msg_data = message.get('message', {})
        sess_data = message.get('session', {})
        print(f"   ✅ 消息已发送")
        print(f"      消息 ID: {msg_data.get('id', 'N/A')}")
        print(f"      角色: {msg_data.get('role', 'N/A')}")
        print(f"      内容预览: {msg_data.get('content', '')[:100]}...")
        if sess_data:
            print(f"      会话 Token: {sess_data.get('prompt_tokens', 0)} prompt + {sess_data.get('completion_tokens', 0)} completion")

        # 获取配置信息
        print("⚙️  获取配置...")
        config = client.get_config(project_path)
        if config:
            print(f"   ✅ 配置已获取")
            print(f"      工作目录: {config.get('working_dir', 'N/A')}")
            print(f"      已配置: {config.get('configured', False)}")

        # 获取权限状态
        print("🔐 获取权限状态...")
        perms = client.get_permissions(project_path)
        if perms:
            print(f"   ✅ 权限状态已获取")
            print(f"      跳过请求: {perms.get('skip_requests', False)}")

        # 获取会话状态
        print("📊 获取会话状态...")
        status = client.get_session_status(project_path)
        if status:
            print(f"   ✅ 状态已获取")
            print(f"      总会话数: {status.get('total_sessions', 0)}")
            print(f"      Agent 就绪: {status.get('agent_ready', False)}")

        # 等待事件处理
        time.sleep(3)

        print(f"\n✅ 演示完成 (收到 {event_count[0]} 个事件)")

    except Exception as e:
        print(f"❌ 错误: {e}")


if __name__ == "__main__":
    main()
