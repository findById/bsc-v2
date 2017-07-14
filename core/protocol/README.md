# Protocol

## Frame format

|长度 |类型|通道Id|载体|
|:----:|:----:|:----:|:----:|
|Length (uint32)|Type (uint8)|ChannelId (uint8)|Payload ([]byte)|
|uint32|uint8|uint8|...|

### Type
|KEY |VALUE|DESC|
|:----:|:----:|:----:|
|DATA|0|数据|
|NEW_CONNECT|1|打开新连接|
|NEW_CONNECT_ACK|2|新连接ACK|
|CLOSE_CH|3|关闭通道|
|CLOSE_CO|4|关闭链接|
|AUTH|5|请求认证|
|AUTH_ACK|6|认证ACK|