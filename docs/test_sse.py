#!/usr/bin/env python3
"""SSE 测试脚本"""

import json
import time
import sseclient
import requests


def test_sse(project_path: str = "/tmp/sse-test"):
    """测试 SSE 连接"""
    print(f"测试 SSE: {project_path}")

    base_url = "http://localhost:8080"
    
    # 注册项目
    print("📁 注册项目...")
    requests.post(f"{base_url}/project", json={"path": project_path})

    # 连接 SSE
    print("📡 连接 SSE...")
    sse_url = f"{base_url}/event"

    try:
        response = requests.get(sse_url, stream=True, 
                              params={"directory": project_path},
                              headers={
            'Accept': 'text/event-stream',
            'Cache-Control': 'no-cache',
        })

        if response.status_code != 200:
            print(f"❌ 连接失败: HTTP {response.status_code}")
            return

        print("✅ SSE 连接成功，接收事件(10秒)...")

        client = sseclient.SSEClient(response)
        start_time = time.time()
        event_count = 0

        for event in client.events():
            event_count += 1
            print(f"📡 [{event.event or 'unknown'}]")
            try:
                data = json.loads(event.data)
                print(f"   {json.dumps(data, ensure_ascii=False, indent=2)}")
            except:
                pass

            if time.time() - start_time > 10:
                break

        print(f"✅ 测试完成 (收到 {event_count} 个事件)")

    except (requests.RequestException, KeyboardInterrupt) as e:
        print(f"❌ 错误: {e}")


if __name__ == "__main__":
    test_sse()
