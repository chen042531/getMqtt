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

// CollectEvent 表示一个收集事件
type CollectEvent struct {
	Timestamp time.Time
	Type      string // "Start" 或 "Finish"
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

// 分析收集事件的配对情况
func analyzeCollectEvents(events []CollectEvent) {
	if len(events) == 0 {
		fmt.Println("未找到任何收集事件")
		return
	}

	fmt.Printf("总共找到 %d 个收集事件\n", len(events))

	// 统计Start和Finish事件
	var startEvents, finishEvents []CollectEvent
	for _, event := range events {
		if event.Type == "Start" {
			startEvents = append(startEvents, event)
		} else if event.Type == "Finish" {
			finishEvents = append(finishEvents, event)
		}
	}

	fmt.Printf("Start Collect 事件: %d 个\n", len(startEvents))
	fmt.Printf("Finish Collect 事件: %d 个\n", len(finishEvents))

	// 检查配对情况
	if len(startEvents) != len(finishEvents) {
		fmt.Printf("❌ Start和Finish事件数量不匹配！\n")
		fmt.Printf("   缺少 %d 个事件\n", abs(len(startEvents)-len(finishEvents)))
	} else {
		fmt.Printf("✅ Start和Finish事件数量匹配\n")
	}

	// 分析Start事件间隔
	if len(startEvents) > 1 {
		fmt.Printf("\n=== Start Collect 间隔分析 ===\n")
		analyzeIntervals(startEvents, "Start")
	}

	// 分析Finish事件间隔
	if len(finishEvents) > 1 {
		fmt.Printf("\n=== Finish Collect 间隔分析 ===\n")
		analyzeIntervals(finishEvents, "Finish")
	}

	// 分析Start到Finish的持续时间
	if len(startEvents) > 0 && len(finishEvents) > 0 {
		fmt.Printf("\n=== Start到Finish持续时间分析 ===\n")
		analyzeStartToFinishDuration(startEvents, finishEvents)
	}
}

// 分析时间间隔
func analyzeIntervals(events []CollectEvent, eventType string) {
	if len(events) < 2 {
		fmt.Printf("需要至少两个%s事件来分析间隔\n", eventType)
		return
	}

	var intervals []time.Duration
	var totalDuration time.Duration

	for i := 1; i < len(events); i++ {
		interval := events[i].Timestamp.Sub(events[i-1].Timestamp)
		intervals = append(intervals, interval)
		totalDuration += interval

		fmt.Printf("%s事件 %d -> %d: %v\n",
			eventType, i, i+1, interval)
	}

	// 计算统计信息
	avgInterval := totalDuration / time.Duration(len(intervals))

	fmt.Printf("\n--- %s事件统计信息 ---\n", eventType)
	fmt.Printf("平均间隔: %v\n", avgInterval)
	fmt.Printf("预期间隔: 15秒\n")
	fmt.Printf("偏差: %v\n", avgInterval-15*time.Second)

	// 检查是否接近15秒
	tolerance := 2 * time.Second
	if avgInterval >= 15*time.Second-tolerance && avgInterval <= 15*time.Second+tolerance {
		fmt.Printf("✅ %s事件平均间隔接近15秒 (在±2秒容差范围内)\n", eventType)
	} else {
		fmt.Printf("❌ %s事件平均间隔与15秒有显著差异\n", eventType)
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

// 分析Start到Finish的持续时间
func analyzeStartToFinishDuration(startEvents, finishEvents []CollectEvent) {
	minLen := len(startEvents)
	if len(finishEvents) < minLen {
		minLen = len(finishEvents)
	}

	if minLen == 0 {
		fmt.Println("没有足够的Start和Finish事件进行配对分析")
		return
	}

	var durations []time.Duration
	var totalDuration time.Duration

	fmt.Printf("分析前 %d 对Start-Finish事件:\n", minLen)

	for i := 0; i < minLen; i++ {
		duration := finishEvents[i].Timestamp.Sub(startEvents[i].Timestamp)
		durations = append(durations, duration)
		totalDuration += duration

		fmt.Printf("配对 %d: Start(%s) -> Finish(%s) = %v\n",
			i+1,
			startEvents[i].Timestamp.Format("15:04:05.000"),
			finishEvents[i].Timestamp.Format("15:04:05.000"),
			duration)
	}

	// 计算统计信息
	avgDuration := totalDuration / time.Duration(len(durations))

	fmt.Printf("\n--- Start到Finish持续时间统计 ---\n")
	fmt.Printf("平均持续时间: %v\n", avgDuration)

	// 找出最大和最小持续时间
	if len(durations) > 0 {
		minDuration := durations[0]
		maxDuration := durations[0]

		for _, duration := range durations {
			if duration < minDuration {
				minDuration = duration
			}
			if duration > maxDuration {
				maxDuration = duration
			}
		}

		fmt.Printf("最小持续时间: %v\n", minDuration)
		fmt.Printf("最大持续时间: %v\n", maxDuration)
	}
}

// 辅助函数：计算绝对值
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func main() {
	// 打开日志文件
	file, err := os.Open("csm.log")
	if err != nil {
		log.Fatal("无法打开文件:", err)
	}
	defer file.Close()

	// 编译正则表达式来匹配Start和Finish Collect事件
	startCollectPattern := regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d+Z).*Start Collect$`)
	finishCollectPattern := regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d+Z).*Finish Collect$`)

	var events []CollectEvent
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// 检查Start Collect事件
		if strings.Contains(line, "Start Collect") {
			matches := startCollectPattern.FindStringSubmatch(line)
			if len(matches) >= 2 {
				timestamp, err := parseTimestamp(matches[1])
				if err != nil {
					fmt.Printf("警告: 无法解析第 %d 行的时间戳: %v\n", lineNum, err)
					continue
				}

				events = append(events, CollectEvent{
					Timestamp: timestamp,
					Type:      "Start",
					Line:      line,
				})

				fmt.Printf("找到Start事件 #%d: %s\n", len(events), timestamp.Format("2006-01-02 15:04:05"))
			}
		}

		// 检查Finish Collect事件
		if strings.Contains(line, "Finish Collect") {
			matches := finishCollectPattern.FindStringSubmatch(line)
			if len(matches) >= 2 {
				timestamp, err := parseTimestamp(matches[1])
				if err != nil {
					fmt.Printf("警告: 无法解析第 %d 行的时间戳: %v\n", lineNum, err)
					continue
				}

				events = append(events, CollectEvent{
					Timestamp: timestamp,
					Type:      "Finish",
					Line:      line,
				})

				fmt.Printf("找到Finish事件 #%d: %s\n", len(events), timestamp.Format("2006-01-02 15:04:05"))
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal("读取文件时出错:", err)
	}

	if len(events) == 0 {
		fmt.Println("未找到任何收集事件")
		return
	}

	fmt.Printf("\n=== 分析结果 ===\n")
	analyzeCollectEvents(events)
}
