# Nucleus Agent-Native Redesign

Implementation tracker: [2026-07-03-agent-native-implementation-sync.md](2026-07-03-agent-native-implementation-sync.md).

## 背景

这份文档记录一次核心方向调整：Nucleus 不应该继续向“微服务平台”或“项目脚手架”演进，而应该成为一套可加入任意 Go 项目的 AI 原生微服务协议层。

目标用户不是传统意义上的人类框架使用者，而是 AI agent、CI、代码审查者和少量需要审定 AI 决策的人类维护者。Nucleus 的核心价值不是替项目选择技术栈，也不是生成完整工程目录，而是给 AI 提供结构化事实、编辑边界、调用链、影响面和验证证据。

当前项目已经有正确的雏形：

- `describe -> plan -> lint -> verify` 的 AI-safe loop。
- `nucleus.yaml` 中的 manifest-first 和 edit surfaces。
- contract-first 的 OpenAPI、proto、errors。
- `flow_graph`、`capability_graph`、`generated_freshness` 等机器可读上下文。
- `apply`、`execute`、`repair` 的证据化执行思路。

但当前实现也存在明显偏移：

- `init --template service` 生成完整项目骨架，容易把 Nucleus 变成脚手架。
- capability catalog 和 provider hint 中写死大量 provider、库和默认选择。
- `sql/postgres` 等特例会自动引入驱动、修改 `go.mod/go.sum`、生成 repository 和 migration。
- 文档中出现 platform readiness、docker platform local 等表述，容易把项目拉向平台化。
- `describe` 已有图信息，但还不足以支持 AI 快速定位业务逻辑影响链。

由于项目尚未正式发布和生产使用，本次改造不设计降级路径、兼容层、legacy 命令或迁移保留。凡是与新定位冲突的能力，直接删除或重写。

## 一句话定位

Nucleus 是面向 AI agent 的 Go 微服务结构化协议层。

它不接管项目结构，不规定业务分层，不内置 provider 实现，不替用户选择 ORM、MQ、配置中心、日志库或监控库。

它只负责：

- 读取任意 Go 项目的结构化事实。
- 标记 contract、capability、symbol、edit surface、verification command。
- 构建调用链、依赖链、影响面。
- 约束 AI 只能在明确边界内修改。
- 要求 AI 对技术选型留下结构化决策和验证证据。

## 设计原则

### 1. 框架只固化协议，不固化技术选型

允许硬编码：

- manifest schema。
- evidence schema。
- edit surface 判定规则。
- generated freshness 规则。
- contract-first 工作流。
- 文件越界、symlink、secret 泄漏等安全约束。
- Go AST/import/call graph 等语言级解析规则。
- CLI JSON envelope 的稳定字段。

不允许硬编码：

- 默认数据库 provider。
- 默认 ORM。
- 默认日志库。
- 默认注册发现组件。
- 默认监控组件。
- 默认目录结构。
- 默认 repository/usecase/domain 分层。
- 自动引入某个第三方 SDK。
- 自动生成业务 repository、migration、Dockerfile、Makefile。

### 2. 任意 Go 项目可加入

Nucleus 的主路径应该是 adoption，而不是 initialization。

正确心智：

```text
已有 Go 代码 + nucleus.yaml + graph/evidence = AI 可安全维护的服务
```

错误心智：

```text
nucleus init --template service = 创建一个符合 Nucleus 目录约束的新项目
```

文档中应避免把用户项目描述成待改造或待迁移对象。Nucleus 不拥有用户项目结构，只为任意 Go 代码增加 AI 可读的协议索引。

### 3. 最小颗粒度是结构化锚点，不是项目模板

Nucleus 的最小操作单元应是：

- 一个 contract。
- 一个 capability。
- 一个 interface。
- 一个 symbol。
- 一条 call edge。
- 一个 route。
- 一个 error code。
- 一个 edit surface。
- 一个 verification command。
- 一份 evidence。

不是：

- 一个完整 service template。
- 一套 `internal/app`、`internal/domain`、`internal/usecase` 目录。
- 一个固定 repository 模式。
- 一个固定 infra provider。

### 4. AI 负责判断，框架负责约束和记录

AI 可以判断：

- 当前项目适合用哪个库。
- 是否新增依赖。
- 某个 capability 应落在哪个已有 interface 或 package。
- 哪些测试需要更新。
- 某个 provider 决策的理由是什么。

Nucleus 必须要求 AI 输出：

- 结构化 decision。
- 影响面。
- 涉及 symbols。
- 涉及 files。
- required verification。
- risks。
- evidence。

AI 不应该被 Nucleus 里的 Go 代码用 `providerHint()` 之类逻辑提前替它决定。

Capability kind 也不应该强枚举。Nucleus 可以提供建议词表帮助 AI 归类，但校验只能约束字段结构，不能因为未知 kind 失败。

### 5. Graph 是核心能力

Nucleus 应参考 GitNexus 类项目的知识图思想，但做成 Go 微服务语义图，而不是通用代码浏览器。

AI 改代码前需要知道：

- 当前 route 进入哪个 handler。
- handler 调用哪些 usecase/function。
- usecase 依赖哪些 interface。
- interface 有哪些实现。
- 这个 symbol 被哪些地方调用。
- 改一个 error code 会影响哪些 response mapper/test。
- 改一个 capability 会影响哪些 config、wiring、handler、tests。

### 6. 不保留错误抽象

本项目尚未正式使用，因此不应该为错误方向付出兼容成本。

直接删除：

- `init --template` 项目生成主流程。
- `capability add --provider` provider scaffold 语义。
- provider 默认值和 provider keyword hint。
- `sql/postgres` 深度实现特例。
- 自动写入第三方依赖的 capability 行为。
- platform readiness、platform mapping、platform upload 相关心智。
- manifest v1 字符串 capability 主模型。
- manifest 内 provider 决策字段。

重写后只保留新协议需要的命令、schema 和检查。

## 非目标

Nucleus 不做：

- 微服务控制台。
- 发布平台。
- 中间件一站式平台。
- provider SDK 集合。
- ORM 抽象大一统。
- 项目脚手架主入口。
- 强制目录规范。
- 自动改造用户项目结构。
- 自动选择第三方库。
- 自动生成业务实现。
- 为已判定错误的模板行为保留入口。
- 为未发布功能设置历史包袱。

## 当前问题清单

### 1. `init --template` 过重

现状：

- `cmd/nucleus/internal/initcmd/templates.go` 会生成 service、worker、library 模板。
- service 模板包含 `cmd`、`internal/app`、`internal/config`、`internal/domain`、`internal/usecase`、`internal/adapter`、`internal/component`、`internal/server`、`deploy`、`Makefile` 等。

问题：

