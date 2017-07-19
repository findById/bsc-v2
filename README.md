bsc
=======

TCP代理

#### 简介
    * NONE

#### 工程结构
    bsc/
    ├── README.md
    ├── doc               文档
    ├── core              核心,协议
    │   ├── doc               文档
    │   ├── protocol          协议
    │   ├── frame.go          数据帧
    │   └── frame_test.go     数据帧单元测试
    ├── client            客户端实现
    └── server            服务端实现
        ├── main.go           启动入口
        ├── server.go         核心入口
        ├── client            代理客户端管理
        ├── handler           代理客户端处理器
        └── site              对外连接管理