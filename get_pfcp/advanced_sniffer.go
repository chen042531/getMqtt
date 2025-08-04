package main

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// PFCP消息类型
const (
	PFCP_HEARTBEAT_REQUEST   = 1
	PFCP_HEARTBEAT_RESPONSE  = 2
	PFCP_PFD_MANAGEMENT      = 3
	PFCP_ASSOCIATION_SETUP   = 4
	PFCP_ASSOCIATION_RELEASE = 5
	PFCP_NODE_REPORT         = 6
	PFCP_SESSION_SETUP       = 7
	PFCP_SESSION_MODIFY      = 8
	PFCP_SESSION_DELETE      = 9
	PFCP_SESSION_REPORT      = 10
)

// PFCP消息结构
type PFCPMessage struct {
	Version     uint8
	MessageType uint8
	Length      uint16
	SEID        uint64
	Sequence    uint32
	Timestamp   time.Time
	SourceIP    net.IP
	DestIP      net.IP
	SourcePort  uint16
	DestPort    uint16
	RawData     []byte
	PacketID    int
	TTL         int
}

// 高级监听器
type AdvancedSniffer struct {
	stats    *PFCPStats
	stopChan chan bool
	packetID int
	filters  *PacketFilters
	display  *DisplayOptions
}

// 数据包过滤器
type PacketFilters struct {
	SourceIP    string
	DestIP      string
	MessageType int
	MinLength   int
	MaxLength   int
	ShowHex     bool
	ShowRaw     bool
	ShowStats   bool
	ShowDetails bool
}

// 显示选项
type DisplayOptions struct {
	ShowPacketList bool
	ShowDetails    bool
	ShowHex        bool
	ShowRaw        bool
	ShowStats      bool
	CompactMode    bool
}

// PFCP统计信息
type PFCPStats struct {
	TotalMessages      int
	HeartbeatCount     int
	SessionSetupCount  int
	SessionDeleteCount int
	AssociationCount   int
	LastMessageTime    time.Time
	MessageTypes       map[uint8]int
	SourceIPs          map[string]int
	DestIPs            map[string]int
	PacketSizes        map[int]int
	StartTime          time.Time
}

// 创建新的高级监听器
func NewAdvancedSniffer() *AdvancedSniffer {
	return &AdvancedSniffer{
		stopChan: make(chan bool),
		packetID: 1,
		filters: &PacketFilters{
			ShowHex: true,
			ShowRaw: true,
		},
		display: &DisplayOptions{
			ShowPacketList: true,
			ShowDetails:    true,
			ShowHex:        true,
			ShowRaw:        true,
			ShowStats:      true,
		},
		stats: &PFCPStats{
			MessageTypes: make(map[uint8]int),
			SourceIPs:    make(map[string]int),
			DestIPs:      make(map[string]int),
			PacketSizes:  make(map[int]int),
			StartTime:    time.Now(),
		},
	}
}

// 开始监听
func (a *AdvancedSniffer) StartCapture() error {
	fmt.Println("=== PFCP 高级监听器 (Wireshark风格) ===")
	fmt.Printf("开始时间: %s\n", a.stats.StartTime.Format("2006-01-02 15:04:05"))
	fmt.Println("监听端口: 8805")
	fmt.Println("按Ctrl+C停止监听...")
	fmt.Println()

	// 创建UDP监听器
	addr := &net.UDPAddr{
		IP:   net.IPv4(0, 0, 0, 0),
		Port: 8805,
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return fmt.Errorf("无法监听UDP端口: %v", err)
	}
	defer conn.Close()

	// 设置缓冲区
	buffer := make([]byte, 4096)

	for {
		select {
		case <-a.stopChan:
			return nil
		default:
			// 设置读取超时
			conn.SetReadDeadline(time.Now().Add(1 * time.Second))

			n, remoteAddr, err := conn.ReadFromUDP(buffer)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				continue
			}

			// 解析PFCP消息
			pfcpMsg := a.parsePFCPMessage(buffer[:n], remoteAddr)
			if pfcpMsg != nil {
				// 应用过滤器
				if a.applyFilters(pfcpMsg) {
					a.updateStats(pfcpMsg)
					a.printPacket(pfcpMsg)
				}
			}
		}
	}
}

// 解析PFCP消息
func (a *AdvancedSniffer) parsePFCPMessage(data []byte, remoteAddr *net.UDPAddr) *PFCPMessage {
	if len(data) < 8 {
		return nil
	}

	msg := &PFCPMessage{
		Version:     (data[0] >> 5) & 0x07,
		MessageType: data[1],
		Length:      binary.BigEndian.Uint16(data[2:4]),
		Timestamp:   time.Now(),
		SourceIP:    remoteAddr.IP,
		SourcePort:  uint16(remoteAddr.Port),
		RawData:     make([]byte, len(data)),
		PacketID:    a.packetID,
	}

	a.packetID++

	// 复制原始数据
	copy(msg.RawData, data)

	// 解析SEID（如果存在）
	if len(data) >= 16 {
		msg.SEID = binary.BigEndian.Uint64(data[8:16])
	}

	// 解析序列号（如果存在）
	if len(data) >= 20 {
		msg.Sequence = binary.BigEndian.Uint32(data[16:20])
	}

	return msg
}