- 对任意已有 Go 项目不友好。
- 暗示 Nucleus 要规定项目结构。
- AI 会被模板结构诱导，倾向于新增目录而不是理解现有代码。

方向：

- 删除 `init --template` 代码路径和相关模板。
- 删除以模板为主路径的 examples 文档。
- 新增 `nucleus adopt` 作为主路径。
- `adopt` 只生成最小 manifest 和可选 agent instruction，不生成业务代码。

### 2. capability catalog 硬编码过多

现状：

- `cmd/nucleus/internal/capcatalog/catalog.go` 写死能力、provider、默认 provider。
- `providerHint()` 根据自然语言关键词推断 provider。
- `CapabilityModule()` 写死 capability 到 module import path 的映射。

问题：

- 框架替 AI 和用户做技术选型。
- 新 provider 必须改 Nucleus 源码。
- 对私有库、公司内部库、已有封装不友好。
- 误导 AI 认为 catalog 中的 provider 都是框架内置能力。

方向：

- capability kind 只能作为外置建议词表，不应由 Go 代码强枚举。
- provider 只能出现在 decision 中，不能出现在 manifest 中，也不应由 Go 代码枚举。
- provider recipe 外置为数据文件，可本地扩展。
- 未知 provider 必须允许存在，只要有 interface、verification 和 evidence。
- 删除 `providerHint()` 这类替 AI 预设技术选型的逻辑。

### 3. `sql/postgres` 特例违反边界

现状：

- `capability add sql --provider postgres` 会修改 `go.mod/go.sum`。
- 自动引入 `github.com/lib/pq`。
- 生成 `internal/component/sql/postgres.go`。
- 生成 `internal/adapter/store/postgres/repository.go`。
- 生成 `deploy/migrations/...sql`。

问题：

- Nucleus 直接替项目选择数据库 driver。
- 自动创建 repository 和 migration 属于业务/基础设施实现。
- 这与“框架只使用抽象接口，具体实现由用户/AI 决定”冲突。

方向：

- 删除所有 provider 特例。
- 删除 `capability add --provider` 语义。
- 新 capability 命令只声明能力和接口锚点。
- 具体 provider implementation 由 AI 在项目上下文中实现，并输出 decision evidence。

### 4. platform readiness 心智容易跑偏

现状：

- `report --platform`、`docs/platform-mapping.md`、`examples/deploy/docker` 等内容存在平台化表达。

问题：

- 容易让项目变成平台或控制面。
- 对任意项目加入不必要的发布平台概念。

方向：

- 删除 `report --platform`。
- 删除 platform mapping、platform upload、release dry-run 等平台化字段。
- 如需报告，只保留本地结构质量、AI 决策质量和验证 evidence。

### 5. graph 还不够强

现状：

- `contract/inspect` 有 flow graph、imports、source functions、source handlers。
- 但 plan 输出主要还是 suggested paths、blocked paths、commands、risks。

问题：

- AI 仍需要大量 grep/read 才能理解逻辑链。
- 不能直接回答“改这个 symbol 影响谁”。

方向：

- 增强 symbol graph、call graph、interface implementation graph、test relation graph。
- `plan` 输出必须包含 affected symbols、callers、callees、contracts、tests、verification。

## 目标架构

```text
+-----------------------------+
| AI Agent / Human Reviewer   |
+--------------+--------------+
               |
               v
+-----------------------------+
| Nucleus CLI / MCP Interface |
+--------------+--------------+
               |
               v
+-----------------------------+
| Protocol Layer              |
| - manifest schema           |
| - evidence schema           |
| - decision schema           |
| - edit surface policy       |
+--------------+--------------+
               |
               v
+-----------------------------+
| Inspection Layer            |
| - Go AST                    |
| - import graph              |
| - symbol graph              |
| - call graph                |
| - interface impl graph      |
| - contract graph            |
| - capability graph          |
+--------------+--------------+
               |
               v
+-----------------------------+
| User Project                |
| 任意 Go 目录结构             |
| 任意 provider/library        |
| 任意业务分层                 |
+-----------------------------+
```

Nucleus 只读取、标记、约束、验证。业务实现和技术栈属于用户项目。

## Source of Truth

Nucleus 必须明确事实源优先级，避免 CLI 输出、manifest 和 AI 判断互相污染。

事实源：

- Contract 文件是外部行为事实源，如 OpenAPI、proto、errors。
- Go 源码是 symbol、call graph、interface implementation 的事实源。
- `nucleus.yaml` 是协议索引，只记录 service、contracts、capabilities、symbols、edit surfaces、verification commands。
- `.nucleus/decisions/*.yaml` 是技术决策事实源，只记录 provider、library、driver、关键依赖选择及其理由。
- `.nucleus/recipes/*.yaml` 和能力 kind 词表是参考知识，不是事实源。
- `describe`、`trace`、`impact`、`adopt` 输出是可重建快照，不是长期事实源。

冲突处理：

- manifest 与源码冲突时，`describe` 应输出 stale/unresolved 诊断，不应自动改写。
- decision 与 manifest capability 冲突时，`decision validate` 失败。
- recipe 与现有代码事实冲突时，以代码事实为准。
- graph 推断不确定时必须标注 confidence，不得编造成确定事实。

## Schema 文件

协议必须实体化为 schema 文件，避免实现时各命令各写一套 JSON。

必须新增或重写：

- `contract/schema/manifest.v2.schema.json`
- `contract/schema/decision.v1.schema.json`
- `contract/schema/recipe.v1.schema.json`
- `contract/schema/graph.v1.schema.json`
- `contract/schema/adopt-result.v1.schema.json`
- `contract/schema/mcp-result.v1.schema.json`
- `contract/schema/recipe-result.v1.schema.json`
- `contract/schema/decision-result.v1.schema.json`
- `contract/schema/trace-result.v1.schema.json`
- `contract/schema/impact-result.v1.schema.json`
- `contract/schema/serve-result.v1.schema.json`
- `contract/schema/report.v1.schema.json`
- `contract/schema/describe-result.v1.schema.json`
- `contract/schema/validate-result.v1.schema.json`
- `contract/schema/lint-result.v1.schema.json`
- `contract/schema/gen-result.v1.schema.json`
- `contract/schema/scenario-result.v1.schema.json`
- `contract/schema/plan-result.v1.schema.json`
- `contract/schema/plan-executable.v1.schema.json`
- `contract/schema/diagnostic.v1.schema.json`
- `contract/schema/evidence.v1.schema.json`

所有 CLI JSON 输出必须带：

- `result_kind`
- `schema_version`
- `schema_ref`
- `ok`
- `diagnostics`

所有 MCP `structuredContent` 也必须带同样的 agent-facing envelope，除非它直接复用某个 CLI result schema。

## Diagnostic 规范

AI 依赖稳定错误结构。所有命令必须使用统一 diagnostic envelope。

示例：

