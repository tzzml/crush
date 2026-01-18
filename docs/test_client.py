#!/usr/bin/env python3
"""
Crush API 客户端测试脚本 - 完整版本
演示如何使用REST API和SSE实时事件
"""

import argparse
import json
import requests
import sseclient
import threading
import time
from typing import Optional, Dict, Any


class CrushClient:
    """Crush API 客户端"""

    def __init__(self, base_url: str = "http://localhost:8080/api/v1"):
        self.base_url = base_url.rstrip("/")
        self.session = requests.Session()
        self.session.headers.update({"Content-Type": "application/json"})

    def _get(self, endpoint: str) -> Dict[str, Any]:
        """GET 请求"""
        response = self.session.get(f"{self.base_url}{endpoint}")
        response.raise_for_status()
        return response.json()

    def _post(self, endpoint: str, data: Dict[str, Any]) -> Dict[str, Any]:
        """POST 请求"""
        response = self.session.post(f"{self.base_url}{endpoint}", json=data)
        response.raise_for_status()
        return response.json()

    # ===== REST API 方法 =====

    def list_projects(self) -> Dict[str, Any]:
        """列出所有项目"""
        return self._get("/projects")

    def create_project(self, path: str) -> Dict[str, Any]:
        """创建项目"""
        return self._post("/projects", {"path": path})

    def open_project(self, project_path: str) -> Dict[str, Any]:
        """打开项目的 app 实例"""
        encoded_path = requests.utils.quote(project_path, safe="")
        return self._post(f"/projects/{encoded_path}/open", {})

    def close_project(self, project_path: str) -> Dict[str, Any]:
        """关闭项目的 app 实例"""
        encoded_path = requests.utils.quote(project_path, safe="")
        return self._post(f"/projects/{encoded_path}/close", {})

    def connect_project(self, project_path: str) -> Dict[str, Any]:
        """检查项目连接状态"""
        encoded_path = requests.utils.quote(project_path, safe="")
        return self._get(f"/projects/{encoded_path}/connect")

    def get_session(self, project_path: str, session_id: str) -> Dict[str, Any]:
        """获取会话详情"""
        encoded_path = requests.utils.quote(project_path, safe="")
        return self._get(f"/projects/{encoded_path}/sessions/{session_id}")

    def create_session(self, project_path: str, title: str) -> Dict[str, Any]:
        """创建新会话"""
        encoded_path = requests.utils.quote(project_path, safe="")
        return self._post(f"/projects/{encoded_path}/sessions", {"title": title})

    def send_message(self, project_path: str, session_id: str, prompt: str) -> Dict[str, Any]:
        """发送消息"""
        encoded_path = requests.utils.quote(project_path, safe="")
        return self._post(f"/projects/{encoded_path}/sessions/{session_id}/messages",
                         {"prompt": prompt})

    # ===== SSE 事件订阅 =====

    def subscribe_events(self, project_path: str, callback=None):
        """订阅项目的实时事件

        Args:
            project_path: 项目路径
            callback: 事件处理回调函数，参数为 (event_type, data)
        """
        encoded_path = requests.utils.quote(project_path, safe="")
        sse_url = f"{self.base_url}/projects/{encoded_path}/events"

        print(f"🔗 连接到 SSE: {sse_url}")

        try:
            response = requests.get(sse_url, stream=True, headers={
                'Accept': 'text/event-stream',
                'Cache-Control': 'no-cache',
            })

            if response.status_code != 200:
                print(f"❌ SSE 连接失败: HTTP {response.status_code}")
                return

            print("✅ SSE 连接成功")

            client = sseclient.SSEClient(response)
            event_count = 0

            for event in client.events():
                try:
                    event_count += 1
                    event_type = event.event or "unknown"
                    data = json.loads(event.data)

                    print(f"📡 事件 #{event_count} [{event_type}]: {json.dumps(data, ensure_ascii=False, indent=2)[:200]}...")

                    # 调用回调函数
                    if callback:
                        callback(event_type, data)

                except json.JSONDecodeError as e:
                    print(f"⚠️ 解析事件失败: {e}")
                except KeyboardInterrupt:
                    print("\n🛑 SSE 连接被中断")
                    break

        except requests.RequestException as e:
            print(f"❌ SSE 连接错误: {e}")
        except KeyboardInterrupt:
            print("\n🛑 SSE 连接被中断")


