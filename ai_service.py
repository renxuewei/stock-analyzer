import grpc
import time
import torch
from concurrent import futures
from transformers import AutoTokenizer, AutoModelForSequenceClassification
import analysis_pb2
import analysis_pb2_grpc
import os

import socket
print(f"🐍 来自容器 {socket.gethostname()} 的 AI 节点正在处理请求...", flush=True)

import os
if not os.environ.get('HF_HOME'):
    os.environ['HF_HOME'] = os.path.join(
        os.path.expanduser('~'),
        '.cache',
        'huggingface',
        'hub'
    )

# 1. 加载金融情感分析模型 (FinBERT)
# 第一次运行会下载模型（约 400MB），请保持网络畅通
MODEL_NAME = "ProsusAI/finbert"
print("正在加载 AI 模型，请稍候...", flush=True)
tokenizer = AutoTokenizer.from_pretrained(MODEL_NAME)
model = AutoModelForSequenceClassification.from_pretrained(
    MODEL_NAME,
    trust_remote_code=True,
    torch_dtype=torch.float32
)
model.eval()  # 设置为评估模式
print("✅ 模型加载完成！", flush=True)


class StockAnalyzerServicer(analysis_pb2_grpc.StockAnalyzerServicer):
    def GetHistoryAnalysis(self, request, context):
        print(f"🐍 正在为 {request.stock_code} 调用深度学习模型...", flush=True)

        # 模拟 5 条真实的金融新闻标题进行分析
        news_headlines = [
            f"{request.stock_code} reports record-breaking quarterly profits.",
            f"Analysts downgrade {request.stock_code} due to supply chain issues.",
            f"New regulatory filing shows massive insider buying in {request.stock_code}.",
            f"{request.stock_code} faces potential lawsuit over patent infringement.",
            f"Market sentiment remains neutral for {request.stock_code} this week.",
        ]

        for i, text in enumerate(news_headlines):
            # 2. 执行 AI 推理
            inputs = tokenizer(text, return_tensors="pt")
            with torch.no_grad():
                outputs = model(**inputs)

            # 获取概率最高的情绪标签
            scores = torch.nn.functional.softmax(outputs.logits, dim=-1)
            label_idx = torch.argmax(scores).item()
            labels = ["Positive (利好)", "Negative (利空)", "Neutral (中性)"]

            sentiment = labels[label_idx]
            confidence = scores[0][label_idx].item()

            # 3. 通过 gRPC 流推送
            yield analysis_pb2.AnalysisResult(
                date=f"2026-04-{i+1:02d}",
                price=150.0 + i,
                sentiment=f"{sentiment} | 来源: {text[:20]}...",
            )
            print(f"已推送第 {i+1} 条 AI 分析", flush=True)


# 核心：根据你的硬件能力设置 max_workers
# 如果是 CPU 推理，建议设为 CPU 核心数；如果是 GPU，通常设为 1-2
executor = futures.ThreadPoolExecutor(max_workers=os.cpu_count() | 1)


def serve():
    server = grpc.server(executor)
    analysis_pb2_grpc.add_StockAnalyzerServicer_to_server(
        StockAnalyzerServicer(), server
    )
    server.add_insecure_port("[::]:50051")
    server.start()
    server.wait_for_termination()


if __name__ == "__main__":
    serve()