```json
{
  "severity": "error",
  "code": "manifest.symbol_ambiguous",
  "path": "nucleus.yaml",
  "symbol_id": "go://example.com/order/internal/order#OrderStore",
  "message": "symbol name OrderStore matched multiple packages",
  "suggested_action": "rerun mark with a fully qualified symbol id"
}
```

字段：

- `severity`: `error | warning | info`
- `code`: 稳定机器码，禁止把自然语言塞进 code。
- `path`: 相关文件，可为空。
- `span`: 可选起止行列。
- `symbol_id`: 可选稳定 symbol ID。
- `message`: 给人类看的简短说明。
- `suggested_action`: 给 AI 的下一步动作建议。

诊断必须可机器处理，不能只依赖 stderr 文本。

## Manifest 设计

### 现有问题

当前 manifest 偏服务模板：

- `capabilities` 是字符串列表。
- `nucleus.providers` 容易表达成框架 provider registry。
- edit surfaces 与模板目录强相关。

### 新 manifest 方向

manifest 应成为 AI 协议索引，而不是项目结构说明书。

示例：

```yaml
schema_version: "2.0"

service:
  name: order-api
  version: "0.1.0"

contracts:
  - id: http
    kind: openapi
    path: api/openapi.yaml
  - id: errors
    kind: errors
    path: api/errors.yaml

capabilities:
  - id: order_store
    kind: relational_store
    intent: "persist and query orders"
    symbols:
      - OrderStore

ai:
  allowed_changes:
    - "**/*.go"
    - "api/**"
    - "configs/**"
  readonly:
    - "**/*.gen.go"
    - "vendor/**"
  forbidden:
    - "**/*secret*"
    - "**/*.local.yaml"

verify:
  commands:
    - "go test ./..."
```

### 字段解释

`contracts`：

- 只标记契约文件。
- 不推导项目结构。

`capabilities`：

- `id` 是服务内稳定标识。
- `kind` 是建议语义标签，如 `relational_store`、`cache`、`message_bus`、`logger`，但不是强枚举。
- `symbols` 是代码锚点，可以是 interface、type、function、method 或 package 内符号。
- manifest 不保存 provider、library、ORM、driver 等技术决策。

`ai`：

- 只定义编辑边界。
- 不暗示目录规范。

`verify`：

- 用户项目自己定义验证命令。
- Nucleus 可以建议，但不能替项目硬编码。

Manifest 应尽量薄。它不是配置中心，不保存 provider 细节，不保存 AI 的技术判断，只保存可长期稳定引用的协议索引。容易变化的判断放入 `.nucleus/decisions`，容易重建的事实放入 describe/adopt 输出。

## Capability 设计

### Capability 是语义锚点

Capability 不等于 provider，不等于 SDK，不等于目录。

Capability 应描述：

- 这个服务需要什么能力。
- AI 应通过哪个 interface 或 symbol 使用它。
- 变更这个能力时要检查哪些调用和验证命令。

### Capability kind

Nucleus 只能提供建议词表，不能把 kind 做成强枚举。原因是能力类型会随项目和时代变化，强枚举会把框架重新拉回硬编码。

建议词表应放在数据文件中，而不是写死在 Go 代码里。可选位置：

- 内置只读 `vocab/capability-kinds.yaml`。
- 用户项目 `.nucleus/vocab/capability-kinds.yaml`。

建议词表示例：

- `relational_store`
- `document_store`
- `cache`
- `message_bus`
- `http_client`
- `scheduler`
- `logger`
- `tracer`
- `metrics`
- `auth`
- `config`
- `lock`
- `rate_limiter`
- `custom`

这些 kind 只用于语义分类、搜索、计划提示和报表聚合，不绑定 Go module 或 provider。未知 kind 合法，lint 不得因为未知 kind 失败。用户项目词表可以增加、覆盖或隐藏内置建议。

### Provider 决策

Provider 决策不应写死在 Nucleus 代码里。

Provider 决策也不应写进 manifest。manifest 是稳定索引，decision 是 AI/用户对当前代码事实做出的可审查判断。AI 或用户可以生成 `.nucleus/decisions/*.yaml`：

```yaml
schema_version: "decision.v1"
id: order-store-provider
capability: order_store
decision:
  provider: gorm
  library: gorm.io/gorm
  status: proposed
  locked: false
reason:
  - "project already uses gorm.io/gorm in storage package"
  - "existing transaction helper accepts *gorm.DB"
impact:
  symbols:
    - OrderStore
    - NewOrderRepository
  files:
    - internal/order/store.go
    - internal/storage/db.go
verification:
  commands:
    - "go test ./internal/order ./internal/storage"
alternatives:
  - provider: database/sql
  - provider: xorm
```

注意：这里不是框架替项目做预设，而是 AI 对当前代码事实做结构化判断。

用户确认后，decision 应进入 locked 状态：

```yaml
decision:
  provider: gorm
  library: gorm.io/gorm
  status: accepted
  locked: true
  accepted_by: human
  accepted_at: "2026-07-03T00:00:00Z"
```

CLI 入口：

```bash
nucleus decision accept .nucleus/decisions/order-store-provider.yaml \
  --by human \
  --accepted-at 2026-07-03T00:00:00Z \
  --json
```

后续 AI 不允许静默替换 locked decision。若确实需要更换 provider 或关键依赖，必须新增 supersede decision：

```yaml
supersedes: order-store-provider
reason:
  - "locked decision is being replaced because ..."
```

CLI 入口：

```bash
nucleus decision supersede .nucleus/decisions/order-store-provider-v2.yaml --json
```

Decision hash 规范：

- hash 使用 canonical JSON 计算。
- canonical payload 不包含 `accepted_at`、本地绝对路径、格式化空白、diagnostics。
- canonical payload 必须包含 capability、decision provider/library/driver、reason、impact、verification、alternatives。
- locked decision 记录 `decision_hash`。
- supersede decision 记录 `supersedes` 和 `supersedes_hash`。

### Capability add 新语义

删除的命令形态：

```bash
nucleus capability add sql --provider postgres
```

新语义：

```bash
nucleus mark capability order_store \
  --kind relational_store \
  --symbol OrderStore
```

或：

```bash
nucleus capability declare order_store --kind relational_store --symbol OrderStore
```

CLI 只更新 manifest，不生成 provider implementation，不写 provider decision。

## Recipe 设计

### 为什么需要 recipe

Nucleus 不应硬编码 provider，但 AI 需要参考资料。

解决方式是 recipe 外置：

- 内置仓库可以提供少量通用 recipe。
- 用户项目可以在 `.nucleus/recipes` 放私有 recipe。
- AI 可以根据 recipe 生成 decision，但 Nucleus 不自动执行 recipe。

### Recipe 示例