def demo_basic_api_usage():
    """演示基本的API使用"""
    print("🚀 演示基本的API使用...")

    client = CrushClient()

    try:
        # 1. 列出项目
        projects = client.list_projects()
        print(f"📂 找到 {projects['total']} 个项目")

        # 2. 创建项目
        project = client.create_project("/tmp/test-project")
        print(f"📁 创建项目: {project.get('path', '/tmp/test-project')}")

        # 3. 打开项目
        open_result = client.open_project("/tmp/test-project")
        print(f"🔓 打开项目: {open_result.get('status', 'opened')}")

        # 4. 创建会话
        session = client.create_session("/tmp/test-project", "API测试会话")
        print(f"💬 创建会话: {session['title'][:20]}...")

        # 5. 发送消息
        message = client.send_message("/tmp/test-project", session['id'], "你好，介绍一下Go语言")
        print(f"💌 发送消息: {message.get('message', {}).get('content', '')[:50]}...")

        # 6. 关闭项目
        close_result = client.close_project("/tmp/test-project")
        print(f"🔒 关闭项目: {close_result.get('status', 'closed')}")

        print("✅ 基本API演示完成")

    except Exception as e:
        print(f"❌ API演示失败: {e}")


def demo_sse_subscription():
    """演示SSE事件订阅"""
    print("\n🌟 演示SSE事件订阅...")

    client = CrushClient()

    # 定义事件处理回调
    def handle_event(event_type: str, data: Dict[str, Any]):
        if event_type == "updated" and "Type" in data:
            if data["Type"] == "state_changed":
                print(f"🔄 LSP {data['Name']} 状态变化: {data['State']}")
            elif data["Type"] == "diagnostics_changed":
                print(f"📊 LSP {data['Name']} 诊断变化: {data['DiagnosticCount']} 个问题")

    # 先打开项目
    project_path = "/tmp/sse-test-project"
    client.create_project(project_path)
    client.open_project(project_path)

    # 在后台启动SSE监听
    sse_thread = threading.Thread(
        target=client.subscribe_events,
        args=(project_path, handle_event),
        daemon=True
    )
    sse_thread.start()

    print("⏳ SSE监听已启动，等待事件...")

    # 等待一段时间让SSE连接建立
    time.sleep(2)

    # 在另一个线程中执行一些API操作来触发事件
    def trigger_events():
        try:
            # 创建项目和会话来触发LSP初始化
            project_path = "/tmp/sse-test-project"
            client.create_project(project_path)
            client.open_project(project_path)
            session = client.create_session(project_path, "SSE测试会话")

            # 发送消息
            for i in range(3):
                client.send_message(project_path, session['id'],
                                  f"测试消息 {i+1}")
                time.sleep(1)

        except Exception as e:
            print(f"⚠️ 触发事件时出错: {e}")

    trigger_thread = threading.Thread(target=trigger_events, daemon=True)
    trigger_thread.start()

    # 等待一段时间观察事件
    print("⏳ 观察事件 10 秒...")
    time.sleep(10)

    print("✅ SSE演示完成")


def demo_combined_usage():
    """演示REST API + SSE的组合使用"""
    print("\n🎯 演示REST API + SSE组合使用...")

    client = CrushClient()

    # 使用REST API
    try:
        projects = client.list_projects()
        print(f"📊 当前有 {projects['total']} 个项目")

        # 创建新项目
        project_path = "/tmp/combined-test"
        project = client.create_project(project_path)
        print(f"🏗️ 创建项目: {project.get('path', project_path)}")

        # 打开项目
        client.open_project(project_path)
        print("🔓 项目已打开")

        # 启动SSE监听
        def handle_event(event_type: str, data: Dict[str, Any]):
            print(f"🎉 实时事件: {event_type} - {json.dumps(data, ensure_ascii=False)[:100]}...")

        sse_thread = threading.Thread(
            target=client.subscribe_events,
            args=(project_path, handle_event),
            daemon=True
        )
        sse_thread.start()

        time.sleep(1)  # 等待SSE连接

        # 创建会话
        session = client.create_session(project_path, "组合测试会话")
        print(f"💬 创建会话: {session['id'][:8]}...")

        # 发送几条消息
        for i in range(2):
            msg = client.send_message(project_path, session['id'],
                                    f"组合测试消息 {i+1}")
            print(f"📤 发送消息: {msg.get('message', {}).get('id', '')[:8]}...")
            time.sleep(2)

        print("✅ 组合使用演示完成")

    except Exception as e:
        print(f"❌ 组合演示失败: {e}")


def main():
    parser = argparse.ArgumentParser(description="Crush API 客户端演示")
    parser.add_argument("--base-url", default="http://localhost:8080/api/v1",
                       help="API服务器地址")
    parser.add_argument("--demo", choices=["basic", "sse", "combined", "all"],
                       default="all", help="演示类型")

    args = parser.parse_args()

    print("🎪 Crush API 客户端演示")
    print("=" * 50)
    print(f"API地址: {args.base_url}")
    print(f"演示类型: {args.demo}")
    print("=" * 50)

    if args.demo in ["basic", "all"]:
        demo_basic_api_usage()

    if args.demo in ["sse", "all"]:
        demo_sse_subscription()

    if args.demo in ["combined", "all"]:
        demo_combined_usage()

    print("\n🎉 所有演示完成!")


if __name__ == "__main__":
    main()