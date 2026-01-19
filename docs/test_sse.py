#!/usr/bin/env python3
"""SSE 测试脚本"""

import json
import time
import sseclient
import requests


def test_sse(project_path: str = "/tmp/sse-test"):
    """测试 SSE 连接"""
    print(f"测试 SSE: {project_path}")

    base_url = "http://localhost:8080/api/v1"
    encoded = requests.utils.quote(project_path, safe="")

    # 创建项目
    print("📁 创建项目...")
    requests.post(f"{base_url}/projects", json={"path": project_path})

    # 打开项目
    print("🔓 打开项目...")
    requests.post(f"{base_url}/projects/{encoded}/open", json={})
    time.sleep(1)  # 等待 LSP 初始化

    # 立即连接 SSE（在项目打开后，捕获后续所有事件）
    print("📡 连接 SSE...")
    sse_url = f"{base_url}/projects/{encoded}/events"

    try:
        response = requests.get(sse_url, stream=True, headers={
            'Accept': 'text/event-stream',
            'Cache-Control': 'no-cache',
        })

        if response.status_code != 200:
            print(f"❌ 连接失败: HTTP {response.status_code}")
            return

        print("✅ SSE 连接成功，接收事件...")

        client = sseclient.SSEClient(response)
        start_time = time.time()
        event_count = 0

        for event in client.events():
            event_count += 1
            print(f"📡 [{event.event or 'unknown'}] 事件 #{event_count}:")
            try:
                data = json.loads(event.data)
                
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
                        
                        print(f"   消息 ID: {msg_id}...")
                        print(f"   角色: {role}")
                        if content:
                            preview = content[:150] + "..." if len(content) > 150 else content
                            print(f"   内容: {preview}")
                    # 检查是否是会话事件
                    elif "title" in data and "id" in data:
                        print(f"   会话: {data.get('title', 'N/A')} ({data.get('message_count', 0)} 条消息)")
                    # 检查是否是 LSP 事件
                    elif "Name" in data and "State" in data:
                        print(f"   LSP {data.get('Name', 'N/A')}: {data.get('State', 'N/A')}")
                    # 其他事件
                    else:
                        print(f"   {json.dumps(data, ensure_ascii=False, indent=4)}")
                else:
                    print(f"   {json.dumps(data, ensure_ascii=False, indent=4)}")
            except:
                print(f"   {event.data}")

            if time.time() - start_time > 10:  # 运行10秒，捕获更多事件
                break

        print(f"✅ 测试完成 (收到 {event_count} 个事件)")

    except (requests.RequestException, KeyboardInterrupt) as e:
        print(f"❌ 错误: {e}")


if __name__ == "__main__":
    test_sse()