```yaml
schema_version: "recipe.v1"
id: gorm-relational-store
kind: relational_store
provider: gorm
language: go
detect:
  imports:
    - gorm.io/gorm
suggest:
  interfaces:
    - "Repository interface should not expose *gorm.DB unless project already does"
  verification:
    - "go test ./..."
risks:
  - "transaction boundary must be explicit"
  - "avoid leaking ORM model into API contract"
```

Recipe 是知识，不是框架实现。

Recipe 安全边界：

- recipe 不允许包含自动执行脚本。
- recipe 不允许声明要写入的文件内容。
- recipe 中的 commands 只能作为 suggested verification。
- recipe 不得自动修改 manifest、decision、go.mod、源码或配置。
- recipe 命中结果只能作为 plan candidates，不能直接变成 accepted decision。

## 多服务与 Monorepo

Nucleus 必须支持 monorepo 和多 module，而不是假设一个 repo 只有一个服务。

规则：

- 允许一个 repo 下存在多个 `nucleus.yaml`。
- `--dir` 始终表示当前 Nucleus service root，不默认等于 git root。
- `adopt --dir` 只在指定目录创建协议索引，不扫描并接管整个仓库。
- `describe --dir` 默认只描述当前 service root，跨 service 关系必须通过 contracts/dependencies 显式声明。
- `.nucleus/workspace.yaml` 暂缓进入 MVP。若未来引入，只能作为服务索引，不作为控制平面。
- graph symbol ID 必须包含 module/package，避免跨 module 短名冲突。

## Graph 设计

### 输出目标

`nucleus describe --json` 应输出：

- `contract_graph`
- `symbol_graph`
- `call_graph`
- `interface_graph`
- `capability_graph`
- `flow_graph`
- `impact_index`
- `test_graph`

### Symbol Graph

节点：

- package
- file
- type
- interface
- method
- function
- variable/const

边：

- contains
- imports
- declares
- calls
- implements
- references
- returns
- accepts

### Symbol ID

所有 graph、mark、trace、impact 输出都必须使用稳定 symbol ID，不能只使用短名。

建议格式：

```text
go://<module>/<package-path>#<symbol>
go://<module>/<package-path>#<receiver>.<method>
```

示例：

```text
go://example.com/order/internal/order#OrderStore
go://example.com/order/internal/order#Service.CreateOrder
```

每个 symbol 节点还应包含：

- `name`：短名，如 `OrderStore`。
- `package`：Go package import path。
- `file`：相对文件路径。
- `span`：起止行列。
- `kind`：type、interface、function、method、const、var。

用户输入短名时，CLI 可以做候选解析；如果存在多个候选，必须返回 ambiguous diagnostics，不能猜。

### Graph Confidence

每个 graph edge 都必须带来源和可信度：

```json
{
  "from": "go://example.com/order/internal/http#Handler.CreateOrder",
  "to": "go://example.com/order/internal/order#Service.CreateOrder",
  "kind": "calls",
  "source": "ast",
  "confidence": "direct",
  "stale": false
}
```

字段：

- `source`: `ast | contract | manifest | mark | decision | recipe | heuristic`。
- `confidence`: `direct | inferred | unknown`。
- `stale`: 当前 edge 是否基于过期快照或未解析锚点。

`trace` 和 `impact` 不得把 `inferred` 或 `unknown` 边表达成确定事实。

### Flow Graph

微服务语义链：

```text
route -> handler -> function/method -> interface -> implementation -> external capability
```

示例 JSON：

```json
{
  "route": "POST /orders",
  "operation_id": "createOrder",
  "chain": [
    {"kind": "handler", "symbol": "Handler.CreateOrder", "file": "internal/http/order.go:21"},
    {"kind": "usecase", "symbol": "CreateOrder", "file": "internal/order/usecase.go:14"},
    {"kind": "interface", "symbol": "OrderStore", "file": "internal/order/store.go:6"},
    {"kind": "implementation", "symbol": "GormOrderStore", "file": "internal/storage/order_store.go:18"},
    {"kind": "capability", "id": "order_store", "kind": "relational_store"}
  ]
}
```

### Impact Graph

输入：

```bash
nucleus impact symbol OrderStore --json
```

输出：

- direct callers。
- transitive callers。
- affected routes。
- affected contracts。
- affected tests。
- affected capabilities。
- suggested verification commands。
- risk notes。

### Test Graph

Nucleus 应建立轻量测试关联：

- 同 package 测试。
- `_test.go` 中引用的 symbol。
- table-driven test 中出现的 operationId/error code。
- HTTP scenario cases 与 OpenAPI route 的关系。

AI 修改前应知道最小验证集，而不是只会跑 `go test ./...`。

## CLI 设计

### 主命令分层

```text
nucleus adopt       在任意 Go 项目中生成最小协议索引
nucleus describe    输出完整结构化事实
nucleus plan        根据任务输出编辑计划和影响面
nucleus apply       应用结构化 plan 中的文件编辑
nucleus execute     执行 allowlisted verification commands
nucleus verify      执行项目声明的验证
nucleus mark        标记 contract/capability/symbol/test 等锚点
nucleus trace       查询 route/symbol/capability 调用链
nucleus impact      查询变更影响面
nucleus decision    校验 AI/用户的结构化技术决策
nucleus report      输出本地质量报告
nucleus mcp         通过 stdio MCP 暴露本地结构化事实
```

### `nucleus adopt`

目标：

- 不生成业务代码。
- 不规定目录。
- 不选择 provider。
- 不修改 go.mod。

行为：

1. 检测 `go.mod`。
2. 扫描常见 contract 文件，但只作为候选。
3. 生成最小 `nucleus.yaml`。
4. 生成项目事实快照。
5. 生成可选 `.nucleus/README.md` 或 agent instruction。
6. 输出 adoption evidence。

示例：

```bash
nucleus adopt --dir . --service order-api --json
```

输出包含：

- detected module。
- package list summary。
- detected contracts。
- detected test commands。
- created files。
- generated file candidates。
- symbol index summary。
- skipped suggestions。
- warnings。

`adopt` 的价值不是初始化目录，而是给 AI 第一次进入项目时提供地图。它应该尽量输出事实，不做决策。

### `nucleus mark`

用于声明结构化锚点：

```bash
nucleus mark contract http --kind openapi --path api/openapi.yaml
nucleus mark capability order_store --kind relational_store --symbol OrderStore
nucleus mark verify "go test ./..."
```

mark 只改 manifest，不生成实现。

`mark` 对 symbol 有两种状态：

- `resolved`：symbol 当前存在，CLI 写入稳定 symbol ID。
- `declared`：symbol 当前不存在，CLI 只记录 intent，后续 `plan` 必须提示需要创建或绑定 symbol。

如果用户输入短名且存在多个候选，`mark` 必须失败并返回候选列表。禁止自动选择第一个匹配项。

