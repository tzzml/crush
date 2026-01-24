#!/bin/bash

# OpsAgent 配置验证脚本

echo "=== OpsAgent 配置验证 ==="
echo ""

# 1. 检查项目配置
echo "1. 检查项目配置 (.crush.json)"
if [ -f ".crush.json" ]; then
    echo "   ✅ 项目配置存在"
    echo "   配置内容："
    cat .crush.json | jq '{disabled_tools: .options.disabled_tools, skills_paths: .options.skills_paths, allowed_tools: .permissions.allowed_tools}'
else
    echo "   ❌ 项目配置不存在"
fi
echo ""

# 2. 检查全局配置
echo "2. 检查全局配置 (~/.config/crush/crush.json)"
if [ -f "$HOME/.config/crush/crush.json" ]; then
    echo "   ✅ 全局配置存在"
    echo "   配置内容："
    cat "$HOME/.config/crush/crush.json" | jq '.'
else
    echo "   ❌ 全局配置不存在"
fi
echo ""

# 3. 检查 Ops 规则
echo "3. 检查 Ops 规则 (~/.config/crush/ops-rules)"
if [ -f "$HOME/.config/crush/ops-rules" ]; then
    echo "   ✅ Ops 规则存在"
    echo "   文件大小：$(wc -l < "$HOME/.config/crush/ops-rules") 行"
else
    echo "   ❌ Ops 规则不存在"
fi
echo ""

# 4. 检查 Skills
echo "4. 检查 Skills 目录"
if [ -d "$HOME/.config/crush/skills" ]; then
    echo "   ✅ Skills 目录存在"
    echo "   可用的 Skills："
    ls -1 "$HOME/.config/crush/skills/" | sed 's/^/     - /'
else
    echo "   ❌ Skills 目录不存在"
fi
echo ""

# 5. 配置优先级说明
echo "5. 配置优先级"
echo "   Crush 会按以下优先级加载配置："
echo "   1. 全局配置: ~/.config/crush/crush.json (优先级最低)"
echo "   2. 全局数据配置: ~/.crush/crush.json"
echo "   3. 项目配置: .crush.json (优先级最高) ← 当前使用"
echo ""

# 6. 使用说明
echo "6. 使用说明"
echo "   在 zorkagent 项目中启动 crush："
echo "   $ cd /Users/zhuminglei/Projects/zorkagent"
echo "   $ crush"
echo ""
echo "   Agent 会自动使用 Ops 模式（因为项目根目录有 .crush.json）"
echo ""

# 7. 切换模式说明
echo "7. 切换回默认模式"
echo "   临时禁用 Ops 模式："
echo "   $ mv .crush.json .crush.json.ops"
echo "   $ crush  # 使用默认 CoderAgent"
echo "   $ mv .crush.json.ops .crush.json  # 恢复 Ops 模式"
echo ""
