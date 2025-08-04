import paho.mqtt.client as mqtt
import threading
import time
import json
from collections import defaultdict

imsi_count = defaultdict(int)
lock = threading.Lock()

def on_connect(client, userdata, flags, rc):
    print("已連線")
    client.subscribe("FiveGC/metric")  # 監聽你要的 topic

def on_message(client, userdata, msg):
    try:
        data = json.loads(msg.payload.decode())
        imsi = data.get("imsi")
        if imsi:
            with lock:
                imsi_count[imsi] += 1
    except Exception as e:
        print("解析失敗", e)

def print_and_reset():
    while True:
        time.sleep(15)
        with lock:
            if imsi_count:
                imsi_list = ', '.join(imsi_count.keys())
                # print(f"這5秒有{len(imsi_count)}個不同的imsi: {imsi_list}")
                print(f"這15秒有{len(imsi_count)}個不同的imsi")
                imsi_count.clear()
            else:
                print("這15秒沒有收到訊息")

client = mqtt.Client()
client.on_connect = on_connect
client.on_message = on_message

client.connect("10.1.153.167", 1883, 60)

threading.Thread(target=print_and_reset, daemon=True).start()

client.loop_forever()