manifest 记录示例：

```yaml
capabilities:
  - id: order_store
    kind: relational_store
    symbols:
      - id: go://example.com/order/internal/order#OrderStore
        status: resolved
      - name: FutureStore
        status: declared
```

### `nucleus capability`

删除旧 `capability add` 命令。

原因：

- `add` 暗示脚手架生成。
- `--provider` 暗示框架负责技术选型。
- 旧命令会诱导 AI 生成 provider 实现，而不是声明语义锚点。

替代命令：

```bash
nucleus mark capability order_store --kind relational_store --symbol OrderStore
nucleus decision validate artifacts/nucleus/decisions/order-store-provider.yaml
```

如果需要专门的 capability namespace，只允许声明式命令：

```bash
nucleus capability declare order_store --kind relational_store --symbol OrderStore
```

该命令只写 manifest，不生成 provider、不修改依赖、不创建目录结构。

### `nucleus plan`

输出应从路径计划升级为结构化作战图：

```json
{
  "task": "add order status filter",
  "task_type": "contract_and_logic_change",
  "required_contract_edits": ["api/openapi.yaml"],
  "suggested_edits": ["internal/order/query.go"],
  "blocked_edits": [],
  "affected_symbols": ["ListOrders", "OrderStore.List"],
  "affected_routes": ["GET /orders"],
  "affected_tests": ["internal/order/query_test.go"],
  "capabilities": ["order_store"],
  "commands": ["go test ./internal/order", "nucleus verify --dir . --json"],
  "risks": ["query filter changes persistence semantics"]
}
```

### `nucleus trace`

示例：

```bash
nucleus trace route "POST /orders" --json
nucleus trace symbol CreateOrder --json
nucleus trace capability order_store --json
```

用途：

- 给 AI 快速定位业务链。
- 减少反复 grep/read。
- 降低误改无关代码概率。

### `nucleus impact`

示例：

```bash
nucleus impact symbol OrderStore --json
nucleus impact file internal/order/store.go --json
nucleus impact contract api/openapi.yaml --json
```

用途：

- 变更前理解 blast radius。
- plan 中自动嵌入结果。
- review 中解释为什么改这些文件。

### `nucleus decision`

用于校验 AI 生成的技术决策：

```bash
nucleus decision validate artifacts/nucleus/decisions/order-store-provider.yaml
```

校验：

- capability 是否存在。
- provider 是否有 evidence。
- required changes 是否在 edit surfaces 内。
- verification commands 是否存在。
- 新增依赖是否有理由。
- locked decision 是否被静默替换。
- supersede decision 是否引用原 decision hash。

Locked decision 信任模型：

- `decision validate` 可以校验 locked decision，但不能默认创建 locked decision。
- 创建 locked decision 必须显式运行 `decision accept <path>`，或由用户手动编辑。
- locked decision 必须记录原始 decision hash。
- supersede decision 必须引用 `supersedes` 和 `supersedes_hash`。
- AI 后续计划若改变 locked provider/library/driver，必须先生成 supersede decision，否则 plan 应标记 blocked。

## MCP 设计

Nucleus 应提供 MCP 工具给 agent，而不是只靠 CLI 文本输出。

当前 stdio MCP 工具：

- `get_service_description`
- `get_edit_surfaces`
- `get_contracts`
- `get_capabilities`
- `trace_route`
- `trace_symbol`
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

MCP 返回结构化 JSON，不返回长篇自然语言。工具结果必须包含可被 agent 直接消费的结构化内容，而不是只把 CLI 文本塞进字符串。MCP `structuredContent` 必须包含 `result_kind`、`schema_version`、`schema_ref`、`ok`、`diagnostics`；复用 CLI 输出的工具直接返回对应 CLI result schema，其余 MCP 聚合工具使用 `mcp-result.v1`。

MCP 只读边界：

- 不写文件。
- 不执行验证命令。
- 不访问网络。
- 不选择 provider。
- 不生成 scaffold。
- 不把 recipe 变成 accepted 或 locked decision。

## 命令副作用矩阵

所有命令默认 local-only，不访问网络，不上传源码，不调用 control plane。

| Command | Read Files | Write Files | Execute Commands | Network | May Edit go.mod/go.sum |
| --- | --- | --- | --- | --- | --- |
| `adopt` | yes | only `nucleus.yaml` / `.nucleus/*` | no | no | no |
| `describe` | yes | no | no | no | no |
| `mark` | yes | only `nucleus.yaml` | no | no | no |
| `trace` | yes | no | no | no | no |
| `impact` | yes | no | no | no | no |
| `plan` | yes | no | no | no | no |
| `decision validate` | yes | no | no | no | no |
| `report` | yes | no | no | no | no |
| `mcp --stdio` | yes | no | no | no | no |
| `gen` | yes | generated contract artifacts only | no | no | no |
| `verify` | yes | evidence artifacts only | explicit manifest verify commands | no by default | no |
| `execute` | yes | evidence artifacts only | explicit allowlisted plan commands | no by default | no |
| `apply` | yes | plan-declared files inside edit surfaces | no | no | no |

`verify` 和 `execute` 只有在用户声明的命令本身访问网络时才可能触发网络行为。Nucleus 不应额外添加网络调用，也不应扩展命令列表。

## 生成策略

### 什么可以生成

可以生成：

- manifest。
- contract-derived metadata。
- route binder。
- type-safe request/response shell。
- decision draft。
- evidence。
- test scenario draft。

decision draft 只是候选文件，不能等同于 accepted 或 locked decision。任何会引入 provider/library/driver 的实现计划，都必须通过 `decision validate`，且 locked 状态必须由显式接受动作产生。

### 什么不应该生成

不应该生成：

- provider SDK wiring。
- ORM repository implementation。
- 数据库 migration。
- Dockerfile。
- Makefile。
- 固定 `internal` 目录结构。
- 固定 config loader。
- 固定 app container。

除非用户明确要求生成某个具体实现，并且 AI 已经给出 decision evidence。

`gen` 边界：

- 只能生成 contract 派生物。
- 不能生成 app wiring。
- 不能生成 runtime bootstrap。
- 不能生成 provider glue。
- 不能自动引入 runtime import。
- 不能修改 `go.mod/go.sum`。
- 不能把 generated artifacts 当作业务代码编辑入口。

Generated artifacts 规则：

- generated 文件必须带稳定 marker。
- marker 必须包含 source hash、schema version、generator version。
- generated 目标默认进入 readonly edit surface。
- AI 不得直接编辑 generated 文件，应修改 source contract 后重新生成。
- freshness marker 不得依赖本地绝对路径或时间戳。

## Report 设计

删除 platform 语义后，`report` 只输出本地质量报告。

允许字段：