// 应用过滤器
func (a *AdvancedSniffer) applyFilters(msg *PFCPMessage) bool {
	// 源IP过滤
	if a.filters.SourceIP != "" && msg.SourceIP.String() != a.filters.SourceIP {
		return false
	}

	// 目标IP过滤
	if a.filters.DestIP != "" && msg.DestIP.String() != a.filters.DestIP {
		return false
	}

	// 消息类型过滤
	if a.filters.MessageType > 0 && int(msg.MessageType) != a.filters.MessageType {
		return false
	}

	// 长度过滤
	if a.filters.MinLength > 0 && len(msg.RawData) < a.filters.MinLength {
		return false
	}

	if a.filters.MaxLength > 0 && len(msg.RawData) > a.filters.MaxLength {
		return false
	}

	return true
}

// 更新统计信息
func (a *AdvancedSniffer) updateStats(msg *PFCPMessage) {
	a.stats.TotalMessages++
	a.stats.LastMessageTime = msg.Timestamp
	a.stats.MessageTypes[msg.MessageType]++

	// 统计IP地址
	a.stats.SourceIPs[msg.SourceIP.String()]++
	if msg.DestIP != nil {
		a.stats.DestIPs[msg.DestIP.String()]++
	}

	// 统计数据包大小
	a.stats.PacketSizes[len(msg.RawData)]++

	switch msg.MessageType {
	case PFCP_HEARTBEAT_REQUEST, PFCP_HEARTBEAT_RESPONSE:
		a.stats.HeartbeatCount++
	case PFCP_SESSION_SETUP:
		a.stats.SessionSetupCount++
	case PFCP_SESSION_DELETE:
		a.stats.SessionDeleteCount++
	case PFCP_ASSOCIATION_SETUP:
		a.stats.AssociationCount++
	}
}

// 打印数据包（Wireshark风格）
func (a *AdvancedSniffer) printPacket(msg *PFCPMessage) {
	msgTypeStr := a.getMessageTypeString(msg.MessageType)

	if a.display.CompactMode {
		// 紧凑模式显示
		fmt.Printf("[%s] #%d %s %s:%d -> %s:%d (%d bytes)\n",
			msg.Timestamp.Format("15:04:05.000"),
			msg.PacketID,
			msgTypeStr,
			msg.SourceIP, msg.SourcePort,
			msg.DestIP, msg.DestPort,
			len(msg.RawData))
		return
	}

	// 详细模式显示
	fmt.Printf("\n%s\n", strings.Repeat("=", 80))
	fmt.Printf("数据包 #%d\n", msg.PacketID)
	fmt.Printf("时间戳: %s\n", msg.Timestamp.Format("2006-01-02 15:04:05.000"))
	fmt.Printf("源地址: %s:%d\n", msg.SourceIP, msg.SourcePort)
	if msg.DestIP != nil {
		fmt.Printf("目标地址: %s:%d\n", msg.DestIP, msg.DestPort)
	}
	fmt.Printf("协议: PFCP\n")
	fmt.Printf("长度: %d 字节\n", len(msg.RawData))

	// PFCP消息详情
	if a.display.ShowDetails {
		fmt.Printf("\nPFCP消息详情:\n")
		fmt.Printf("  版本: %d\n", msg.Version)
		fmt.Printf("  消息类型: %d (%s)\n", msg.MessageType, msgTypeStr)
		fmt.Printf("  长度: %d\n", msg.Length)
		fmt.Printf("  SEID: %d\n", msg.SEID)
		fmt.Printf("  序列号: %d\n", msg.Sequence)
	}

	// 十六进制显示
	if a.display.ShowHex {
		fmt.Printf("\n十六进制数据:\n")
		hexData := hex.EncodeToString(msg.RawData)
		for i := 0; i < len(hexData); i += 32 {
			end := i + 32
			if end > len(hexData) {
				end = len(hexData)
			}
			fmt.Printf("  %s\n", hexData[i:end])
		}
	}

	// 原始数据显示
	if a.display.ShowRaw {
		fmt.Printf("\n原始数据:\n")
		for i, b := range msg.RawData {
			if i%16 == 0 {
				if i > 0 {
					fmt.Println()
				}
				fmt.Printf("  %04x: ", i)
			}
			fmt.Printf("%02x ", b)
		}
		fmt.Println()
	}

	fmt.Printf("%s\n", strings.Repeat("=", 80))
}

