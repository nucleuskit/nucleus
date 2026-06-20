# Nucleus 本地平台栈

该目录提供本地开发用 compose 示例，用于验证 Nucleus 能力接入协议和平台字段映射，不是生产部署模板。

## 启动

```bash
docker compose -f deploy/docker/docker-compose.yml up -d
```

或使用仓库 Make 目标：

```bash
make docker-up
make docker-down
```

## 服务

| 服务 | 地址 | 用途 |
|------|------|------|
| Nacos | http://localhost:8848/nacos | 配置和注册示例 |
| Redis | localhost:6379 | 缓存、锁能力示例 |
| Kafka | localhost:9092 | MQ 能力示例 |
| OTel Collector | localhost:4317 / localhost:4318 | OTLP gRPC/HTTP |
| Jaeger | http://localhost:16686 | 本地 trace 查询 |
| Prometheus | http://localhost:9090 | 指标查询 |

## 配置约束

- 不包含生产密钥，Nacos auth 在本地示例中关闭。
- Redis 未设置密码，仅绑定本机开发端口。
- Kafka 使用单节点 KRaft 模式，仅适合本地验证。
- 生产环境应由平台注入 Secret、采样策略、持久化和网络策略。