- graph coverage。
- decision quality。
- locked decision drift。
- recipe candidate usage。
- verification status。
- edit surface violations。
- generated freshness。
- AI task evidence。
- unresolved/stale symbols。
- locked decision changes。

禁止字段：

- platform upload。
- release dry-run。
- control plane。
- production bridge readiness。
- provider SDK readiness。

Graph coverage 必须是可解释计数，不是神秘分数。允许示例：

- resolved symbols / declared symbols。
- direct edges / inferred edges / unknown edges。
- routes with resolved handler chain / total routes。
- capabilities with resolved symbols / total capabilities。
- tests linked to changed symbols / changed symbols.

## Verify、Execute、Lint

`verify`：

- 执行 manifest 中声明的验证集合。
- 校验 decision evidence、locked decision hash 和 supersedes hash。
- 只产出 evidence。
- 不擅自增加命令。
- 不修改源码、manifest、decision 或依赖文件。

`execute`：

- 执行 plan 中显式声明且 allowlisted 的命令。
- 只产出 evidence。
- 不读取未声明的命令建议。
- 不把 recipe commands 当作可执行命令。

`lint`：

- 只检查协议一致性、安全边界、decision evidence、generated freshness、graph stale/unresolved。
- 不检查固定目录分层。
- 不要求 `internal/app`、`internal/domain`、`internal/usecase` 等目录存在。
- 不因为 unknown capability kind 或 unknown provider decision 失败。

## 无 Contract 项目

没有 OpenAPI、proto、errors 的 Go 项目也合法。

此时 Nucleus 运行在 graph-only 模式：

- `adopt` 可以只生成 service、ai、verify、capabilities 空索引。
- `describe` 输出 symbol/call/test graph。
- `plan` 不能要求先创建 API contract，除非任务涉及外部行为。
- 不得使用 “library template” 或类似脚手架术语描述该模式。

## 模块边界

当前 `cap`、`bridge`、`runtime/*` 模块需要重新评估。

### `bridge`

建议删除。

原因：

- Nucleus 不应提供 provider SDK adapter 集合。
- provider 选择属于项目 decision，不属于框架内置能力。
- 保留 bridge 模块会持续诱导贡献者添加 gorm、xorm、nacos、redis、kafka 等实现。

如果未来需要参考实现，应放在独立 examples 或 recipes，不进入核心模块边界。

工程动作：

- 移除 `bridge` submodule。
- 移除 README、implementation status、root docs 中的 bridge 核心模块描述。
- 移除 lint/import mapping 中对 `github.com/nucleuskit/bridge/*` 的特殊识别。
- 移除 capability/provider catalog 对 bridge candidates 的输出。
- 若需要示例 adapter，以独立 recipe 或外部 example 表达，不进入核心仓库模块边界。

### `cap`

建议收缩到协议类型，甚至先暂停扩展。

允许保留：

- 极少量与 Nucleus 协议直接相关的通用类型。
- no-op/testing helper，且不能表达某个具体 provider 偏好。

不应保留：

- 一套覆盖 Redis、MQ、SQL、Mongo、Metric、Trace、Config 的接口全集。
- provider-specific option。
- 试图统一所有基础设施的抽象层。

### `runtime/*`

运行时模块也要接受同一条边界：

- 可以提供轻量协议适配，如 HTTP route binder 需要的最小接口。
- 不应成为应用容器、配置系统、服务发现系统或启动框架。
- 任何 runtime import 都必须是用户显式选择，而不是 `adopt` 或 capability 默认引入。

## 文档拆分

当前文件是总设计文档，用于统一方向。进入实施前应拆成三类文档，避免一份大文档同时承担决策、设计和任务跟踪。

建议拆分：

- ADR：记录为什么删除 template、provider scaffold、platform 叙事、bridge 核心模块。
- Design：记录 manifest v2、decision v1、graph schema、trace/impact 输出协议。
- Implementation Plan：记录具体删除文件、修改包、测试用例和 PR 顺序。

总设计文档可以保留为索引，但具体执行以拆分后的文档为准。

## MVP 顺序

最小可用闭环应该优先解决 AI 看不清、定位慢、影响面不明的问题，而不是优先生成代码。

推荐顺序：

```text
adopt -> describe graph -> trace -> impact -> plan -> decision -> verify
```

`gen` 可以保留 contract-derived 生成能力，但不应成为 MVP 主线。Nucleus 的差异化不是“生成更多代码”，而是“让 AI 更准确地理解和修改现有代码”。

## 测试策略

除功能正向测试外，必须加入反能力测试，确保 Nucleus 不会做不该做的事。反能力测试应成为 CI gate，而不是普通建议。

必须测试：

- `adopt` 不修改 `go.mod/go.sum`。
- `adopt` 不生成业务目录。
- capability 命令不生成 provider SDK wiring。
- capability 命令不创建 repository、migration、Dockerfile、Makefile。
- unknown capability kind 合法。
- unknown provider decision 合法，只要 decision schema 和 evidence 完整。
- manifest 不允许 provider/library/driver 字段。
- lint 不要求 `internal/app`、`internal/domain`、`internal/usecase` 等目录。
- plan 不输出 provider 默认选择。
- report 不输出 platform upload、release dry-run、control plane 字段。
- trace/impact 在 graph 不完整时输出 confidence/unknown，而不是编造确定链路。
- generated 文件默认 readonly，且 freshness marker 不含时间戳/绝对路径。
- no-contract graph-only 项目合法。

只要出现以下行为，CI 必须失败：

- `adopt` 或 capability 命令修改 `go.mod/go.sum`。
- 生成 provider SDK wiring。
- 要求固定目录存在。
- unknown kind/provider 导致失败。
- manifest 接受 provider/library/driver 字段。
- report 输出 platform/control plane 字段。
- lint 要求固定分层目录。
- recipe 触发写文件或执行命令。

## 代码改造计划

### 阶段 1：删除错误方向并收紧定位

目标：

- 删除 template/platform 主路径。
- 文档从 template/platform 改为 protocol/adoption。
- 明确 Nucleus 不做 provider selection。

任务：

- 修改 `README.md` 中 Project Shape、Roadmap、Capability Protocol 表述。
- 修改 `docs/adr/0001-project-scope.md`，移除 `template` 作为核心 area。
- 新增 ADR：Nucleus is an agent-native protocol layer, not a scaffold.
- 新增 manifest/decision/recipe/graph/report/diagnostic/adopt-result schema 文件。
- 删除 `docs/platform-mapping.md` 或重写为纯本地 quality report 文档。
- 删除 `examples/deploy/docker` 平台化示例。
- 删除以 service/worker/library template 为主的 example 文档。
- 删除 `cmd/nucleus/internal/initcmd` 或重写为 `adopt`。

验收：

