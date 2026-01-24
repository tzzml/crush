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

    def __init__(self, base_url: str = "http://localhost:8080"):
        self.base_url = base_url.rstrip("/")
        self.session = requests.Session()
        self.session.headers.update({"Content-Type": "application/json"})

    def _request(self, method: str, endpoint: str, params: Optional[Dict] = None, **kwargs) -> requests.Response:
        url = f"{self.base_url}{endpoint}"
        return self.session.request(method, url, params=params, **kwargs)

    def _handle_response(self, response: requests.Response) -> Dict[str, Any]:
        if response.status_code >= 400:
            try:
                error = response.json().get("error", {})
                print(f"❌ [{error.get('code', 'UNKNOWN')}]: {error.get('message', 'Unknown error')}")
            except:
                print(f"❌ HTTP {response.status_code}")
                print(response.text)
            return {}
        return response.json()

    def list_projects(self) -> list:
        print("\n[1] 获取项目列表...")
        data = self._handle_response(self._request("GET", "/project"))
        projects = data.get("projects", [])
        print(f"   找到 {len(projects)} 个项目")
        for i, proj in enumerate(projects, 1):
            print(f"   [{i}] 项目路径: {proj.get('path', 'N/A')}")
            print(f"       数据目录: {proj.get('data_dir', 'N/A')}")
            print(f"       最后访问: {proj.get('last_accessed', 'N/A')}")
        return projects

    def create_project(self, path: str) -> Optional[Dict]:
        print(f"\n[2] 注册项目: {path}")
        data = self._handle_response(self._request("POST", "/project", json={"path": path}))
        project = data.get("project")
        if project:
            print(f"   ✅ 项目已注册")
            print(f"      路径: {project.get('path', 'N/A')}")
            print(f"      数据目录: {project.get('data_dir', 'N/A')}")
        return project

    def list_sessions(self, project_path: str) -> list:
        print(f"\n[3] 获取会话列表...")
        data = self._handle_response(self._request("GET", "/session", params={"directory": project_path}))
        sessions = data.get("sessions", [])
        total = data.get("total", len(sessions))
        print(f"   找到 {len(sessions)} 个会话 (总计: {total})")
        for i, sess in enumerate(sessions, 1):
            print(f"   [{i}] 会话 ID: {sess.get('id', 'N/A')[:16]}...")
            print(f"       标题: {sess.get('title', 'N/A')}")
            print(f"       消息数: {sess.get('message_count', 0)}")
        return sessions

    def create_session(self, project_path: str, title: str) -> Optional[Dict]:
        print(f"\n[4] 创建会话: {title}")
        data = self._handle_response(self._request("POST", "/session", 
                                                 params={"directory": project_path},
                                                 json={"title": title}))
        session = data.get("session")
        if session:
            print(f"   ✅ 会话已创建")
            print(f"      ID: {session.get('id', 'N/A')}")
            print(f"      标题: {session.get('title', 'N/A')}")
        return session

    def update_session(self, project_path: str, session_id: str, title: str) -> Optional[Dict]:
        print(f"\n[5] 更新会话: {title}")
        data = self._handle_response(self._request("PUT", f"/session/{session_id}", 
                                                 params={"directory": project_path},
                                                 json={"title": title}))
        session = data.get("session")
        if session:
            print(f"   ✅ 会话已更新")
            print(f"      ID: {session.get('id', 'N/A')}")
            print(f"      新标题: {session.get('title', 'N/A')}")
        return session

    def send_message_sync(self, project_path: str, session_id: str, prompt: str) -> Optional[Dict]:
        print(f"\n[6] 发送消息: {prompt[:50]}...")
        start = time.time()
        data = self._handle_response(self._request("POST", f"/session/{session_id}/message",
                                                   params={"directory": project_path},
                                                   json={"prompt": prompt, "stream": False}))
        elapsed = time.time() - start
        if data.get("message"):
            msg = data.get("message", {})
            print(f"   ✅ 响应完成 ({elapsed:.1f}秒)")
            print(f"      消息 ID: {msg.get('id', 'N/A')}")
            print(f"      内容预览: {msg.get('content', '')[:100]}...")
        return data

    def get_config(self, project_path: str) -> Optional[Dict]:
        print(f"\n[7] 获取配置信息...")
        data = self._handle_response(self._request("GET", "/project/config", params={"directory": project_path}))
        if data:
            print(f"   ✅ 配置已获取")
            print(f"      工作目录: {data.get('working_dir', 'N/A')}")
            print(f"      已配置: {data.get('configured', False)}")
        return data

    def get_permissions(self, project_path: str) -> Optional[Dict]:
        print(f"\n[8] 获取权限状态...")
        data = self._handle_response(self._request("GET", "/project/permissions", params={"directory": project_path}))
        if data:
            print(f"   ✅ 权限状态已获取")
            print(f"      跳过请求: {data.get('skip_requests', False)}")
        return data

    def get_session_status(self, project_path: str) -> Optional[Dict]:
        print(f"\n[9] 获取会话状态...")
        data = self._handle_response(self._request("GET", "/session/status", params={"directory": project_path}))
        if data:
            print(f"   ✅ 状态已获取")
            print(f"      总会话数: {data.get('total_sessions', 0)}")
        return data

    def subscribe_events(self, project_path: str, callback=None, duration: int = 5):
        """订阅 SSE 事件"""
        sse_url = f"{self.base_url}/event"
        print(f"   📡 连接 SSE: {sse_url}")

        try:
            response = requests.get(sse_url, stream=True, 
                                  params={"directory": project_path},
                                  headers={
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
    print("Crush REST API 测试 (Flat Structure)")
    print("=" * 50)
    print(f"API: {base_url}")
    print(f"项目: {project_path}")
    print("=" * 50)

    client = CrushAPIClient(base_url)

    try:
        # 1. 列出项目
        client.list_projects()

        # 2. 注册项目
        client.create_project(project_path)
        
        # 3. 立即订阅 SSE 事件
        print(f"\n[SSE] 启动后台订阅...")
        
        def handle_event(event_type, data):
            # 简化版日志
            if event_type not in ["heartbeat", "unknown"]:
                 print(f"   📡 [Event: {event_type}]")

        sse_thread = threading.Thread(
            target=client.subscribe_events,
            args=(project_path, handle_event, 10),
            daemon=True
        )
        sse_thread.start()
        time.sleep(1)

        # 4. 创建会话
        session = client.create_session(project_path, f"测试会话 - {time.strftime('%H:%M:%S')}")
        if not session:
            print("❌ 无法创建会话")
            return

        # 5. 更新会话
        client.update_session(project_path, session["id"], f"Updated - {time.strftime('%H:%M:%S')}")

        # 6. 发送消息
        client.send_message_sync(project_path, session["id"], "Hello API")

        # 7. 获取配置
        client.get_config(project_path)
        
        # 8. 获取权限
        client.get_permissions(project_path)

        # 9. 获取状态
        client.get_session_status(project_path)

        time.sleep(2)
        print("\n" + "=" * 50)
        print("✅ 测试流程结束")

    except Exception as e:
        print(f"\n❌ 错误: {e}")
        sys.exit(1)


def main():
    parser = argparse.ArgumentParser(description="Crush REST API 测试脚本")
    parser.add_argument("--base-url", default="http://localhost:8080", help="API 服务器地址")
    parser.add_argument("--project-path", default="/tmp/crush-test-project", help="测试项目路径")
    args = parser.parse_args()
    
    run_test(args.base_url, args.project_path)


if __name__ == "__main__":
    main()
