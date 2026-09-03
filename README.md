# xy-wealth

面向多数据源的资产聚合服务。当前接入 Binance Spot、USDⓈ-M Futures、COIN-M Futures、Weex Spot 和 Weex Contract 基础只读接口，HTTP API 由 Gin 统一暴露；PostgreSQL 使用原生 `database/sql`，Redis 用于后续缓存、锁和任务状态，不引入 ORM。

Go module 对应正式仓库：`github.com/cnxianyi/xy_wealth`。

## 目录

```text
cmd/server/                         服务入口
api/                                OpenAPI 3.1 规范与交互式文档
configs/                           非敏感配置模板
internal/app/                      依赖组装与应用生命周期
internal/config/                   Viper 配置加载和校验
internal/domain/                   跨模块核心数据类型
internal/modules/exchange/         交易所稳定接口
internal/modules/exchange/binance/ Binance API 适配器
internal/modules/exchange/weex/    Weex API 适配器
internal/modules/bank/             银行接口扩展点
internal/modules/summary/          跨数据源并发聚合
internal/platform/                 PostgreSQL、Redis、Zap 等基础设施
internal/transport/http/           Gin 路由、中间件和 Handler
migrations/                        原生 SQL 迁移
```

依赖方向是 `transport -> modules -> domain`，外部系统适配器实现模块接口。新增交易所或银行时，应创建新的 provider 包并在 `internal/app/app.go` 注册。

## 本地启动

要求 Go 1.25+ 和 Docker Compose。

```bash
cp .env.example .env
docker compose up -d postgres redis
set -a; source .env; set +a
go run ./cmd/server
```

也可以复制 `configs/config.example.yaml` 为 `configs/config.yaml`。环境变量优先级更高，命名规则为 `XY_WEALTH_` 加配置路径，例如 `postgres.dsn` 对应 `XY_WEALTH_POSTGRES_DSN`。可通过 `XY_WEALTH_CONFIG_FILE` 指定其他配置文件。

Binance 账户接口需要只读 API Key。请启用读取权限、配置 IP 白名单，并且不要授予提现权限。未配置密钥时，服务仍可启动，但 Binance 相关响应会标记上游配置错误。

Binance USDⓈ-M Futures 使用 `binance.futures_base_url`（默认 `https://fapi.binance.com`），与 Spot 的 `binance.base_url` 相互独立。

Binance COIN-M Futures 使用 `binance.coin_m_futures_base_url`（默认 `https://dapi.binance.com`），与 USDⓈ-M Futures 相互独立。

Weex V3 使用独立的 Spot 与 Contract REST 域名：`weex.spot_base_url`（默认 `https://api-spot.weex.com`）和 `weex.contract_base_url`（默认 `https://api-contract.weex.com`）。当前已接入 Spot 与 Contract 基础只读接口，以及 Spot 账户余额查询。

Weex 公共 Spot/Contract 接口不需要密钥；账户余额查询需要配置 API Key、Secret Key 和 Passphrase，并通过签名请求访问。

## HTTP API

