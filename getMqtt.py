import paho.mqtt.client as mqtt

# 連線成功時的 callback
def on_connect(client, userdata, flags, rc):
    print("事件: on_connect")
    client.subscribe("#")  # 訂閱所有主題

# 斷線時的 callback
def on_disconnect(client, userdata, rc):
    print("事件: on_disconnect")

# 收到訊息時的 callback
def on_message(client, userdata, msg):
    print("事件: on_message")
    print(f"主題: {msg.topic}, 訊息: {msg.payload.decode()}")

# 訂閱成功時的 callback
def on_subscribe(client, userdata, mid, granted_qos):
    print("事件: on_subscribe")

# 取消訂閱時的 callback
def on_unsubscribe(client, userdata, mid):
    print("事件: on_unsubscribe")

# 發佈完成時的 callback
def on_publish(client, userdata, mid):
    print("事件: on_publish")

client = mqtt.Client()
client.on_connect = on_connect
client.on_disconnect = on_disconnect
client.on_message = on_message
client.on_subscribe = on_subscribe
client.on_unsubscribe = on_unsubscribe
client.on_publish = on_publish

client.connect("10.1.153.187", 1883, 60)
client.loop_forever()
