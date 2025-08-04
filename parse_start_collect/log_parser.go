package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"time"
)

// StartCollectEvent 表示一个Start Collect事件
type StartCollectEvent struct {
	Timestamp time.Time
	Line      string
}

// 解析时间戳的函数
func parseTimestamp(timestampStr string) (time.Time, error) {
	// 移除颜色代码
	cleanStr := strings.TrimSpace(timestampStr)

	// 解析ISO 8601格式的时间戳
	layout := "2006-01-02T15:04:05.999999999Z"
	return time.Parse(layout, cleanStr)
}

// 分析时间间隔
func analyzeIntervals(events []StartCollectEvent) {
	if len(events) < 2 {
		fmt.Println("需要至少两个事件来分析间隔")
		return
	}

	fmt.Printf("总共找到 %d 个 Start Collect 事件\n\n", len(events))

	var intervals []time.Duration
	var totalDuration time.Duration

	for i := 1; i < len(events); i++ {
		interval := events[i].Timestamp.Sub(events[i-1].Timestamp)
		intervals = append(intervals, interval)
		totalDuration += interval

		fmt.Printf("事件 %d -> %d: %v\n",
			i, i+1, interval)
	}

	// 计算统计信息
	avgInterval := totalDuration / time.Duration(len(intervals))

	fmt.Printf("\n=== 统计信息 ===\n")
	fmt.Printf("平均间隔: %v\n", avgInterval)
	fmt.Printf("预期间隔: 15秒\n")
	fmt.Printf("偏差: %v\n", avgInterval-15*time.Second)

	// 检查是否接近15秒
	tolerance := 2 * time.Second
	if avgInterval >= 15*time.Second-tolerance && avgInterval <= 15*time.Second+tolerance {
		fmt.Printf("✅ 平均间隔接近15秒 (在±2秒容差范围内)\n")
	} else {
		fmt.Printf("❌ 平均间隔与15秒有显著差异\n")
	}

	// 找出最大和最小间隔
	if len(intervals) > 0 {
		minInterval := intervals[0]
		maxInterval := intervals[0]

		for _, interval := range intervals {
			if interval < minInterval {
				minInterval = interval
			}
			if interval > maxInterval {
				maxInterval = interval
			}
		}

		fmt.Printf("最小间隔: %v\n", minInterval)
		fmt.Printf("最大间隔: %v\n", maxInterval)
	}
}

func main() {
	// 打开日志文件
	file, err := os.Open("csm.log")
	if err != nil {
		log.Fatal("无法打开文件:", err)
	}
	defer file.Close()

	// 编译正则表达式来匹配Start Collect事件
	startCollectPattern := regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d+Z).*Start Collect$`)

	var events []StartCollectEvent
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// 检查是否包含"Start Collect"
		if strings.Contains(line, "Start Collect") {
			matches := startCollectPattern.FindStringSubmatch(line)
			if len(matches) >= 2 {
				timestamp, err := parseTimestamp(matches[1])
				if err != nil {
					fmt.Printf("警告: 无法解析第 %d 行的时间戳: %v\n", lineNum, err)
					continue
				}

				events = append(events, StartCollectEvent{
					Timestamp: timestamp,
					Line:      line,
				})

				fmt.Printf("找到事件 #%d: %s\n", len(events), timestamp.Format("2006-01-02 15:04:05"))
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal("读取文件时出错:", err)
	}

	if len(events) == 0 {
		fmt.Println("未找到任何 Start Collect 事件")
		return
	}

	fmt.Printf("\n=== 分析结果 ===\n")
	analyzeIntervals(events)
}
