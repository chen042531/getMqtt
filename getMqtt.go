package main

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

type MetricData struct {
	Imsi string `json:"imsi"`
}

var (
	imsiCount = make(map[string]int)
	lock      sync.Mutex
)

func onConnect(client MQTT.Client) {
	fmt.Println("已連線")
	client.Subscribe("FiveGC/metric", 0, nil)
}

func onMessage(client MQTT.Client, msg MQTT.Message) {
	var data MetricData
	err := json.Unmarshal(msg.Payload(), &data)
	if err != nil {
		log.Printf("解析失敗: %v", err)
		return
	}

	if data.Imsi != "" {
		lock.Lock()
		imsiCount[data.Imsi]++
		lock.Unlock()
	}
}

func printAndReset() {
	for {
		time.Sleep(15 * time.Second)
		lock.Lock()
		if len(imsiCount) > 0 {
			fmt.Printf("這15秒有%d個不同的imsi\n", len(imsiCount))
			imsiCount = make(map[string]int)
		} else {
			fmt.Println("這15秒沒有收到訊息")
		}
		lock.Unlock()
	}
}

func main() {
	opts := MQTT.NewClientOptions()
	opts.AddBroker("tcp://10.1.153.161:1883")
	opts.SetDefaultPublishHandler(onMessage)
	opts.OnConnect = onConnect

	client := MQTT.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
	}

	go printAndReset()

	// 保持程式運行
	select {}
}
