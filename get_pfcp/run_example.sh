#!/bin/bash

echo "=== PFCP即时分析工具使用示例 ==="
echo

# 检查Go是否安装
if ! command -v go &> /dev/null; then
    echo "错误: 未找到Go编译器"
    echo "请先安装Go 1.22.0或更高版本"
    exit 1
fi

echo "Go版本: $(go version)"
echo

# 检查是否以root权限运行
if [ "$EUID" -ne 0 ]; then
    echo "警告: 此工具需要root权限来捕获网络数据包"
    echo "请使用以下命令运行:"
    echo "sudo ./run_example.sh"
    echo
    echo "或者直接运行:"
    echo "sudo go run pfcp_analyzer.go"
    exit 1
fi

echo "开始运行PFCP分析器..."
echo "按Ctrl+C停止程序"
echo

# 运行PFCP分析器
go run pfcp_analyzer.go 