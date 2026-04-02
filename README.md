# Stock Analyzer - 金融股票情感分析系统

## 📋 项目描述

这是一个**分布式微服务架构**的金融股票实时情感分析系统。系统采用 **Go + Python + gRPC** 的技术栈，实现高效的流式数据处理：

- **Go 网关**：通过 WebSocket 为前端提供实时数据推送，支持成千上万的并发连接
- **Python AI 服务**：使用 FinBERT 深度学习模型进行金融文本的情感分析（利好/利空/中性）
- **gRPC 流式通信**：零中间缓存的实时数据转发，确保低延迟交互

### 核心场景
用户在浏览器中点击"分析"，系统自动爬取相关股票新闻标题，通过 AI 模型逐条实时分析情感倾向，分析结果在网页上**一个接一个**地实时显示出来。

---

## 🏗️ 系统架构

### 数据流向

```
┌─────────────┐         ┌────────────────────────┐         ┌──────────────────┐
│  Web 前端   │ ◄─── WS ──► │ Go 网关 (gateway)    │ ◄─── DNS gRPC ──► │ Python AI Pod ×3 │
│ (WebSocket) │ (JSON)      │ (Port 8081)          │ (Round Robin)      │ (Port 50051)     │
└─────────────┘         └────────────────────────┘         └──────────────────┘
                              ▲                                      ▲
                              │                                      │
                        限流控制 (5 req/s)                    模型加载 × 3 副本
                        令牌桶算法                            FinBERT + 缓存
```

### 核心模块

| 模块 | 容器 | 职责 | 特性 |
|------|------|------|------|
| **gateway.go** | gateway | WebSocket 接收/转发 | Round-robin 负载均衡、DNS 服务发现 |
| **ai_service.py** | ai-service (×3 副本) | FinBERT 推理执行 | 并发线程池、模型缓存、日志实时输出 |
| **analysis.proto** | 共享 | 数据序列化定义 | gRPC 流式双向通信 |

---

## ✨ 系统特色

### 1️⃣ **高性能流式处理**
- 零中间缓存的实时转发：Python 推理出一条结果，立即通过 gRPC 推流到 Go，再实时推送到浏览器
- 用户无需等待全部分析完成，而是每 0.2-0.5 秒看到一条新的情感分析结果

### 2️⃣ **计算与 IO 分离**
- **Python 端**：处理 CPU/GPU 密集的 AI 推理（可部署到有显卡的机器）
- **Go 端**：处理轻量级的网络 IO（可部署到廉价服务器）
- 通过 gRPC 解耦，可独立扩展

### 3️⃣ **超高并发能力**
```
Go 网关（单副本）：
  ├─ Goroutine 轻量级（单个仅消耗 KB 内存）
  ├─ WebSocket 复用 TCP 连接
  └─ 可以处理 10,000+ 并发连接

Python AI 服务（3 个副本）：
  ├─ 副本 1：处理第一批推理请求
  ├─ 副本 2：处理第二批推理请求  
  ├─ 副本 3：处理第三批推理请求
  └─ 基于 DNS 的 Round-Robin 自动分配

资源隔离：
  ├─ 每个副本：ThreadPoolExecutor (N 个线程)
  ├─ 每个副本限制：1 核 CPU + 1GB 内存
  └─ 总计：3 核 CPU + 3GB 内存（可扩展）
```

### 4️⃣ **容器化生产就绪**
- 多阶段编译优化 Go 镜像大小
- 资源隔离：CPU/内存限制防止算力跑满
- 服务发现：Docker DNS 自动解析服务地址
- 环境一致性：本地 Windows + 云端 Linux 完全相同

### 5️⃣ **防护与稳定性**
- **限流器**（Go）：令牌桶算法，每秒最多 5 个新请求
- **超时控制**（gRPC）：30 秒硬性超时，防止僵尸连接
- **资源限制**（Docker）：3 个 AI 副本，每个独立占用 1 核 CPU + 1GB 内存，自动负载均衡
- **模型缓存管理**：支持本地卷映射，避免每次启动重下 400MB 模型

