package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

type MetricData struct {
	Imsi string `json:"imsi"`
	IP   string `json:"ip,omitempty"`
}

type PacketStats struct {
	DestinationIP string
	ImsiSet       map[string]bool
	Count         int
}

var (
	// 按目標IP分组的统计
	ipStats = make(map[string]*PacketStats)
	lock    sync.RWMutex

	// 配置参数
	targetIP      = "10.1.153.153" // 目標IP，根據需要修改
	statsInterval = 15 * time.Second
	debugMode     = false // 調試模式
)

func listInterfaces() []pcap.Interface {
	devices, err := pcap.FindAllDevs()
	if err != nil {
		log.Fatal("無法獲取網路介面列表:", err)
	}

	fmt.Println("可用的網路介面:")
	for _, device := range devices {
		fmt.Printf("  %s: %s\n", device.Name, device.Description)
		for _, address := range device.Addresses {
			fmt.Printf("    IP: %s\n", address.IP)
		}
	}
	return devices
}

func capturePacketsOnAny() {
	// 使用 "any" 介面捕獲所有網路流量
	handle, err := pcap.OpenLive("any", 1600, true, pcap.BlockForever)
	if err != nil {
		log.Printf("無法打開 any 介面: %v", err)
		log.Println("嘗試列出可用的網路介面...")
		listInterfaces()
		os.Exit(1)
	}
	defer handle.Close()

	// 設置過濾器，只捕獲發送到目標IP的MQTT流量（端口1883）
	filter := fmt.Sprintf("tcp port 1883 and dst host %s", targetIP)
	err = handle.SetBPFFilter(filter)
	if err != nil {
		log.Fatal("設置BPF過濾器失敗:", err)
	}

	fmt.Printf("開始監控 any 介面的MQTT流量\n")
	fmt.Printf("監控目標IP: %s\n", targetIP)
	fmt.Printf("統計間隔: %v\n", statsInterval)
	fmt.Printf("過濾器: %s\n", filter)

	// 開始捕獲封包
	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	packetCount := 0

	for packet := range packetSource.Packets() {
		packetCount++
		if debugMode && packetCount%10 == 0 {
			fmt.Printf("[any] 已處理 %d 個封包\n", packetCount)
		}
		processPacket(packet)
	}
}

func processPacket(packet gopacket.Packet) {
	// 解析網路層
	networkLayer := packet.NetworkLayer()
	if networkLayer == nil {
		if debugMode {
			log.Println("[any] 無法解析網路層")
		}
		return
	}

	// 檢查是否為IP封包
	ipLayer, ok := networkLayer.(*layers.IPv4)
	if !ok {
		if debugMode {
			log.Println("[any] 不是IPv4封包")
		}
		return
	}

	// 檢查目標IP是否為目標IP
	if ipLayer.DstIP.String() != targetIP {
		if debugMode {
			log.Printf("[any] 目標IP %s 不是監控目標 %s", ipLayer.DstIP.String(), targetIP)
		}
		return
	}

	// 解析傳輸層
	transportLayer := packet.TransportLayer()
	if transportLayer == nil {
		if debugMode {
			log.Println("[any] 無法解析傳輸層")
		}
		return
	}

	// 檢查是否為TCP封包
	tcpLayer, ok := transportLayer.(*layers.TCP)
	if !ok {
		if debugMode {
			log.Println("[any] 不是TCP封包")
		}
		return
	}

	// 檢查是否為MQTT端口
	if tcpLayer.DstPort != 1883 {
		if debugMode {
			log.Printf("[any] 目標端口 %d 不是MQTT端口 1883", tcpLayer.DstPort)
		}
		return
	}

	// 嘗試解析MQTT payload
	payload := tcpLayer.Payload
	if len(payload) == 0 {
		if debugMode {
			log.Println("[any] TCP payload為空")
		}
		return
	}

	// 嘗試解析JSON格式的MQTT消息
	var data MetricData
	err := json.Unmarshal(payload, &data)
	if err != nil {
		if debugMode {
			log.Printf("[any] 無法解析MQTT payload為JSON: %v", err)
			log.Printf("[any] Payload內容: %s", string(payload))
		}
		return
	}

	if data.Imsi != "" {
		lock.Lock()
		defer lock.Unlock()

		destinationIP := ipLayer.DstIP.String()

		// 初始化该IP的统计
		if ipStats[destinationIP] == nil {
			ipStats[destinationIP] = &PacketStats{
				DestinationIP: destinationIP,
				ImsiSet:       make(map[string]bool),
				Count:         0,
			}
		}

		// 更新统计
		ipStats[destinationIP].ImsiSet[data.Imsi] = true
		ipStats[destinationIP].Count++

		log.Printf("[any] 捕獲發送到 %s 的MQTT封包，IMSI: %s", destinationIP, data.Imsi)
	}
}

func printAndReset() {
	for {
		time.Sleep(statsInterval)

		lock.Lock()
		stats := make(map[string]*PacketStats)
		for ip, stat := range ipStats {
			stats[ip] = &PacketStats{
				DestinationIP: stat.DestinationIP,
				ImsiSet:       make(map[string]bool),
				Count:         stat.Count,
			}
			// 复制IMSI集合
			for imsi := range stat.ImsiSet {
				stats[ip].ImsiSet[imsi] = true
			}
		}

		// 清空当前统计
		ipStats = make(map[string]*PacketStats)
		lock.Unlock()

		// 打印统计结果
		if len(stats) > 0 {
			fmt.Printf("\n=== %s 統計報告 ===\n", time.Now().Format("2006-01-02 15:04:05"))
			for ip, stat := range stats {
				if stat.Count > 0 {
					fmt.Printf("目標IP: %s\n", ip)
					fmt.Printf("  封包總數: %d\n", stat.Count)
					fmt.Printf("  獨立IMSI數量: %d\n", len(stat.ImsiSet))
					fmt.Printf("  IMSI列表: ")
					imsiList := make([]string, 0, len(stat.ImsiSet))
					for imsi := range stat.ImsiSet {
						imsiList = append(imsiList, imsi)
					}
					fmt.Printf("%v\n", imsiList)
					fmt.Println()
				}
			}
		} else {
			fmt.Printf("\n[%s] 這%d秒沒有捕獲到發送到目標IP %s 的MQTT封包\n",
				time.Now().Format("15:04:05"),
				int(statsInterval.Seconds()),
				targetIP)
		}
	}
}

func main() {
	fmt.Println("MQTT封包監控工具 (目標IP版本)")
	fmt.Println("================================")

	// 檢查是否為root權限
	if os.Geteuid() != 0 {
		fmt.Println("警告: 此程序需要root權限來捕獲網路封包")
		fmt.Println("請使用 sudo 運行此程序")
		os.Exit(1)
	}

	fmt.Printf("開始監控發送到 %s 的MQTT封包...\n", targetIP)

	// 啟動統計報告協程
	go printAndReset()

	// 開始捕獲 any 介面的封包
	capturePacketsOnAny()
}
