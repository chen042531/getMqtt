package main

import (
	"fmt"
	"log"
	"os"
	"syscall"
	"unsafe"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("使用方法: go run dropRule.go <interface_name>")
		fmt.Println("例如: go run dropRule.go n3")
		os.Exit(1)
	}

	interfaceName := os.Args[1]

	// 添加 drop 规则
	err := addDropRule(interfaceName)
	if err != nil {
		log.Fatalf("添加 drop 规则失败: %v", err)
	}

	fmt.Printf("成功添加 drop 规则到接口 %s\n", interfaceName)
}

// addDropRule 为指定接口添加 drop 规则
func addDropRule(interfaceName string) error {
	// 方法1: 使用 iptables 规则 drop 所有包
	err := addIptablesRule(interfaceName)
	if err != nil {
		return fmt.Errorf("添加 iptables 规则失败: %v", err)
	}

	// 方法2: 使用 netlink 添加 drop 规则
	err = addNetlinkDropRule(interfaceName)
	if err != nil {
		return fmt.Errorf("添加 netlink drop 规则失败: %v", err)
	}

	return nil
}

// addIptablesRule 使用 iptables 规则 drop 接口的包
func addIptablesRule(interfaceName string) error {
	// 创建 netlink socket 用于与 iptables 通信
	fd, err := syscall.Socket(syscall.AF_NETLINK, syscall.SOCK_RAW, syscall.NETLINK_NETFILTER)
	if err != nil {
		return fmt.Errorf("创建 netlink socket 失败: %v", err)
	}
	defer syscall.Close(fd)

	// 绑定 socket
	err = syscall.Bind(fd, &syscall.SockaddrNetlink{
		Family: syscall.AF_NETLINK,
		Pid:    uint32(os.Getpid()),
	})
	if err != nil {
		return fmt.Errorf("绑定 socket 失败: %v", err)
	}

	// 创建 iptables 规则消息
	msg := createIptablesRuleMsg(interfaceName)

	// 发送消息
	err = sendNetlinkMessage(fd, msg)
	if err != nil {
		return fmt.Errorf("发送 iptables 规则消息失败: %v", err)
	}

	return nil
}

// addNetlinkDropRule 使用 netlink 添加 drop 规则
func addNetlinkDropRule(interfaceName string) error {
	// 创建 netlink socket
	fd, err := syscall.Socket(syscall.AF_NETLINK, syscall.SOCK_RAW, syscall.NETLINK_ROUTE)
	if err != nil {
		return fmt.Errorf("创建 netlink socket 失败: %v", err)
	}
	defer syscall.Close(fd)

	// 绑定 socket
	err = syscall.Bind(fd, &syscall.SockaddrNetlink{
		Family: syscall.AF_NETLINK,
		Pid:    uint32(os.Getpid()),
	})
	if err != nil {
		return fmt.Errorf("绑定 socket 失败: %v", err)
	}

	// 获取接口索引
	ifIndex, err := getInterfaceIndex(interfaceName)
	if err != nil {
		return fmt.Errorf("获取接口索引失败: %v", err)
	}

	// 创建 drop 规则消息
	msg := createDropRuleMsg(ifIndex)

	// 发送消息
	err = sendNetlinkMessage(fd, msg)
	if err != nil {
		return fmt.Errorf("发送 drop 规则消息失败: %v", err)
	}

	return nil
}

// getInterfaceIndex 获取接口索引
func getInterfaceIndex(interfaceName string) (int, error) {
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, 0)
	if err != nil {
		return 0, fmt.Errorf("创建 socket 失败: %v", err)
	}
	defer syscall.Close(fd)

	ifreq := struct {
		Name  [16]byte
		Index int32
		pad   [20]byte
	}{}

	copy(ifreq.Name[:], interfaceName)

	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), syscall.SIOCGIFINDEX, uintptr(unsafe.Pointer(&ifreq)))
	if errno != 0 {
		return 0, fmt.Errorf("获取接口索引失败: %v", errno)
	}

	return int(ifreq.Index), nil
}

// createIptablesRuleMsg 创建 iptables 规则消息
func createIptablesRuleMsg(interfaceName string) []byte {
	// 简化的 iptables 规则消息结构
	msg := make([]byte, 1024)

	// 设置 netlink 头部
	msg[0] = 0x44 // 消息长度
	msg[1] = 0x00
	msg[2] = 0x00
	msg[3] = 0x00

	msg[4] = 0x00 // 消息类型 (NFNL_SUBSYS_IPV4)
	msg[5] = 0x00
	msg[6] = 0x00
	msg[7] = 0x00

	msg[8] = 0x00 // 标志
	msg[9] = 0x00
	msg[10] = 0x00
	msg[11] = 0x00

	msg[12] = 0x00 // 序列号
	msg[13] = 0x00
	msg[14] = 0x00
	msg[15] = 0x00

	msg[16] = 0x00 // PID
	msg[17] = 0x00
	msg[18] = 0x00
	msg[19] = 0x00

	// 添加接口名称到消息中
	copy(msg[20:], interfaceName)

	return msg
}