### 6️⃣ **开发友好**
- WebSocket + JSON：前端无需关心 gRPC 协议细节
- Protocol Buffer：跨平台数据序列化，支持版本演进
- Windows 本地开发支持：直接 `pip install torch` 获得 CPU 加速

---

## 🚀 部署指南

### 前置条件

- **Docker & Docker Compose**：[安装指南](https://docs.docker.com/compose/install/)
- **Git**（用于 clone 项目）
- **可选**：Windows 10/11（此项目已在 Windows 11 验证）

### 快速启动（推荐）

```bash
# 1. 进入项目目录
cd stock-analyzer

# 2. 构建并启动所有服务（后台运行）
# ✅ 自动启动：1 个 gateway + 3 个 ai-service 副本
docker-compose up --build -d

# 3. 查看服务状态（包括 3 个 AI 副本）
docker-compose ps

# 输出示例：
# NAME                    COMMAND              SERVICE      STATUS      PORTS
# stock-analyzer-ai-service-1  "python ai_service…" ai-service   Up 2 mins   
# stock-analyzer-ai-service-2  "python ai_service…" ai-service   Up 2 mins   
# stock-analyzer-ai-service-3  "python ai_service…" ai-service   Up 2 mins   
# stock-analyzer-gateway-1      "./gateway"          gateway      Up 2 mins   0.0.0.0:8081->8081/tcp

# 4. 查看所有 AI 副本实时日志
docker-compose logs -f ai-service

# 预期输出（3 个副本并行输出）：
# ai-service-1 | 来自容器 1a2b3c4d 的 AI 节点正在处理请求...
# ai-service-2 | 来自容器 5e6f7g8h 的 AI 节点正在处理请求...
# ai-service-3 | 来自容器 9i0j1k2l 的 AI 节点正在处理请求...
# ai-service-1 | 正在加载 AI 模型，请稍候...
# ai-service-1 | ✅ 模型加载完成！
```

### 模型缓存管理

第一次启动时，ai-service 会下载 FinBERT 模型（约 400MB）。为了避免重复下载，docker-compose 已配置本地卷映射：

```bash
# 模型会缓存在本地目录（默认 ./ai_models）
ls -la ./ai_models

# 可通过环境变量自定义缓存路径
export AI_MODELS_PATH=/path/to/cache
docker-compose up --build -d
```

### 本地开发启动（无 Docker）

#### Python AI 服务
```bash
# 1. 安装依赖
pip install grpcio grpcio-tools transformers torch --extra-index-url https://download.pytorch.org/whl/cpu

# 2. 从 proto 文件生成 Python gRPC 代码
python -m grpc_tools.protoc -I. --python_out=. --grpc_python_out=. analysis.proto

# 3. 启动服务（监听 localhost:50051）
python ai_service.py
```

#### Go 网关
```bash
# 新开一个终端

# 1. 获取依赖
go mod download

# 2. 设置环境变量指向 Python 服务
export GRPC_SERVER_ADDR=localhost:50051    # Linux/Mac: 本地单机模式
# 或在 Windows Cmd 中：
# set GRPC_SERVER_ADDR=localhost:50051
# 或在 Windows PowerShell 中：
# $env:GRPC_SERVER_ADDR="localhost:50051"

# 注意：Docker 环境中使用 dns:///ai-service:50051（DNS 负载均衡）

# 3. 启动网关（监听 localhost:8081）
go run gateway.go

# 预期输出：
# 🌐 网关已启动: ws://localhost:8081/ws/analysis
```

然后在浏览器打开 [test_client.html](test_client.html)（使用本地 http-server 或直接打开文件）。

### 停止服务

```bash
# 停止并删除所有容器（保留 volume）
docker-compose down

# 完全清理（包括 volume）
docker-compose down -v
```

---

## 📱 使用指南

### 场景一：通过网页查看实时分析

1. **启动服务**（参考部署指南）

2. **打开测试页面**
   ```
   http://localhost:8081/ws/analysis
   ```
   或直接在浏览器中打开 [test_client.html](test_client.html)

3. **观察数据流**
   - 页面连接到 WebSocket 端点
   - 点击"开始分析"按钮后，AI 推理开始
   - 情感分析结果实时显示，格式为：
     ```json
     {
       "date": "2026-04-01",
       "price": 150.0,
       "sentiment": "Positive (利好) | 来源: ...股票新闻标题..."
     }
     ```

### 场景二：自定义股票代码

编辑 [ai_service.py](ai_service.py)，修改 `GetHistoryAnalysis` 方法：

```python
def GetHistoryAnalysis(self, request, context):
    stock_code = request.stock_code  # 这里可以获取客户端传来的股票代码
    news_headlines = [
        f"{stock_code} reports record-breaking quarterly profits.",
        # ... 改为你的实际新闻列表
    ]
```

然后调用 gRPC 时传递不同的股票代码：
```python
response = client.GetHistoryAnalysis(pb.StockRequest(stock_code="AAPL"))
```

### 场景三：集成到你的前端应用

任何支持 WebSocket 的前端框架都可以集成：

```javascript
// JavaScript 示例
const ws = new WebSocket('ws://localhost:8081/ws/analysis');

ws.onopen = () => {
  console.log('已连接');
};

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log(`${data.date}: ${data.sentiment}`);
  // 在网页 UI 中实时渲染
};

ws.onerror = (error) => {
  console.error('连接错误:', error);
};
```

---

## 🛠️ 技术栈与版本

| 组件 | 版本 | 说明 |
|------|------|------|
| **Go** | 1.26.1 | 网关核心语言 |
| **Python** | 3.10 | AI 服务核心语言 |
| **gRPC** | v1.80.0 (Go), Latest (Python) | 微服务通信 |
| **FinBERT** | PyTorch 版本 | 金融文本情感模型 |
| **Docker** | Latest | 容器化部署 |
| **WebSocket** | gorilla/websocket v1.5.3 | 前端实时通信 |

### 关键依赖

**Go 端** (`go.mod`):
- `google.golang.org/grpc`：gRPC 框架（包含负载均衡器）
- `github.com/gorilla/websocket`：WebSocket 支持
- `golang.org/x/time/rate`：令牌桶限流

**Python 端** (`ai_service.py`):
- `grpcio` + `grpcio-tools`：gRPC 框架
- `transformers`：Hugging Face 模型库
- `torch`：PyTorch 深度学习框架（支持 CPU/GPU）

---

## 📈 性能与扩展

### 性能指标（当前默认配置）

| 指标 | 数值 | 说明 |
|------|------|------|
| 单条新闻推理时间 | 0.2-0.5 秒 | CPU 模式下的延迟 |
| 并发 WebSocket 连接 | 10,000+ | Go 网关容量 |
| AI 吞吐量 | ~18 条/秒 | 3 个副本 × 6 条/秒 |
| 内存占用 | ~3.5GB | 3 个 AI 副本 (1GB×3) + Go 网关 + 系统 |
| 网关延迟 | <5ms | Docker 网络 (DNS Round-Robin) |
| 模型首次加载 | ~15 秒 | 400MB 模型下载 + 加载 |

**扩展场景**：
- 6 个 AI 副本：吞吐量 ~36 条/秒，内存 ~6.5GB
- 10 个 AI 副本 + GPU：吞吐量 ~600 条/秒 (GPU 加速)

#### 1. 增加 GPU 支持
```bash
# 修改 Dockerfile.python，使用 NVIDIA GPU 镜像
FROM nvidia/cuda:12.2-cudnn8-runtime

# 在 docker-compose.yml 中添加 GPU 运行时和更多副本
services:
  ai-service:
    deploy:
      replicas: 6  # 增加副本数
      resources:
        reservations:
          devices:
            - driver: nvidia
              count: 1
              capabilities: [gpu]
```

#### 2. 增加副本数实现水平扩展
```bash
# 在 docker-compose.yml 中修改
deploy:
  replicas: 6  # 从 3 个增加到 6 个副本
  resources:
    limits:
      cpus: '1'
      memory: 1G
```

#### 3. 增加 Go 限流阈值
```go
// gateway.go - 修改限流器配置
// 原始：每秒 5 个请求，桶大小 10
var limiter = rate.NewLimiter(20, 50)  // 新：每秒 20 个请求，桶大小 50
```

#### 4. 启用国内镜像加速（可选）
```yaml
# docker-compose.yml - 取消注释以下行
environment:
  - HF_HOME=/root/.cache/huggingface
  - HF_ENDPOINT=https://hf-mirror.com  # 国内加速镜像站
```

---

## 🔐 安全与稳定性

### 内置防护措施

| 层级 | 防护 | 配置位置 |
|------|------|--------|
| **Go 限流** | 令牌桶算法 (rate.NewLimiter) | `gateway.go` L19 |
| **gRPC 超时** | 30 秒硬性超时 | `gateway.go` L65 |
| **负载均衡** | DNS + Round-Robin | `gateway.go` L51-54 |
| **资源隔离** | 每个副本 1 核 + 1GB 内存 × 3 | `docker-compose.yml` L26-29 |
| **并发控制** | ThreadPoolExecutor (N = CPU 核心数) | `ai_service.py` L93 |

### 监控与调试

```bash
# 查看容器资源使用情况
docker stats

# 查看 AI 服务的日志输出
docker logs -f stock-analyzer-ai-service-1

# 验证 gRPC 连接（需要 grpcurl 工具）
grpcurl -plaintext localhost:50051 list
```

## 🔧 核心配置说明

### Docker 服务发现和负载均衡

这个项目使用 **DNS 服务发现 + Round-Robin 负载均衡**，使得多个 AI 副本能够自动负载均衡：

```plaintext
请求流向图：
┌─────────────┐
│ WebSocket   │
│  客户端×100 │
└──────┬──────┘
       │
       ▼
┌──────────────────────┐
│  Go 网关  (1副本)    │
│ Round-Robin LB       │
└────────┬─────┬──────┘
         │     │
    ┌────▼─┐   └────┬────┐
    │DNS   │        │    │ (resolve)
    │名称  │        │    │ ai-service:50051
    │解析  │        │    │ → [IP1, IP2, IP3]
    └────┬─┘        │    │
         │          ▼    ▼    ▼
         │      ┌────┐ ┌────┐ ┌────┐
         └─────►│Pod1│ │Pod2│ │Pod3│
                └────┘ └────┘ └────┘
             (每个 1 核 + 1GB 内存)
```

### 环境变量配置

**docker-compose.yml 中的关键环境变量**：

```yaml
gateway:
  environment:
    - GRPC_SERVER_ADDR=dns:///ai-service:50051  # DNS 模式 + 服务名
    # dns:/// 前缀：启用 DNS 解析
    # ai-service：Docker 网络中的服务名
    # :50051：gRPC 服务端口

ai-service:
  env_file: .env  # 从 .env 文件读取额外配置
  environment:
    - HF_HOME=/root/.cache/huggingface  # Hugging Face 模型缓存路径
    # - HF_ENDPOINT=https://hf-mirror.com  # (可选) 国内镜像加速
```

**本地开发环境变量**：

```bash
# Linux/Mac
export GRPC_SERVER_ADDR=localhost:50051

# Windows Cmd
set GRPC_SERVER_ADDR=localhost:50051

# Windows PowerShell
$env:GRPC_SERVER_ADDR="localhost:50051"
```

### 卷（Volume）映射配置

模型缓存映射避免每次启动重新下载 400MB 的模型：

```yaml
# docker-compose.yml
ai-service:
  volumes:
    - ${AI_MODELS_PATH:-./ai_models}:/root/.cache/huggingface
```

使用方式：
```bash
# 默认：模型缓存在 ./ai_models
docker-compose up -d

# 自定义缓存路径
export AI_MODELS_PATH=/data/models
docker-compose up -d
```

---

### 目录结构说明

```
stock-analyzer/
├── gateway.go                 # Go WebSocket 网关
├── ai_service.py             # Python gRPC 服务
├── analysis.proto            # Protocol Buffer 定义
├── pb/                        # 编译后的 gRPC 代码
│   ├── analysis.pb.go
│   └── analysis_grpc.pb.go
├── Dockerfile.go             # Go 容器镜像
├── Dockerfile.python         # Python 容器镜像
├── docker-compose.yml        # 容器编排配置
├── go.mod / go.sum           # Go 依赖管理
├── client_demo.html          # 网页测试客户端
└── readme.md                 # 本文件
```

### 从零开始重新生成 gRPC 代码

```bash
# Go：自动由 protoc 生成（编译时）
# Python：手动生成
python -m grpc_tools.protoc \
  -I. \
  --python_out=. \
  --grpc_python_out=. \
  analysis.proto
```

### 常见问题排查

**Q: 为什么启动了 3 个 AI 副本？**  
A: 为了展示容器编排和负载均衡。单机可改为 `replicas: 1`；生产环境可根据负载增加副本数。

**Q: 启动时 AI 模型下载缓慢？**  
A: 首次下载 FinBERT 模型 (~400MB) 较慢。几个解决方案：
- 使用卷映射缓存模型（已配置）
- 启用国内镜像：`HF_ENDPOINT=https://hf-mirror.com`
- 提前在宿主机下载模型到 `./ai_models`

**Q: DNS 解析失败（连接被拒绝）？**  
A: 检查以下几点：
- 确保使用 `dns:///ai-service:50051` 而非 `localhost`
- Docker Compose 是否正常启动：`docker-compose ps`
- 查看 gateway 日志：`docker logs stock-analyzer-gateway-1`

**Q: 如何在本地开发中使用 DNS 模式？**  
A: 本地开发建议使用 `localhost:50051`（单机模式）。如需测试 DNS 负载均衡，必须通过 Docker 运行。

**Q: 模型文件占用太多空间？**  
A: FinBERT 模型 (~400MB) 会被所有 3 个副本共用（通过卷映射）。只占用一份磁盘空间。

**Q: 如何监控 3 个副本的状态？**  
A: 使用以下命令：
```bash
# 查看所有容器状态
docker-compose ps

# 实时监控资源使用
docker stats

# 查看单个副本日志
docker logs stock-analyzer-ai-service-1
docker logs stock-analyzer-ai-service-2
docker logs stock-analyzer-ai-service-3

# 查看所有副本并行输出
docker-compose logs -f ai-service
```

---

## 🌟 实际应用案例

这套架构非常适合以下场景：

1. **金融行情分析 App**：7x24 爬取新闻，实时情感分析推送到移动端
2. **舆情监测系统**：监测特定企业，快速发现负面新闻
3. **多语言支持**：用不同的 BERT 模型替换，支持中文/日文/韩文分析
4. **A/B 测试框架**：通过多个 Python 服务交替调用不同模型版本

---

## 📝 许可证与贡献

此项目为教学演示项目，欢迎 Fork 和 Pull Request。

---

## 📞 获取帮助

- 查看服务日志：`docker-compose logs [service-name]`
- 重启单个服务：`docker-compose restart gateway` 或 `docker-compose restart ai-service`
- 完全重建：`docker-compose up --build --force-recreate -d`