- README 第一屏不再强调 template。
- 主流程展示 `adopt -> describe -> plan -> impact -> verify`。
- 文档明确 provider 不由框架决定。
- 仓库中不存在平台化发布/上传/readiness 叙事。

### 阶段 2：新增 `adopt` 和项目事实快照

目标：

- 任意 Go 项目可加入 Nucleus。
- AI 第一次进入项目时拿到结构化地图。

任务：

- 新增 `cmd/nucleus/internal/adopt`。
- 扫描 `go.mod`、contract candidates、test command candidates。
- 生成最小 `nucleus.yaml`。
- 输出 package list summary、generated file candidates、symbol index summary。
- 输出 `nucleus.adopt_result`。
- 加测试覆盖空项目、已有 openapi、无 contract 的 library、monorepo 子目录。

验收：

- `nucleus adopt --dir . --json` 不生成业务代码。
- 不修改 `go.mod/go.sum`。
- 不要求特定目录。
- 无 contract 项目合法。
- 可对当前仓库自身运行。

### 阶段 3：增强 describe graph

目标：

- 让 AI 用一次查询理解项目结构、逻辑链和影响面基础数据。

任务：

- 在 `contract/inspect` 增加 symbol graph。
- 定义稳定 symbol ID。
- 解析 package、file、type、interface、method、function。
- 增加 calls/references/implements edges。
- 增加 route 到 handler 到 symbol chain。
- 增加 interface implementation detection。
- 增加 test relation detection。
- 为 graph edge 输出 source、confidence、stale。
- 输出 graph schema。

验收：

- `describe --json` 包含 symbol/call/interface/test graph。
- 能查询 symbol callers/callees。
- 能从 route 找到 handler chain。
- 能从 interface 找到 implementations。
- 短名冲突时返回 ambiguous diagnostics。

### 阶段 4：新增 trace/impact

目标：

- 让 AI 快速定位修改影响。

任务：

- 新增 `cmd/nucleus/internal/trace`。
- 新增 `cmd/nucleus/internal/impact`。
- `plan` 自动嵌入 impact summary。
- 输出稳定 JSON。

验收：

- `nucleus trace route` 可返回 route execution chain。
- `nucleus impact symbol` 可返回 affected symbols/files/routes/tests。
- `plan` 中不再只有路径，还包含 affected graph。

### 阶段 5：重构 capability

目标：

- capability 从 provider scaffold 改为 semantic anchor。

任务：

- 将 `capcatalog.Provider` 从核心流程移除。
- 删除 provider 强枚举和 provider 默认值。
- 删除 capability kind 强枚举校验，只保留建议词表。
- 将 capability kind 建议词表外置为 vocab 数据。
- 删除 `postgresOperations` 特例。
- 删除自动写 `go.mod/go.sum` 行为。
- 新增 capability object schema。
- 增加 `nucleus mark capability`。
- 删除 `cmd/nucleus/internal/capability` 旧 add/scaffold 实现，或重写为纯 declaration。

验收：

- 添加 mysql/gorm/xorm/vector_store/payment_gateway/custom kind 不需要修改 Nucleus 源码。
- capability 命令不引入任何 provider SDK。
- `lint` 不因未知 kind 或未知 provider 失败，只要求 capability 有锚点或 decision。

### 阶段 6：decision/evidence 闭环

目标：

- AI 技术选型可审查、可锁定、可回放。

任务：

- 定义 `decision.v1` schema。
- 新增 `nucleus decision validate`。
- 支持 locked decision 和 supersede decision。
- 支持 decision hash、supersedes_hash。
- hash 使用 canonical payload，排除 accepted_at、绝对路径、diagnostics。
- `plan` 对新增依赖/provider 要求 decision。
- `verify/report` 汇总 decisions。

当前实现状态：

- `decision validate`、`decision accept`、`decision supersede` 已实现。
- `plan` 已通过 `blocked_decisions` 阻止未带 supersede evidence 的 locked provider/library/driver 替换。
- `verify` 已新增 `decision` evidence step，并把 decision diagnostics 合并到顶层 diagnostics。
- `report` 已输出 `ai_quality.decision_quality`，包含 files、valid、errors、warnings、accepted_locked、supersedes、drift。

验收：

- 新 provider、新依赖、新外部服务都有 decision evidence。
- locked decision 不能被后续 AI 静默替换。
- supersede decision 必须引用原 decision hash。
- decision 中 required changes 必须落在 edit surfaces。
- report 能展示 AI 决策质量。

### 阶段 7：外置 recipe

目标：

- provider 知识外置，框架不硬编码。

任务：

- 定义 `recipe.v1` schema。
- 支持 `.nucleus/recipes/*.yaml`。
- 支持内置只读 recipes 目录。
- `plan` 可以读取 recipe 作为候选建议。
- 输出 candidate，不自动选择。

当前实现状态：

- MCP 已支持读取项目本地 `.nucleus/recipes/*.yaml`、`.yml`、`.json` 和内置只读 recipe。
- 已支持内置只读 recipes，当前内置项以 provider-neutral 的 `sql-port-boundary` 为主，不编码 ORM、driver、依赖、DSN、命令或文件写入默认值。
- 项目本地 recipe 与内置 recipe 同 id 时，本地 recipe 覆盖内置 recipe；候选结果通过 `source: project|builtin` 标记来源。
- recipe 使用 strict schema，未知字段会失败，避免把脚本或写文件意图混入知识文件。
- `plan` 已输出 `context.recipe_candidates`、`context.recipe_diagnostics` 和 `context.recipe_policy`。
- `report` 已从 AI task evidence 中汇总 `recipe_candidate_usage`，统计 candidate task、candidate count、decision-required count 和 unique candidate ids。
- recipe candidates 只作为 `candidate_only` 参考，不会改变 plan commands、generated outputs、provider decision 或 locked decision 状态。
- 无效 recipe 会被排除在 candidates 之外，并作为诊断和风险提示暴露给 agent。

验收：

- 新增 provider recipe 不需要改 Go 代码。
- 未命中 recipe 时仍可使用 custom provider。
- AI 看到的是候选项和理由，不是默认实现。
- recipe 不允许写文件、执行脚本或直接生成 accepted decision。

### 阶段 8：MCP 化

目标：

- 让 Codex、Claude Code、Cursor 等 agent 直接查询结构化上下文。

任务：

- 暴露 MCP server。
- 工具覆盖 describe、trace、impact、plan、decision validation。
- 输出 schema 化 JSON。

当前实现状态：

- 已提供 `nucleus mcp --stdio`。
- 已覆盖 contracts、edit surfaces、capabilities、symbols、trace、impact、decision validation、report、plan 和 recipes。
- MCP 工具是 local-only/read-only，不执行命令、不写文件、不做 provider 选择。

验收：

- agent 不需要 shell grep/read 多轮查找即可获取调用链。
- MCP 工具结果可直接进入 plan。