// 获取消息类型字符串
func (a *AdvancedSniffer) getMessageTypeString(msgType uint8) string {
	switch msgType {
	case PFCP_HEARTBEAT_REQUEST:
		return "心跳请求"
	case PFCP_HEARTBEAT_RESPONSE:
		return "心跳响应"
	case PFCP_PFD_MANAGEMENT:
		return "PFD管理"
	case PFCP_ASSOCIATION_SETUP:
		return "关联建立"
	case PFCP_ASSOCIATION_RELEASE:
		return "关联释放"
	case PFCP_NODE_REPORT:
		return "节点报告"
	case PFCP_SESSION_SETUP:
		return "会话建立"
	case PFCP_SESSION_MODIFY:
		return "会话修改"
	case PFCP_SESSION_DELETE:
		return "会话删除"
	case PFCP_SESSION_REPORT:
		return "会话报告"
	default:
		return fmt.Sprintf("未知类型(%d)", msgType)
	}
}

// 打印详细统计信息
func (a *AdvancedSniffer) PrintDetailedStats() {
	duration := time.Since(a.stats.StartTime)

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("PFCP流量详细统计")
	fmt.Println(strings.Repeat("=", 80))

	fmt.Printf("监听开始时间: %s\n", a.stats.StartTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("监听持续时间: %s\n", duration.String())
	fmt.Printf("总数据包数: %d\n", a.stats.TotalMessages)
	fmt.Printf("平均速率: %.2f 包/秒\n", float64(a.stats.TotalMessages)/duration.Seconds())
	fmt.Printf("最后消息时间: %s\n", a.stats.LastMessageTime.Format("2006-01-02 15:04:05"))

	if a.stats.TotalMessages == 0 {
		fmt.Println("\n未捕获到任何PFCP数据包")
		return
	}

	fmt.Println("\n消息类型分布:")
	for msgType, count := range a.stats.MessageTypes {
		percentage := float64(count) / float64(a.stats.TotalMessages) * 100
		fmt.Printf("  %s: %d (%.1f%%)\n", a.getMessageTypeString(msgType), count, percentage)
	}

	fmt.Println("\n源IP地址分布:")
	for ip, count := range a.stats.SourceIPs {
		percentage := float64(count) / float64(a.stats.TotalMessages) * 100
		fmt.Printf("  %s: %d (%.1f%%)\n", ip, count, percentage)
	}

	fmt.Println("\n目标IP地址分布:")
	for ip, count := range a.stats.DestIPs {
		percentage := float64(count) / float64(a.stats.TotalMessages) * 100
		fmt.Printf("  %s: %d (%.1f%%)\n", ip, count, percentage)
	}

	fmt.Println("\n数据包大小分布:")
	for size, count := range a.stats.PacketSizes {
		percentage := float64(count) / float64(a.stats.TotalMessages) * 100
		fmt.Printf("  %d 字节: %d (%.1f%%)\n", size, count, percentage)
	}

	fmt.Println("\n特定消息统计:")
	fmt.Printf("  心跳消息: %d\n", a.stats.HeartbeatCount)
	fmt.Printf("  会话建立: %d\n", a.stats.SessionSetupCount)
	fmt.Printf("  会话删除: %d\n", a.stats.SessionDeleteCount)
	fmt.Printf("  关联建立: %d\n", a.stats.AssociationCount)
}

// 设置过滤器
func (a *AdvancedSniffer) SetFilters(sourceIP, destIP, msgType string) {
	if sourceIP != "" {
		a.filters.SourceIP = sourceIP
	}
	if destIP != "" {
		a.filters.DestIP = destIP
	}
	if msgType != "" {
		if mt, err := strconv.Atoi(msgType); err == nil {
			a.filters.MessageType = mt
		}
	}
}

// 设置显示选项
func (a *AdvancedSniffer) SetDisplayOptions(compact bool) {
	a.display.CompactMode = compact
}

// 停止监听
func (a *AdvancedSniffer) Stop() {
	close(a.stopChan)
}

// 主函数
func main() {
	// 检查是否以root权限运行
	if os.Geteuid() != 0 {
		fmt.Println("警告: 此程序需要root权限来捕获网络数据包")
		fmt.Println("请使用 'sudo go run advanced_sniffer.go' 运行")
		fmt.Println()
	}

	// 创建高级监听器
	sniffer := NewAdvancedSniffer()

	// 解析命令行参数
	if len(os.Args) > 1 {
		for i, arg := range os.Args[1:] {
			switch arg {
			case "-c", "--compact":
				sniffer.SetDisplayOptions(true)
			case "-f", "--filter":
				if i+1 < len(os.Args) {
					sniffer.SetFilters(os.Args[i+2], "", "")
				}
			}
		}
	}

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 定期打印统计信息
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if sniffer.display.ShowStats {
					sniffer.PrintDetailedStats()
				}
			case <-sniffer.stopChan:
				return
			}
		}
	}()

	// 启动捕获
	go func() {
		if err := sniffer.StartCapture(); err != nil {
			log.Fatal("启动捕获失败:", err)
		}
	}()

	// 等待中断信号
	<-sigChan
	fmt.Println("\n正在停止PFCP监听器...")
	sniffer.Stop()
	sniffer.PrintDetailedStats()
}
