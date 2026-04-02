package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"stock-analyzer/pb" // 确保这里与你的 go.mod 一致
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	// 必须匿名导入轮询驱动
)

// 每秒允许 5 个新请求，桶里最多存 10 个令牌
var limiter = rate.NewLimiter(5, 10)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true }, // 允许跨域测试
}

func handleWS(w http.ResponseWriter, r *http.Request) {
	// 尝试获取令牌，如果拿不到说明流量过大
	if !limiter.Allow() {
		http.Error(w, "服务器太忙了，请稍后再试 (Too Many Requests)", http.StatusTooManyRequests)
		return
	}
	// 1. 升级 HTTP 连接为 WebSocket
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WS 升级失败:", err)
		return
	}
	defer ws.Close()

	// 在 gateway.go 中增加从环境变量读取地址的逻辑
	addr := os.Getenv("GRPC_SERVER_ADDR")
	if addr == "" {
		addr = "localhost:50051" // 本地开发默认值
	}

	// 2. 连接 Python gRPC 服务端
	// 设置负载均衡策略
	// 在 Docker 网络中，ai-service 会对应多个容器 IP，dns 模式能自动发现它们
	conn, err := grpc.Dial(
		addr,
		grpc.WithDefaultServiceConfig(`{"loadBalancingConfig": [{"round_robin":{}}]}`),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	if err != nil {
		log.Println("gRPC 连接失败:", err)
		return
	}
	defer conn.Close()
	client := pb.NewStockAnalyzerClient(conn)

	// 给 gRPC 调用增加一个 30 秒的硬性总超时
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	// 如果 WebSocket 断开了，ctx 也会被取消
	// Python 端的 yield 循环会自动收到这个信号并停止
	// 3. 发起 gRPC 流式请求
	stream, err := client.GetHistoryAnalysis(ctx, &pb.StockRequest{StockCode: "GOOGL"})
	if err != nil {
		log.Println("无法开启 gRPC 流:", err)
		return
	}

	// 4. “泵”数据：将 gRPC 流实时转发给 WebSocket
	for {
		res, err := stream.Recv()
		if err != nil {
			break // 流结束或出错
		}
		// 将数据转为 JSON 发送给网页
		ws.WriteJSON(res)
	}
	log.Println("✅ 转发完成")
}

func main() {
	http.HandleFunc("/ws/analysis", handleWS)
	log.Println("🌐 网关已启动: ws://localhost:8081/ws/analysis")
	http.ListenAndServe(":8081", nil)
}