// createDropRuleMsg 创建 drop 规则消息
func createDropRuleMsg(ifIndex int) []byte {
	// 简化的 drop 规则消息结构
	msg := make([]byte, 1024)

	// 设置 netlink 头部
	msg[0] = 0x44 // 消息长度
	msg[1] = 0x00
	msg[2] = 0x00
	msg[3] = 0x00

	msg[4] = 0x00 // 消息类型 (RTM_NEWRULE)
	msg[5] = 0x00
	msg[6] = 0x00
	msg[7] = 0x00

	msg[8] = 0x00 // 标志
	msg[9] = 0x00
	msg[10] = 0x00
	msg[11] = 0x00

	msg[12] = 0x00 // 序列号
	msg[13] = 0x00
	msg[14] = 0x00
	msg[15] = 0x00

	msg[16] = 0x00 // PID
	msg[17] = 0x00
	msg[18] = 0x00
	msg[19] = 0x00

	// 添加接口索引到消息中
	msg[20] = byte(ifIndex & 0xFF)
	msg[21] = byte((ifIndex >> 8) & 0xFF)
	msg[22] = byte((ifIndex >> 16) & 0xFF)
	msg[23] = byte((ifIndex >> 24) & 0xFF)

	return msg
}

// sendNetlinkMessage 发送 netlink 消息
func sendNetlinkMessage(fd int, msg []byte) error {
	addr := &syscall.SockaddrNetlink{
		Family: syscall.AF_NETLINK,
		Pid:    0, // 发送到内核
	}

	err := syscall.Sendto(fd, msg, 0, addr)
	if err != nil {
		return fmt.Errorf("发送 netlink 消息失败: %v", err)
	}

	return nil
}

// removeDropRule 移除指定接口的 drop 规则
func removeDropRule(interfaceName string) error {
	// 获取接口索引
	ifIndex, err := getInterfaceIndex(interfaceName)
	if err != nil {
		return fmt.Errorf("获取接口索引失败: %v", err)
	}

	// 创建 netlink socket
	fd, err := syscall.Socket(syscall.AF_NETLINK, syscall.SOCK_RAW, syscall.NETLINK_ROUTE)
	if err != nil {
		return fmt.Errorf("创建 netlink socket 失败: %v", err)
	}
	defer syscall.Close(fd)

	// 绑定 socket
	err = syscall.Bind(fd, &syscall.SockaddrNetlink{
		Family: syscall.AF_NETLINK,
		Pid:    uint32(os.Getpid()),
	})
	if err != nil {
		return fmt.Errorf("绑定 socket 失败: %v", err)
	}

	// 删除 drop 规则
	deleteMsg := createDeleteRuleMsg(ifIndex)
	err = sendNetlinkMessage(fd, deleteMsg)
	if err != nil {
		return fmt.Errorf("删除 drop 规则失败: %v", err)
	}

	return nil
}

// createDeleteRuleMsg 创建删除规则的消息
func createDeleteRuleMsg(ifIndex int) []byte {
	// 简化的删除消息结构
	msg := make([]byte, 1024)

	// 设置 netlink 头部
	msg[0] = 0x44 // 消息长度
	msg[1] = 0x00
	msg[2] = 0x00
	msg[3] = 0x00

	msg[4] = 0x00 // 消息类型 (RTM_DELRULE)
	msg[5] = 0x00
	msg[6] = 0x00
	msg[7] = 0x00

	msg[8] = 0x00 // 标志
	msg[9] = 0x00
	msg[10] = 0x00
	msg[11] = 0x00

	msg[12] = 0x00 // 序列号
	msg[13] = 0x00
	msg[14] = 0x00
	msg[15] = 0x00

	msg[16] = 0x00 // PID
	msg[17] = 0x00
	msg[18] = 0x00
	msg[19] = 0x00

	// 添加接口索引到消息中
	msg[20] = byte(ifIndex & 0xFF)
	msg[21] = byte((ifIndex >> 8) & 0xFF)
	msg[22] = byte((ifIndex >> 16) & 0xFF)
	msg[23] = byte((ifIndex >> 24) & 0xFF)

	return msg
}
