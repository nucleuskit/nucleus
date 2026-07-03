# Nucleus 使用指南

这份文档说明当前改造后的 Nucleus 应该如何使用。

Nucleus 现在不是脚手架、平台、provider SDK 集合，也不是项目模板生成器。它是给 AI agent、人类 reviewer、CI 共用的一层本地协议工具：把项目事实、可编辑边界、调用链、影响面、技术决策和验证证据变成结构化数据。

## 核心原则

- 用 `adopt` 接入已有 Go 项目，不用 `init` 创建项目。
- 用 `mark` 声明契约、能力锚点和验证命令，不生成业务代码。
- 用 `.nucleus/decisions/*.yaml` 记录 provider/library/driver/ORM 等技术选择。
- 用 `describe/trace/impact/plan` 给 AI 提供定位链、影响链和编辑边界。
- 用 `verify` 产出最终机器可读证据。
- 不用 Nucleus 选择 MySQL ORM、Redis client、Kafka SDK、目录结构或应用 wiring。

## 命令入口

如果已经安装 CLI：

```bash
nucleus --help
```

在本仓库源码内临时运行：

```bash
go run ./cmd/nucleus --help
```

后文统一写成 `nucleus`。如果还没安装，可以把命令替换为：

```bash
go run ./cmd/nucleus
```

## 推荐主流程

```text
adopt -> mark -> describe -> trace/impact -> plan -> decision -> change code -> verify -> report
```

对应命令：

```bash
nucleus adopt --dir . --agent codex --json --pretty
nucleus mark contract http --kind openapi --path api/openapi.yaml --json
nucleus mark capability order_store --kind relational_store --symbol OrderStore --json
nucleus mark verify "go test ./..." --json
nucleus describe --dir . --json --pretty --flow
nucleus trace symbol OrderStore --json --pretty
nucleus impact symbol OrderStore --json --pretty
nucleus plan --dir . --task "add mysql capability" --json --pretty --executable
nucleus decision validate .nucleus/decisions/order-store-mysql.yaml --json --pretty
nucleus verify --dir . --json --pretty
nucleus report --dir . --json --pretty
```

## 1. 接入已有项目

在已有 Go 项目根目录执行：

```bash
nucleus adopt --dir . --agent codex --json --pretty
```

`adopt` 只写：

- `nucleus.yaml`
- `.nucleus/*`

它不会：

- 创建 service/worker/library 模板
- 生成业务目录
- 修改 `go.mod` 或 `go.sum`
- 选择 provider、ORM、driver 或 SDK

如果已有 `nucleus.yaml`，需要显式覆盖：

```bash
nucleus adopt --dir . --force --json --pretty
```

## 2. 声明契约

HTTP OpenAPI：

```bash
nucleus mark contract http --kind openapi --path api/openapi.yaml --json --pretty
```

gRPC proto：

```bash
nucleus mark contract grpc --kind proto --path api/proto/order.proto --json --pretty
```

错误码契约：

```bash
nucleus mark contract errors --kind errors --path api/errors.yaml --json --pretty
```

契约只是声明项目已有或将要维护的协议文件。外部行为变更应先改契约，再改实现。

## 3. 声明能力锚点

能力是语义锚点，不是 provider 实现。

例如项目里已经有 `OrderStore` 接口：

```bash
nucleus mark capability order_store \
  --kind relational_store \
  --symbol OrderStore \
  --intent "order persistence boundary" \
  --json --pretty
```

如果短 symbol 名不唯一，命令会返回候选项；应改用稳定 symbol ID。

能力声明可以表达：

- 这个项目有一个 `order_store` 能力
- 它是 `relational_store` 这类语义能力
- 它锚定到项目自己的接口或类型

能力声明不能表达：

- 使用 GORM 还是 Xorm
- 使用 MySQL 还是 PostgreSQL
- DSN 怎么配置
- 事务实现怎么写
- 要不要引入某个 SDK

这些属于技术决策，放到 `.nucleus/decisions`。

## 4. 声明验证命令

`verify` 只运行 `nucleus.yaml` 里声明的项目命令，不隐式运行 `go test ./...`。

