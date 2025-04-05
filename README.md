# Go-load-balancer

基于Go实现的轻量级负载均衡器

## 功能特性

- 支持多种负载均衡算法：轮询(Round Robin)、最少连接(Least Connections)、加权轮询(Weighted RR)、IP哈希(IP Hash)
- 健康检查机制，自动剔除故障节点
- 支持HTTP反向代理
- 可配置的监听地址和端口
- YAML格式配置文件

## 快速开始

### 安装

```bash
git clone https://github.com/xcc-d/Go-load-balancer.git
cd Go-load-balancer
go build -o lb cmd/lb/main.go
```

### 详细配置说明

#### 基础配置项

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
    
# 健康检查配置
health_check:
  interval: "10s"  # 检查间隔(如: 10s, 1m)
  timeout: "5s"    # 检查超时时间
  path: "/health"  # 健康检查端点
```

#### 高级配置示例

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
  
# 日志配置(可选)
log:
  level: "info"  # 日志级别(debug/info/warn/error)
  path: "/var/log/lb.log"
```

#### 注意事项
1. 时间单位支持: ns(纳秒), us(微秒), ms(毫秒), s(秒), m(分钟), h(小时)
2. URL必须包含协议(http://或https://)
3. 权重仅在weighted_rr算法下生效
4. 修改配置后需重启服务生效

### 运行

```bash
./lb -config configs/example.yaml
```

### 测试

```bash
curl http://localhost:8080
```

## 项目结构

```
go-load-balancer/
├── cmd/
│   └── lb/                     # 主程序入口
│       └── main.go
├── internal/
│   ├── config/                 # 配置处理
│   ├── algorithms/             # 负载均衡算法
│   ├── backend/                # 后端管理
│   ├── proxy/                  # 代理功能
│   └── server/                 # 服务器
├── configs/                    # 配置文件示例
└── scripts/                    # 辅助脚本
```

## 开发计划

- [x] 基础架构搭建
- [x] HTTP反向代理
- [x] 轮询算法
- [ ] 最少连接算法
- [ ] 加权轮询算法
- [ ] IP哈希算法
- [ ] Prometheus监控集成
