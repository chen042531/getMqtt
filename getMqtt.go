package main

import (
	"fmt"
	"os"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func main() {
	// 設定各種事件的 callback
	opts := mqtt.NewClientOptions().AddBroker("tcp://10.1.153..187:1883")
	opts.SetClientID("go_mqtt_client")

	// 連線成功
	opts.OnConnect = func(c mqtt.Client) {
		fmt.Println("事件: OnConnect")
		if token := c.Subscribe("#", 0, nil); token.Wait() && token.Error() != nil {
			fmt.Println("訂閱失敗:", token.Error())
		}
	}

	// 斷線
	opts.OnConnectionLost = func(c mqtt.Client, err error) {
		fmt.Println("事件: OnConnectionLost")
		fmt.Println("錯誤:", err)
	}

	// 收到訊息
	messagePubHandler := func(client mqtt.Client, msg mqtt.Message) {
		fmt.Println("事件: OnMessageReceived")
		fmt.Printf("主題: %s, 訊息: %s\n", msg.Topic(), msg.Payload())
	}

	// 建立 client
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		fmt.Println("連線失敗:", token.Error())
		os.Exit(1)
	}

	// 設定收到訊息的 callback
	client.AddRoute("#", messagePubHandler)

	// 持續運作
	for {
		time.Sleep(1 * time.Second)
	}
}