添加验证命令：

```bash
nucleus mark verify "go test ./..." --json --pretty
nucleus mark verify "go test ./internal/order/..." --json --pretty
```

这会写入 `nucleus.yaml` 的 `verify.commands`。

## 5. 让 AI 读取项目事实

修改前先跑：

```bash
nucleus describe --dir . --json --pretty --flow
```

重点读取：

- `result_kind/schema_version/schema_ref/ok/diagnostics`
- `service`
- `endpoints`
- `grpc_services`
- `error_codes`
- `capabilities`
- `edit_surfaces`
- `symbol_graph`
- `flow_graph`
- `verification`
- `generated_freshness`

AI 修改代码前必须看 `edit_surfaces`：

- `allowed`: 可以改
- `readonly`: 只能读
- `forbidden`: 不能改

如果目标文件不在 allowed 范围内，不要绕过去写业务代码，应先调整协议边界或让用户确认。

## 6. 查询调用链和影响面

查 symbol 调用链：

```bash
nucleus trace symbol OrderStore --json --pretty
```

查能力锚点：

```bash
nucleus trace capability order_store --json --pretty
```

查路由链路：

```bash
nucleus trace route "POST /orders" --json --pretty
```

查 symbol 影响面：

```bash
nucleus impact symbol OrderStore --json --pretty
```

查文件影响面：

```bash
nucleus impact file internal/order/store.go --json --pretty
```

查契约影响面：

```bash
nucleus impact contract api/openapi.yaml --json --pretty
```

这些命令用于帮助 AI 快速定位：

- 谁调用了这个 symbol
- 这个改动影响哪些文件
- 哪些测试可能相关
- 哪些路由或契约可能受影响
- 哪些边是确定事实，哪些只是启发式推断

短 symbol 名有歧义时，不要猜，使用输出里的稳定 ID。

## 7. 生成计划

```bash
nucleus plan --dir . --task "add mysql capability" --json --pretty --executable
```

重点读取：

- `ok`
- `diagnostics`
- `suggested_edits`
- `blocked_edits`
- `blocked_decisions`
- `impact_summary`
- `commands`
- `risks`
- `context.recipe_candidates`

规则：

- `blocked_edits` 不为空时，停止修改。
- `blocked_decisions` 不为空时，先补或 supersede decision。
- `recipe_candidates` 只是参考知识，不是默认技术选型。
- `commands` 是建议链路，不代表 Nucleus 会自动执行所有命令。

## 8. 记录技术决策

当引入或替换 provider/library/driver/ORM/SDK 时，写 decision 文件。

示例：`.nucleus/decisions/order-store-mysql.yaml`

```yaml
schema_version: "decision.v1"
id: order-store-mysql
capability: order_store
decision:
  provider: mysql
  library: gorm.io/gorm
  driver: github.com/go-sql-driver/mysql
  status: proposed
  locked: false
reason:
  - project team selected MySQL for order persistence
  - existing service conventions use GORM repositories
impact:
  symbols:
    - go://example.com/order/internal/order#OrderStore
  files:
    - internal/order/store.go
    - internal/order/mysql_store.go
verification:
  commands:
    - go test ./internal/order/...
alternatives:
  - provider: mysql
    library: xorm.io/xorm
    driver: github.com/go-sql-driver/mysql
    reason: rejected because project already uses GORM transaction helpers
```

校验：

```bash
nucleus decision validate .nucleus/decisions/order-store-mysql.yaml --json --pretty
```

接受并锁定：

```bash
nucleus decision accept .nucleus/decisions/order-store-mysql.yaml --by human --json --pretty
```

锁定后，后续 AI 不能静默替换 provider/library/driver。

如果确实要替换，创建新的 supersede decision，然后执行：

```bash
nucleus decision supersede .nucleus/decisions/order-store-mysql-v2.yaml --json --pretty
nucleus decision validate .nucleus/decisions/order-store-mysql-v2.yaml --json --pretty
```

## 9. 生成契约产物

契约变更后运行：

