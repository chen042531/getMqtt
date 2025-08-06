# Drop Interface Packet 工具

这个 Go 程序用于 drop 指定网络接口的 packet，不使用 `exec.Command`。

## 功能

- 通过系统调用直接操作网络接口
- 将指定接口设置为 down 状态，从而 drop 所有通过该接口的 packet
- 提供恢复接口功能

## 使用方法

### 添加 drop 规则

```bash
go run dropRule.go n3
```

这将把 "n3" 接口设置为 down 状态，从而 drop 所有通过该接口的 packet。

### 编译后使用

```bash
# 编译程序
go build -o dropRule dropRule.go

# 运行程序
sudo ./dropRule n3
```

## 注意事项

1. **需要 root 权限**: 修改网络接口状态需要 root 权限
2. **接口名称**: 确保指定的接口名称存在
3. **影响**: 设置接口为 down 会中断该接口的所有网络通信

## 代码说明

程序使用以下方法来实现 drop 功能：

1. **系统调用**: 使用 `syscall.Socket` 和 `syscall.SYS_IOCTL` 直接操作网络接口
2. **接口标志**: 通过清除 `IFF_UP` 标志来设置接口为 down 状态
3. **无外部依赖**: 只使用 Go 标准库，不依赖外部包

## 恢复接口

如果需要恢复接口，可以修改代码中的 `removeDropRule` 函数，或者使用系统命令：

```bash
sudo ip link set n3 up
```

## 安全提示

- 请确保在测试环境中使用此工具
- 不要在生产环境中随意 drop 重要的网络接口
- 建议在操作前备份网络配置 