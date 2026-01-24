#!/usr/bin/env python3
"""Crush API 客户端演示"""

import json
import requests
import sseclient
import time
from typing import Optional, Dict, Any


class CrushClient:
    """Crush API 客户端"""

    def __init__(self, base_url: str = "http://localhost:8080"):
        self.base_url = base_url.rstrip("/")
        self.session = requests.Session()
        self.session.headers.update({"Content-Type": "application/json"})

    def _post(self, endpoint: str, params: Dict = None, data: Dict[str, Any] = None) -> Dict[str, Any]:
        response = self.session.post(f"{self.base_url}{endpoint}", params=params, json=data or {})
        response.raise_for_status()
        return response.json()

    def _get(self, endpoint: str, params: Dict = None) -> Dict[str, Any]:
        response = self.session.get(f"{self.base_url}{endpoint}", params=params)
        response.raise_for_status()
        return response.json()

    def create_project(self, project_path: str):
        """注册项目"""
        return self._post("/project", data={"path": project_path})

    def create_session(self, project_path: str, title: str) -> Dict[str, Any]:
        """创建会话"""
        return self._post("/session", params={"directory": project_path}, data={"title": title})

    def send_message(self, project_path: str, session_id: str, prompt: str) -> Dict[str, Any]:
        """发送消息"""
        return self._post(f"/session/{session_id}/message", 
                         params={"directory": project_path},
                         data={"prompt": prompt})

    def get_config(self, project_path: str) -> Dict[str, Any]:
        """获取配置"""
        return self._get("/project/config", params={"directory": project_path})

    def get_permissions(self, project_path: str) -> Dict[str, Any]:
        """获取权限状态"""
        return self._get("/project/permissions", params={"directory": project_path})

    def get_session_status(self, project_path: str) -> Dict[str, Any]:
        """获取会话状态"""
        return self._get("/session/status", params={"directory": project_path})

    def subscribe_events(self, project_path: str, callback=None):
        """订阅 SSE 事件"""
        sse_url = f"{self.base_url}/event"

        try:
            response = requests.get(sse_url, stream=True, 
                                  params={"directory": project_path},
                                  headers={
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
        # 注册项目
        print("📁 注册项目...")
        project_resp = client.create_project(project_path)
        print(f"   ✅ 项目已注册")

        # 立即订阅事件
        print("📡 订阅事件 (后台运行)...")
        event_count = [0]

        def handle_event(event_type, data):
            event_count[0] += 1
            if event_type not in ["heartbeat", "unknown"]:
                print(f"   📡 [Event: {event_type}]")

        import threading
        thread = threading.Thread(target=client.subscribe_events, args=(project_path, handle_event), daemon=True)
        thread.start()
        time.sleep(1)

        # 创建会话
        print("💬 创建会话...")
        session_resp = client.create_session(project_path, "演示会话")
        session_data = session_resp.get('session', {})
        print(f"   ✅ 会话 ID: {session_data.get('id', 'N/A')}")

        # 发送消息
        print("📤 发送消息...")
        msg_resp = client.send_message(project_path, session_data['id'], "你好")
        msg_data = msg_resp.get('message', {})
        print(f"   ✅ 消息 ID: {msg_data.get('id', 'N/A')}")

        # 获取配置信息
        print("⚙️  获取配置...")
        config = client.get_config(project_path)
        print(f"   ✅ 已配置: {config.get('configured', False)}")

        # 获取权限状态
        print("🔐 获取权限状态...")
        perms = client.get_permissions(project_path)
        print(f"   ✅ 跳过请求: {perms.get('skip_requests', False)}")

        # 获取会话状态
        print("📊 获取会话状态...")
        status = client.get_session_status(project_path)
        print(f"   ✅ 总会话数: {status.get('total_sessions', 0)}")

        time.sleep(3)
        print(f"\n✅ 演示完成 (收到 {event_count[0]} 个事件)")

    except Exception as e:
        print(f"❌ 错误: {e}")


if __name__ == "__main__":
    main()
