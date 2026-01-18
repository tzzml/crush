#!/usr/bin/env python3
"""
SSE 测试脚本
"""

import json
import time
import sseclient
import requests

def test_sse(project_path: str = "/tmp/sse-test-project"):
    """测试SSE连接"""
    print(f"测试 SSE 连接 (项目: {project_path})...")

    # 先创建并打开项目
    import requests
    base_url = "http://localhost:8080/api/v1"
    
    # 创建项目
    encoded_path = requests.utils.quote(project_path, safe="")
    requests.post(f"{base_url}/projects", json={"path": project_path})
    
    # 打开项目
    requests.post(f"{base_url}/projects/{encoded_path}/open", json={})
    
    # SSE URL
    sse_url = f"{base_url}/projects/{encoded_path}/events"

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

        event_count = 0
        start_time = time.time()

        for event in client.events():
            try:
                event_count += 1
                print(f"📡 收到事件 [{event.event}]: {event.data[:100]}...")

                # 运行 5 秒后停止
                if time.time() - start_time > 5:
                    print("✓ SSE 测试完成 (5 秒)")
                    break

            except json.JSONDecodeError as e:
                print(f"⚠️ 无法解析事件数据: {e}")
                continue
            except KeyboardInterrupt:
                print("\n✓ SSE 连接已断开")
                break

        print(f"总共收到 {event_count} 个事件")

    except requests.RequestException as e:
        print(f"✗ SSE 连接错误: {e}")
    except KeyboardInterrupt:
        print("\n✓ SSE 连接已断开")

if __name__ == "__main__":
    test_sse()