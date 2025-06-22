# Go-load-balancer

基于Go实现的轻量级负载均衡器

## 功能特性

- 支持多种负载均衡算法：轮询(Round Robin)、最少连接(Least Connections)、加权轮询(Weighted RR)、IP哈希(IP Hash)
- 健康检查机制，自动剔除故障节点
- 高性能并行健康检查，防止阻塞
- 支持HTTP反向代理
- 可配置的监听地址和端口
- YAML格式配置文件
- Prometheus监控集成
- 实时状态API

## 架构设计

```
+------------------+
|   HTTP请求        |
+--------+---------+
         |
+--------v---------+
|   负载均衡器      |
+--------+---------+
         |
    +----+-----+
    |          |
+---v---+  +---v---+
|后端1   |  |后端2   |
+-------+  +-------+
```

## 快速开始

### 安装

```bash
git clone https://github.com/xcc-d/Go-load-balancer.git
cd Go-load-balancer
go build -o lb cmd/lb/main.go
```

### 运行

```bash
./lb -config configs/example.yaml
```

## 监控与状态

### Prometheus指标

访问`/metrics`端点可获取Prometheus格式指标，包括：
- 请求计数
- 响应时间
- 活动连接数
- 后端状态
- 错误计数

### 状态API

访问`/status`端点可获取负载均衡器当前状态的JSON报告，包括：
- 总请求数
- 活动请求数
- 后端服务状态
- 运行时间

## 详细配置说明

### 基础配置项

```yaml
# 监听地址和端口 (必需)
listen_addr: "127.0.0.1:8080"

# 负载均衡算法 (必需，可选值: round_robin, least_conn, weighted_rr, ip_hash)
algorithm: "round_robin"

# 后端服务器列表 (必需)
servers:
  - url: "http://localhost:8001"  # 后端地址
    weight: 1                     # 权重(加权算法使用)
    health: "healthy"             # 初始状态
    health_check_path: "/health"  # 健康检查路径
    
# 健康检查配置
health_check:
  interval: "10s"     # 检查间隔(如: 10s, 1m)
  timeout: "5s"       # 检查超时时间
  path: "/health"     # 健康检查端点
  retry_count: 3      # 重试次数
  retry_interval: "5s" # 重试间隔
  max_failures: 3     # 最大失败次数后移除服务
```

### 高级配置示例

```yaml
listen_addr: "0.0.0.0:80"
algorithm: "weighted_rr"
servers:
  - url: "http://10.0.0.1:8080"
    weight: 3
  - url: "http://10.0.0.2:8080" 
    weight: 2
  - url: "http://10.0.0.3:8080"
    weight: 1
    
health_check:
  interval: "30s"
  timeout: "10s"
  path: "/api/health"
  retry_count: 3
  retry_interval: "5s"
  max_failures: 3
  
# 日志配置(可选)
log:
  level: "info"  # 日志级别(debug/info/warn/error)
  path: "/var/log/lb.log"
```

#### 注意事项
1. 时间单位支持: ns(纳秒), us(微秒), ms(毫秒), s(秒), m(分钟), h(小时)
2. URL必须包含协议(http://或https://)
3. 权重仅在weighted_rr算法下生效
4. IP哈希算法根据客户端IP决定后端，适合保持会话的场景
5. 修改配置后需重启服务生效

### 测试

```bash
curl http://localhost:8080
```

## 性能优化

### 并行健康检查

本项目实现了高效的并行健康检查机制，具有以下特点：

- **并行执行**：对所有后端服务器并行执行健康检查，大幅降低总检查时间
- **锁优化**：通过快照机制减少锁持有时间，显著降低锁争用
- **超时控制**：多级超时保护，确保单个后端响应慢不会阻塞整个系统
- **防止并发执行**：使用原子标志避免健康检查任务堆积
- **一致的失败策略**：根据配置的失败次数决定节点状态

### 响应能力

即使在部分后端服务不可用或响应缓慢的情况下，负载均衡器依然能够：

- 保持对新请求的处理能力
- 准确记录和处理后端状态变化
- 自动从重试池恢复健康的后端服务
- 彻底移除持续失败的后端服务

## 项目结构

```
go-load-balancer/
├── cmd/
│   └── lb/                     # 主程序入口
│       └── main.go
├── internal/
│   ├── config/                 # 配置处理
│   ├── algorithms/             # 负载均衡算法
│   │   ├── round_robin.go      # 轮询算法
│   │   ├── least_conn.go       # 最少连接
│   │   ├── weighted_rr.go      # 加权轮询
│   │   ├── ip_hash.go          # IP哈希
│   │   └── factory.go          # 算法工厂
│   ├── backend/                # 后端管理
│   │   ├── backend.go          # 后端服务器实现
│   │   ├── pool.go             # 服务器池管理
│   │   ├── health_checker.go   # 健康检查器
│   │   └── status.go           # 状态常量
│   ├── proxy/                  # 代理功能
│   │   ├── reverse_proxy.go    # 反向代理实现
│   │   └── error_handler.go    # 错误处理
│   ├── stats/                  # 统计监控
│   │   ├── collector.go        # 数据收集
│   │   ├── prometheus.go       # Prometheus集成
│   │   └── reporter.go         # 报告接口
│   └── server/                 # 服务器
│       ├── manager.go          # 服务器管理器
│       ├── interface.go        # 服务器接口
│       └── standard_http_server.go # 标准HTTP服务器
├── configs/                    # 配置文件示例
└── scripts/                    # 辅助脚本
```

## 负载均衡算法说明

### 轮询 (Round Robin)
依次将请求分配给每个后端，适用于后端服务器性能相近的场景。

### 最少连接 (Least Connection)
将请求发送到当前连接数最少的后端，适用于请求处理时间差异较大的场景。

### 加权轮询 (Weighted Round Robin)
考虑服务器权重的轮询分配，性能更强的服务器可以设置更高的权重接收更多请求。

### IP哈希 (IP Hash)
根据客户端IP地址的哈希值选择后端，确保同一客户端的请求总是发送到相同的后端，适合需要会话一致性的场景。

## 开发计划

- [x] 基础架构搭建
- [x] HTTP反向代理
- [x] 轮询算法
- [x] 最少连接算法
- [x] 加权轮询算法
- [x] IP哈希算法
- [x] Prometheus监控集成
- [x] 并行健康检查优化
- [x] 防阻塞机制
- [ ] 更完善的日志系统
- [ ] HTTPS支持
- [ ] 会话持久化
- [ ] 速率限制
- [ ] Web管理界面

## 贡献

欢迎提交问题和功能请求。如果您想贡献代码，请先讨论您想要进行的更改。

## 许可证

本项目采用Apache-2.0许可证 - 详见LICENSE文件
