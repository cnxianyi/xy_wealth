# xy-wealth

面向多数据源的资产聚合服务。当前接入 Binance Spot 账户资产，HTTP API 由 Gin 统一暴露；PostgreSQL 使用原生 `database/sql`，Redis 用于后续缓存、锁和任务状态，不引入 ORM。

Go module 对应正式仓库：`github.com/cnxianyi/xy_wealth`。

## 目录

```text
cmd/server/                         服务入口
configs/                           非敏感配置模板
internal/app/                      依赖组装与应用生命周期
internal/config/                   Viper 配置加载和校验
internal/domain/                   跨模块核心数据类型
internal/modules/exchange/         交易所稳定接口
internal/modules/exchange/binance/ Binance API 适配器
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

## HTTP API

| Method | Path | Description |
| --- | --- | --- |
| GET | `/health/live` | 进程存活检查 |
| GET | `/health/ready` | PostgreSQL 与 Redis 就绪检查 |
| GET | `/api/v1/exchanges/binance/balances` | Binance Spot 余额 |
| GET | `/api/v1/summary/exchanges` | 所有交易所聚合结果 |
| GET | `/api/v1/summary/banks` | 所有银行聚合结果（当前为空） |
| GET | `/api/v1/summary` | 所有分类的统一聚合结果 |

金额始终以十进制字符串返回，避免 JSON 浮点数造成精度损失。聚合接口允许部分成功：每个 provider 都有独立的 `status` 和可选 `error`。

## 常用命令

```bash
make deps
make fmt
make vet
make test
make build
```

首次创建 PostgreSQL 数据卷时，Compose 会执行 `migrations/000001_create_exchange_snapshots.up.sql`。生产环境应由发布流程中的迁移工具按顺序执行 `migrations`，不要依赖应用进程自动改表。