```bash
nucleus gen --dir . --json --pretty
```

`gen` 只生成契约派生产物。它不生成 provider glue、业务 wiring、目录模板或依赖变更。

## 10. 验证

快速检查：

```bash
nucleus validate --dir . --json --pretty
nucleus lint --dir . --strict --json --pretty
```

最终证据：

```bash
nucleus verify --dir . --json --pretty
```

`verify` 会输出 `evidence.v1`，重点看：

- `ok`
- `summary`
- `steps`
- `diagnostics`
- 每个 step 的 `kind/status/ok/exit_code`

如果根目录没有 `nucleus.yaml`，`verify` 会失败并给出 `manifest.read_failed`。这是正确行为；先运行 `adopt`。

## 11. MCP 给 agent 使用

启动本地 stdio MCP：

```bash
nucleus --dir . mcp --stdio
```

MCP 是只读结构化工具层。它不会写文件、不会执行验证命令、不会访问网络、不会选择 provider。

常用工具：

- `get_service_description`
- `get_edit_surfaces`
- `get_contracts`
- `get_capabilities`
- `trace_symbol`
- `trace_route`
- `trace_capability`
- `impact_symbol`
- `impact_file`
- `impact_contract`
- `find_symbol`
- `list_callers`
- `list_callees`
- `validate_decision`
- `list_decisions`
- `get_report`
- `build_plan`
- `list_recipes`
- `get_recipe`

所有 MCP `structuredContent` 都应带：

- `result_kind`
- `schema_version`
- `schema_ref`
- `ok`
- `diagnostics`

agent 不要解析自然语言文本来判断成功失败，应以这些字段为准。

## 12. 常见场景

### 接入一个已有服务

```bash
nucleus adopt --dir . --agent codex --json --pretty
nucleus mark verify "go test ./..." --json --pretty
nucleus describe --dir . --json --pretty
```

### 给已有项目增加 MySQL 能力

```bash
nucleus mark capability order_store --kind relational_store --symbol OrderStore --json --pretty
nucleus plan --dir . --task "add mysql capability for order_store" --json --pretty --executable
```

然后由用户或 AI 写 `.nucleus/decisions/order-store-mysql.yaml`，明确 MySQL、ORM、driver、理由、影响面和验证命令。

Nucleus 不会替你决定用 GORM 还是 Xorm。

### 修改一个 API

```bash
nucleus describe --dir . --json --pretty --flow
nucleus impact contract api/openapi.yaml --json --pretty
nucleus plan --dir . --task "change GET /orders response" --json --pretty --executable
```

先改 `api/openapi.yaml`，再运行：

```bash
nucleus gen --dir . --json --pretty
nucleus verify --dir . --json --pretty
```

### 替换已锁定 provider

```bash
nucleus plan --dir . --task "switch order_store provider to xorm" --json --pretty --executable
```

如果出现 `blocked_decisions`，创建 supersede decision：

```bash
nucleus decision supersede .nucleus/decisions/order-store-xorm.yaml --json --pretty
nucleus decision validate .nucleus/decisions/order-store-xorm.yaml --json --pretty
```

再重新 plan。

## 13. 不要使用的旧路径

这些已经不是当前方向：

- `nucleus init`
- `nucleus capability add`
- `nucleus capability add --provider ...`
- `nucleus migrate`
- `report --platform`
- `bridge` provider adapter 集合
- 自动修改 `go.mod/go.sum` 的能力接入
- 把 provider/library/driver 写进 `nucleus.yaml`

如果一个需求必须选择技术栈，把选择写成 decision evidence，而不是写成框架默认值。

## 14. 推荐给 AI 的最小工作守则

1. 先跑 `describe`，读取 edit surfaces、symbol graph、verification。
2. 对目标 symbol/route/file 跑 `trace` 或 `impact`。
3. 跑 `plan --executable`，遇到 blocked 就停止。
4. provider/library/driver 变化必须写 decision。
5. 只改 allowed edit surfaces 内的文件。
6. 变更后跑 `verify --json`。
7. 最终交付时引用 evidence，而不是说“我觉得可以”。
