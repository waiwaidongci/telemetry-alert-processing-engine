# Telemetry Alert Engine

纯 Go 后端设备遥测接入与实时告警规则引擎。项目只提供 HTTP API 和后台任务，不包含网页前端。

## 项目说明

系统接收设备通过接入令牌上报的遥测数据，按租户和指标保存原始数据及 1 分钟、5 分钟、1 小时、1 天聚合数据。用户可以为设备或设备标签配置告警规则，引擎在每次写入后增量评估，并由定时扫描兜底。告警进入触发、持续、恢复阶段时生成事件，通过 webhook 或日志通知，通知失败会重试并记录，不影响主流程。

服务治理能力包括 YAML 与环境变量配置、结构化日志、请求日志、panic 恢复、请求超时、优雅退出、`/healthz`、`/readyz` 和 `/metrics`。

## 架构

代码遵循 `cmd`、`internal`、`api` 分层，并按照 `domain`、`application`、`adapter`、`infrastructure` 拆分。

- `domain` 定义设备、租户、遥测、告警规则、告警事件、通知、保留策略等领域模型。
- `application` 包含业务服务和端口接口，仓储、时钟、ID 生成、通知发送等依赖全部通过构造函数注入。
- `adapter` 实现指标窗口读取、通知发送、令牌桶限流和时钟等基础设施适配器。
- `infrastructure` 包含 SQLite 数据库、表结构和仓储实现。
- `api` 包含 HTTP handler、路由和中间件。

数据库默认使用 `modernc.org/sqlite`，因此本地无需安装 PostgreSQL 或启动 Docker 即可运行。应用启动时自动创建表；PostgreSQL 生产迁移文件保留在 `migrations/postgres`。

## 目录结构

```text
.
├── api/                    # HTTP 路由、响应、中间件和 handler
├── cmd/telemetry-alert/   # 可执行入口
├── configs/                # YAML 配置
├── deploy/                 # Dockerfile 与 PostgreSQL compose 参考配置
├── internal/
│   ├── adapter/            # 指标、通知、限流、时钟适配器
│   ├── application/        # 服务与端口接口
│   ├── config/             # 配置加载
│   ├── domain/             # 领域模型
│   ├── infrastructure/     # SQLite 与仓储实现
│   ├── logging/            # 结构化日志上下文
│   └── system/             # 时钟、ID、令牌哈希
├── migrations/postgres/    # PostgreSQL 生产迁移
├── scripts/                # 本地开发验证脚本
├── go.mod
└── Makefile
```

## 启动步骤

```bash
cd telemetry-alert-engine
go mod tidy
go build ./...
./scripts/run-dev.sh
```

`run-dev.sh` 会构建二进制，使用默认配置启动服务，等待 `/healthz` 和 `/readyz`，创建租户、设备、告警规则，上报单条和批量遥测，查询原始数据、聚合数据和告警事件，最后停止服务。默认验证端口为 `18080`，可以通过 `TELEMETRY_DEV_PORT` 修改。

手动启动服务：

```bash
go run ./cmd/telemetry-alert -config configs/config.yaml
```

## API 示例

### 健康检查

```bash
curl http://127.0.0.1:8080/healthz
curl http://127.0.0.1:8080/readyz
curl http://127.0.0.1:8080/metrics
```

### 创建租户

```bash
curl -X POST http://127.0.0.1:8080/v1/tenants \
  -H 'Content-Type: application/json' \
  -d '{"name":"factory-a"}'
```

### 创建设备

返回的 `token` 只在创建时返回，普通查询不会暴露令牌。

```bash
curl -X POST http://127.0.0.1:8080/v1/tenants/{tenant_id}/devices \
  -H 'Content-Type: application/json' \
  -d '{"name":"pump-01","type":"pump","serial_number":"PUMP-0001","location":"plant-a","tags":{"area":"east"}}'
```

### 创建告警规则

```bash
curl -X POST http://127.0.0.1:8080/v1/tenants/{tenant_id}/rules \
  -H 'Content-Type: application/json' \
  -d '{"name":"high-temp","device_id":"{device_id}","metric_name":"temperature","operator":"gt","threshold":80,"window":"5m","duration":"0s","level":"warning","cooldown":"1m","channels":["log","webhook"]}'
```