| Method | Path | Description |
| --- | --- | --- |
| GET | `/docs` | Scalar API Reference 交互式文档 |
| GET | `/openapi.yaml` | OpenAPI 3.1.0 原始规范 |
| GET | `/health/live` | 进程存活检查 |
| GET | `/health/ready` | PostgreSQL 与 Redis 就绪检查 |
| GET | `/api/v1/exchanges/binance/balances` | Binance Spot 余额 |
| GET | `/api/v1/exchanges/weex/balances` | Weex Spot 余额 |
| GET | `/api/v1/exchanges/binance/spot/ping` | Binance Spot 连通性检查 |
| GET | `/api/v1/exchanges/binance/spot/time` | Binance Spot 服务器时间 |
| GET | `/api/v1/exchanges/binance/spot/exchange-info` | Binance Spot 交易规则 |
| GET | `/api/v1/exchanges/binance/spot/depth?symbol=BTCUSDT` | Binance Spot 订单簿 |
| GET | `/api/v1/exchanges/binance/spot/klines?symbol=BTCUSDT&interval=1m` | Binance Spot K 线 |
| GET | `/api/v1/exchanges/binance/spot/ticker/24hr?symbol=BTCUSDT` | Binance Spot 24 小时行情 |
| GET | `/api/v1/exchanges/binance/spot/ticker/price?symbol=BTCUSDT` | Binance Spot 最新价格 |
| GET | `/api/v1/exchanges/binance/spot/ticker/book?symbol=BTCUSDT` | Binance Spot 最优买卖价 |
| GET | `/api/v1/exchanges/weex/spot/ping` | Weex Spot 连通性检查 |
| GET | `/api/v1/exchanges/weex/spot/time` | Weex Spot 服务器时间 |
| GET | `/api/v1/exchanges/weex/spot/exchange-info` | Weex Spot 交易规则 |
| GET | `/api/v1/exchanges/weex/spot/depth?symbol=BTCUSDT` | Weex Spot 订单簿 |
| GET | `/api/v1/exchanges/weex/spot/klines?symbol=BTCUSDT&interval=1m` | Weex Spot K 线 |
| GET | `/api/v1/exchanges/weex/spot/ticker/24hr?symbol=BTCUSDT` | Weex Spot 24 小时行情 |
| GET | `/api/v1/exchanges/weex/spot/ticker/price?symbol=BTCUSDT` | Weex Spot 最新价格 |
| GET | `/api/v1/exchanges/weex/spot/ticker/book?symbol=BTCUSDT` | Weex Spot 最优买卖价 |
| GET | `/api/v1/exchanges/weex/futures/usdm/ping` | Weex Contract 连通性检查 |
| GET | `/api/v1/exchanges/weex/futures/usdm/time` | Weex Contract 服务器时间 |
| GET | `/api/v1/exchanges/weex/futures/usdm/exchange-info` | Weex Contract 交易规则 |
| GET | `/api/v1/exchanges/weex/futures/usdm/depth?symbol=BTCUSDT` | Weex Contract 订单簿 |
| GET | `/api/v1/exchanges/weex/futures/usdm/klines?symbol=BTCUSDT&interval=1m` | Weex Contract K 线 |
| GET | `/api/v1/exchanges/weex/futures/usdm/ticker/24hr?symbol=BTCUSDT` | Weex Contract 24 小时行情 |
| GET | `/api/v1/exchanges/weex/futures/usdm/ticker/price?symbol=BTCUSDT` | Weex Contract 指数价格 |
| GET | `/api/v1/exchanges/weex/futures/usdm/ticker/book?symbol=BTCUSDT` | Weex Contract 最优买卖价 |
| GET | `/api/v1/exchanges/weex/futures/usdm/premium-index?symbol=BTCUSDT` | Weex Contract 标记价格和资金费率 |
| GET | `/api/v1/exchanges/binance/futures/usdm/ping` | USDⓈ-M Futures 连通性检查 |
| GET | `/api/v1/exchanges/binance/futures/usdm/time` | USDⓈ-M Futures 服务器时间 |
| GET | `/api/v1/exchanges/binance/futures/usdm/exchange-info` | USDⓈ-M Futures 交易规则 |
| GET | `/api/v1/exchanges/binance/futures/usdm/depth?symbol=BTCUSDT` | USDⓈ-M Futures 订单簿 |
| GET | `/api/v1/exchanges/binance/futures/usdm/klines?symbol=BTCUSDT&interval=1m` | USDⓈ-M Futures K 线 |
| GET | `/api/v1/exchanges/binance/futures/usdm/ticker/24hr?symbol=BTCUSDT` | USDⓈ-M Futures 24 小时行情 |
| GET | `/api/v1/exchanges/binance/futures/usdm/ticker/price?symbol=BTCUSDT` | USDⓈ-M Futures 最新价格 |
| GET | `/api/v1/exchanges/binance/futures/usdm/ticker/book?symbol=BTCUSDT` | USDⓈ-M Futures 最优买卖价 |
| GET | `/api/v1/exchanges/binance/futures/usdm/premium-index?symbol=BTCUSDT` | 标记价格和资金费率 |
| GET | `/api/v1/exchanges/binance/futures/coinm/ping` | COIN-M Futures 连通性检查 |
| GET | `/api/v1/exchanges/binance/futures/coinm/time` | COIN-M Futures 服务器时间 |
| GET | `/api/v1/exchanges/binance/futures/coinm/exchange-info` | COIN-M Futures 交易规则 |
| GET | `/api/v1/exchanges/binance/futures/coinm/depth?symbol=BTCUSD_PERP` | COIN-M Futures 订单簿 |
| GET | `/api/v1/exchanges/binance/futures/coinm/klines?symbol=BTCUSD_PERP&interval=1m` | COIN-M Futures K 线 |
| GET | `/api/v1/exchanges/binance/futures/coinm/ticker/24hr?symbol=BTCUSD_PERP` | COIN-M Futures 24 小时行情 |
| GET | `/api/v1/exchanges/binance/futures/coinm/ticker/price?symbol=BTCUSD_PERP` | COIN-M Futures 最新价格 |
| GET | `/api/v1/exchanges/binance/futures/coinm/ticker/book?symbol=BTCUSD_PERP` | COIN-M Futures 最优买卖价 |
| GET | `/api/v1/exchanges/binance/futures/coinm/premium-index?symbol=BTCUSD_PERP` | COIN-M Futures 标记价格和资金费率 |
| GET | `/api/v1/summary/exchanges` | 所有交易所聚合结果 |
| GET | `/api/v1/summary/banks` | 所有银行聚合结果（当前为空） |
| GET | `/api/v1/summary` | 所有分类的统一聚合结果 |

