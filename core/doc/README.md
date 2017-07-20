# Protocol

## Frame format

|长度 |类型|通道Id|载体|
|:----:|:----:|:----:|:----:|
|Length (uint32)|Type (uint8)|ChannelId (uint32)|Payload ([]byte)|
|uint24|uint8|uint32|...|

### Type
|KEY |VALUE|DESC|Format|
|:----:|:----:|:----:|:----:|
|DATA|0|数据|[L0,L1,L2,0,CHANNEL_ID,DATA....]|
|NEW_CONNECT|1|打开新连接|[L0,L1,L2,1,0]|
|NEW_CONNECT_ACK|2|新连接ACK|[L0,L1,L2,2,0,REL] REL: 0 success, 1 failed|
|CLOSE_CH|3|关闭通道|[L0,L1,L2,3,CHANNEL_ID]|
|CLOSE_CO|4|关闭链接|[L0,L1,L2,4,0]|
|AUTH|5|请求认证|[L0,L1,L2,5,0,DATA....] DATA:MD5(UID:TOKEN)|
|AUTH_ACK|6|认证ACK|[L0,L1,L2,6,0,REL] REL: 0 success, 1 failed|