### 上报单条遥测

```bash
curl -X POST http://127.0.0.1:8080/v1/ingest \
  -H "Authorization: Bearer {device_token}" \
  -H 'Content-Type: application/json' \
  -d '{"metric_name":"temperature","value":85.5,"timestamp":"2026-08-18T01:00:00Z","idempotency_key":"sample-1"}'
```

### 批量上报

```bash
curl -X POST http://127.0.0.1:8080/v1/ingest/batch \
  -H "Authorization: Bearer {device_token}" \
  -H 'Content-Type: application/json' \
  -d '{"points":[{"metric_name":"temperature","value":86,"timestamp":"2026-08-18T01:00:05Z"}]}'
```

gzip 请求体通过 `Content-Encoding: gzip` 支持。

### 查询遥测

```bash
curl 'http://127.0.0.1:8080/v1/tenants/{tenant_id}/telemetry/raw?device_id={device_id}&metric_name=temperature&start=2026-08-18T00:00:00Z&end=2026-08-18T02:00:00Z&limit=100'

curl 'http://127.0.0.1:8080/v1/tenants/{tenant_id}/telemetry/aggregates?device_id={device_id}&metric_name=temperature&granularity=5m&start=2026-08-18T00:00:00Z&end=2026-08-18T02:00:00Z'
```

### 查询告警事件

```bash
curl 'http://127.0.0.1:8080/v1/tenants/{tenant_id}/events?rule_id={rule_id}'
```

## 配置项

默认配置在 `configs/config.yaml`。环境变量会覆盖同名字段，例如：

- `SERVER_ADDR`
- `DATABASE_SQLITE_PATH`
- `DATABASE_DRIVER`
- `LOG_LEVEL`
- `TELEMETRY_MAX_BATCH_SIZE`
- `TELEMETRY_INGEST_RATE_LIMIT`
- `ALERT_EVALUATION_INTERVAL_SECONDS`
- `RETENTION_RUN_INTERVAL_SECONDS`
- `NOTIFICATION_WEBHOOK_URL`

SQLite 与 PostgreSQL 的差异：

- 当前二进制默认只支持 `sqlite`，用于零外部依赖的本地开发验证。
- `migrations/postgres/0001_init.sql` 和 `deploy/docker-compose.yml` 保留 PostgreSQL 生产结构及启动参考。
- 生产环境如需切换 PostgreSQL，需要在仓储层使用 Postgres 方言并运行迁移；本项目代码当前将 SQLite 作为默认驱动。

## 迁移

SQLite 表结构在应用启动时由 `internal/infrastructure/sqlite.Migrate` 自动创建。PostgreSQL 迁移文件位于 `migrations/postgres/0001_init.sql`。

## 验证结果

已在本机执行：

```text
gofmt -w $(find cmd internal api -name '*.go' -type f)
go mod tidy
go vet ./...
go build ./...
go test ./...
```

`go vet ./...`、`go build ./...`、`go test ./...` 均通过；测试命令显示所有包 `[no test files]`。

`scripts/run-dev.sh` 验证输出摘要：

```text
== /healthz ==
{"status":"ok"}

== /readyz ==
{"status":"ready"}

== create tenant ==
{"id":"...","name":"dev-tenant",...}

== create device ==
{"device":{...},"token":"..."}

== create alert rule ==
{"id":"...","name":"high-temp",...}

== ingest single point ==
{"id":"...","value":85.5,...}

== ingest batch ==
{"inserted":2,...}

== query raw telemetry ==
{"items":[...],"meta":{"count":3}}

== query aggregates ==
{"items":[{"granularity":"5m","count":3,"sum":255.5,"min":84,"max":86}],...}

== query alert events ==
{"items":[{"event_type":"triggered","level":"warning",...}],...}

== stopping service ==
service stopped
```

非测试 Go 源码统计：

```text
find cmd internal api -name '*.go' -type f ! -name '*_test.go' -print0 | xargs -0 wc -l | tail -1
4870 total
```
