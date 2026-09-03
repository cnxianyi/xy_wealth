## 项目约束

- 使用 Gin 作为 HTTP 入口，Viper 管理配置，Zap 管理日志。
- PostgreSQL 和 Redis 是基础依赖；PostgreSQL 只使用原生 SQL（`database/sql`/`pgx`），禁止引入 ORM。
- 配置、迁移、日志和 HTTP 层按 Go 与 GitHub 常见项目约定组织；敏感配置只能通过环境变量或本地配置文件注入，不提交密钥。
- 就绪检查必须能够反映 PostgreSQL 与 Redis 的连接状态；修改连接配置时应验证 local 配置是否可以连通。

## 模块与目录

- 业务按领域和 provider 分模块，交易所模块位于 `internal/modules/exchange/`。
- Binance 是第一个交易所 provider，按能力区分 Spot、USDⓈ-M Futures 与 COIN-M Futures；后续交易所（如 Bitget）应以独立 provider 扩展，不要把 provider 特有逻辑散落到公共层。
- Weex 使用独立 provider 包和 Spot/Contract 双 REST 域名；初始化阶段不得注册未实现的业务接口。
- 后续银行等其他资产来源使用独立领域模块（如 `internal/modules/bank/`），通过统一聚合服务对外提供汇总数据。
- Gin 路由统一暴露各 provider 的数据和跨 provider 汇总接口，路由层不承担上游协议转换和业务逻辑。

## API 文档与 Binance 接入

- API 文档使用 OpenAPI `3.1.0`。
- 交互式文档只使用 Scalar（`@scalar/api-reference`），禁止接入 Swagger 或 Swagger UI。
- Scalar 侧边栏按类似 `Exchanges → Binance → Spot / USDⓈ-M Futures` 组织。
- Binance 先接入官方 Spot 基础只读 API，再接入 Futures（USDⓈ-M、COIN-M）基础只读 API；新增接口应同时更新 provider、handler、路由、OpenAPI 规范和测试。
- 上游 Binance 响应必须经过明确的 JSON 解码和错误归一化，向客户端返回统一的错误结构，不泄露密钥、签名或完整敏感请求 URL。

## git

提交使用 Angular / Conventional Commits 风格：

subject 允许使用中文更好的描述

每次完成一个阶段可直接提交git保证工作区干净

```text
<type>(<scope>): <subject>
```