## 删除策略

项目未正式使用，不保留旧协议。

直接删除：

- `nucleus init --template service|worker|library`。
- `cmd/nucleus/internal/initcmd` 中的模板生成实现。
- `nucleus capability add --provider`。
- `cmd/nucleus/internal/capability` 中 provider scaffold、go.mod/go.sum 写入、postgres 特例。
- `capcatalog` 中 provider 默认值、provider 枚举、planning keyword provider hint。
- `report --platform`。
- platform upload、release dry-run、platform mapping、docker platform local 示例。
- manifest v1 的 `capabilities: []string` 主模型。
- manifest 中 provider/library/driver 决策字段。
- capability kind 强枚举校验。
- `bridge` 模块作为核心模块。

直接重写：

- manifest schema 到 v2 object model。
- capability graph 读取 v2 capability objects。
- lint 中依赖硬编码 capability module/provider/kind 的规则。
- plan 中 capability/provider 的自然语言猜测逻辑。
- `cap` 模块边界，收缩为协议类型或暂缓扩展。

如果旧测试覆盖这些行为，应删除或改写测试，不做兼容断言。

## 风险和约束

### 风险 1：过度抽象

如果 capability kind 太抽象，AI 可能不知道如何落地。

应对：

- 用 recipe 提供建议。
- 用 graph 展示现有项目事实。
- 用 decision 记录 AI 判断。

### 风险 2：graph 不准确

Go 动态调用、interface、多态、反射会让静态分析不完整。

应对：

- 输出 confidence。
- 区分 `direct`、`inferred`、`unknown`。
- 允许用户 mark symbol/capability 补充事实。

### 风险 3：AI 决策质量不稳定

AI 可能选择不适合的库。

应对：

- 强制 decision evidence。
- 强制 alternatives。
- 强制 verification。
- report 汇总失败决策。

## 成功标准

### 产品层

- 用户可以在任意 Go 项目执行 `nucleus adopt`。
- Nucleus 不要求项目目录符合某个模板。
- AI 可以通过 Nucleus 获取结构化上下文，而不是靠大量 grep。
- 新增 mysql/gorm/xorm/vector_store/payment_gateway/custom kind 或 provider 不需要改 Nucleus 源码。
- 用户可以拒绝任何 provider 默认选择，因为框架不做默认选择。

### 技术层

- `describe` 输出可解释的 graph。
- `plan` 输出 affected symbols/routes/tests。
- `impact` 能定位 blast radius。
- `capability` 不再自动引入 provider SDK。
- `capability` 不强校验 kind 枚举。
- manifest 不包含 provider/library/driver。
- `decision` 能校验 AI 技术选型。
- locked decision 需要 supersede decision 才能替换。
- `verify` 能基于项目自己的命令产出 evidence。

### 心智层

Nucleus 从：

```text
一个 AI-first Go 微服务脚手架/平台
```

变成：

```text
任意 Go 微服务都可加入的 AI 结构化协议层
```

## 推荐第一批 PR

### PR 1：删除错误方向并收敛文档

- 改 README。
- 新增 ADR。
- 拆分 ADR/Design/Implementation Plan。
- 新增 schema 文件清单。
- 定义 diagnostic envelope。
- 定义命令副作用矩阵。
- 删除 template/platform 叙事。
- 删除 platform mapping 或重写为本地质量报告。
- 删除 `bridge` 核心模块定位。
- 明确 Source of Truth 优先级。

### PR 2：删除 template init，新增 `adopt`

- 新增命令。
- 删除 `cmd/nucleus/internal/initcmd` 模板生成路径。
- 输出检测结果和创建的最小 manifest。
- 输出项目事实快照。
- 支持 monorepo 子目录。
- 支持 no-contract graph-only 项目。
- 不写业务代码。

### PR 3：describe graph MVP

- 定义 symbol ID。
- 输出 package/file/type/interface/function/method。
- 输出 calls。
- 输出 implements。
- 输出 test relation summary。
- edge 输出 source/confidence/stale。

### PR 4：trace/impact MVP

- `trace symbol` 支持 callers/callees。
- `trace route` 支持 route 到 handler 的链路。
- `impact symbol` 支持 affected files/routes/tests。
- graph 不完整时输出 confidence/unknown。

### PR 5：capability object schema

- 删除 `postgresOperations`。
- 删除 `capability add --provider`。
- capability 命令不再改 `go.mod/go.sum`。
- manifest 直接切到 capability object。
- manifest 不允许 provider/library/driver 字段。
- 删除 v1 string list 主模型和相关测试。
- unknown kind 合法。
- capability kind 建议词表外置。

### PR 6：decision v1 和 locked decision

- 定义 decision schema。
- provider/library/driver 只允许出现在 decision。
- 支持 locked 和 supersede。
- 支持 decision hash 和 supersedes_hash。
- `decision validate` 校验 edit surfaces 和 verification。
- canonical hash 排除 accepted_at、绝对路径、diagnostics。

### PR 7：反能力测试

- 覆盖不改 `go.mod/go.sum`。
- 覆盖不生成 provider SDK、repository、migration。
- 覆盖不要求固定目录。
- 覆盖不输出 platform 字段。
- 覆盖 unknown kind/provider 不失败。
- 将反能力测试纳入 CI gate。
- 覆盖 recipe 不写文件、不执行命令。
- 覆盖 generated marker 不含时间戳/绝对路径。

### PR 8：report 本地质量模型

- 删除 platform/control plane 字段。
- 输出 graph coverage、decision quality、verification status。
- 输出 unresolved/stale symbols。
- 输出 locked decision changes。
- 输出 recipe candidate usage。
- graph coverage 使用可解释计数。

### PR 9：verify/execute/lint 职责收敛

- `verify` 只执行协议验证、decision evidence 校验和 manifest 声明验证集合。
- `execute` 只执行 plan allowlisted commands。
- `lint` 只检查协议一致性、安全边界、decision evidence、generated freshness、graph stale。
- `lint` 不检查固定目录分层。

### PR 10：删除 bridge 工程动作

- 移除 `bridge` submodule。
- 移除文档和 implementation status 中 bridge 核心模块描述。
- 移除 lint/import mapping 中 bridge 特殊识别。
- 移除 capability/provider catalog 的 bridge candidates。

## 最终判断

Nucleus 最有价值的方向不是“帮人生成一个标准微服务项目”，而是“让 AI 理解并安全修改任何 Go 微服务项目”。

因此，所有硬编码都应接受一条判断：

```text
如果它是协议、安全、证据或语言事实，可以固化。
如果它是技术栈、provider、目录结构或业务实现，必须外置给 AI/用户判断，并留下结构化 evidence。
```

只要守住这条线，Nucleus 就会从又一个框架，变成 AI 时代真正有差异化的 Go 微服务协议内核。