金额始终以十进制字符串返回，避免 JSON 浮点数造成精度损失。聚合接口允许部分成功：每个 provider 都有独立的 `status` 和可选 `error`。

当前 Binance Spot 首期只读基础接口已接入：连通性、服务器时间、交易规则、订单簿、K 线和行情。下单、撤单、订单查询、账户流水及用户数据流属于后续阶段，暂未开放。

当前 USDⓈ-M Futures 同样只开放只读基础行情接口；合约下单、持仓、保证金、账户资产和用户数据流属于后续阶段。

当前 COIN-M Futures 同样只开放只读基础行情接口；合约下单、持仓、保证金、账户资产和用户数据流属于后续阶段。

当前 Weex Spot 开放连通性、服务器时间、交易规则、订单簿、K 线、行情和账户余额查询；Weex Contract 开放对应的只读合约行情接口。Spot/Contract 交易写操作属于后续阶段，分别使用 `/api/v1/exchanges/{provider}/spot/...` 和 `/api/v1/exchanges/{provider}/futures/usdm/...` 路由。

`/docs` 使用固定版本的 Scalar API Reference 渲染 `/openapi.yaml`，浏览器需要能够访问 jsDelivr CDN。文档侧边栏按嵌套标签组织为 `Exchanges → Binance → Spot / USDⓈ-M Futures / COIN-M Futures` 与 `Exchanges → Weex → Spot / Contract`，并预留 `Bitget` 节点。OpenAPI 规范和文档页面作为 Go embed 资源编入服务二进制，部署时无需额外挂载文件。

## 常用命令

```bash
make deps
make fmt
make vet
make test
make build
```

首次创建 PostgreSQL 数据卷时，Compose 会执行 `migrations/000001_create_exchange_snapshots.up.sql`。生产环境应由发布流程中的迁移工具按顺序执行 `migrations`，不要依赖应用进程自动改表。
