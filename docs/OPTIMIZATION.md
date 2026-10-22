# rest 优化方案

| # | 优先级 | 问题 | 位置 | 建议 | 工作量 |
|---|---|---|---|---|---|
| 1 | 高 | **数组过滤会绕过租户隔离**：只要 filter 里有数组值（如 `filter={"id":[...]}`），`q.IsReference=true`，函数**只返回 `inFilters`**，商户条件 `_.meta.merchant_id` 和其他条件全部丢失，可能读到其他商户的数据 | `query.go:241` AsMongoFilter | 合并成 `bson.D{ {"$and", [mongoFilters, inFilters]} }`；商户条件必须始终加上；补跨租户测试 | S |
| 2 | 高 | **公开仓库里硬编码了默认 AES 密钥** `HuangLiJi2026_SecretKey_32Bytes!`；密钥长度不对时会**悄悄退回**这个默认值；`init()` 执行时 viper 还没加载配置，所以实际一定用默认密钥，除非业务方调用了 `SetDataEncryptKey` | `secret.go:17,28` | 删掉默认密钥，未设置时加解密直接报错；启动时强制调用 `SetDataEncryptKey`；**评估已加密数据是否需要用新密钥重新加密**（这个密钥已经公开） | M |
| 3 | 高 | JWT Cookie 设置成 `secure=false, httpOnly=false`，JS 可以读取，出现 XSS 时 token（连同里面的 TOTP 种子）会被窃取 | `login.go:87` | `httpOnly=true`、`secure=true`（本地开发时可配置），并设置 `SameSite=Lax/Strict` | S |
| 4 | 高 | filter 值原样进入 Mongo 查询（default 分支 `bson.E{finalKey, val}`），值可以是 `{"$ne":null}`、`{"$regex":...}` 这类操作符对象，存在 NoSQL 注入（字段受白名单限制，但操作符不受限） | `query.go` AsMongoFilter default 分支 | 值是 map 时拒绝，或只允许白名单操作符 | S |
| 5 | 中 | 排序方向写反：`Order=="ASC"` 时 `OrderType=-1`（降序） | `query.go:117` | ASC→1，DESC→-1；前端如果依赖了当前行为，要同步调整 | S |
| 6 | 中 | 分页计算错误：`Page = total % limit + 1` 算出来的不是页码；limit 取 `end-start`，再靠 `SetLimit(PerPage+1)` 补回 1，这种写法容易误读 | `query.go:88-99`、`AsMongoFilter` SetLimit | `limit = end-start+1`，`page = start/limit + 1`，`SetLimit(limit)`；如果“+1 用于判断是否还有下一页”，要写清注释 | S |
| 7 | 中 | `MakeError` 里的 `requestID.(string)` 是强制断言，RequestID 不是 string 时会 panic；`MakeResponse` 把 `err.Error()` 原样返回给客户端（泄露内部信息） | `error.go`、`render.go` | 安全断言；500 错误对外只返回通用信息和 request_id | S |
| 8 | 中 | `CUSTOM_LOGIN_EXPIRE_TIME` 解析失败时 `Atoi` 返回 0，token 立刻过期；`b64.Encode` 后再 `Decode` jwtKey 是多余操作 | `login.go:29-33,58` | 解析失败时保持默认值并打日志；删除这个多余步骤 | S |
| 9 | 中 | AES 加密、脱敏、哈希和 `util9s/crypto/aes.go` 重复（两处的硬编码密钥还不一样）；手机号哈希是不加盐的 SHA-256，11 位手机号可以穷举还原 | `secret.go` | 统一用 `util9s/crypto`；哈希改成 HMAC-SHA256（独立密钥） | M |
| 10 | 低 | 直接用 logrus，没走 `open4go/log`，日志里没有 trace | `login.go` | 改用 `log.Log(c.Request.Context())` | S |
| 11 | 中 | 依赖：`dgrijalva/jwt-go`（已归档）、x/net v0.39.0、旧版 log 引入的 docker；middle 落后很多（v0.3.25，最新 v0.3.60） | `go.mod` | 迁移到 golang-jwt/v5，升级 middle 和 log | M |
| 12 | 中 | 没有测试，而这里恰好是租户隔离和分页逻辑最集中的地方 | — | 给 `LoadQuery`、`AsMongoFilter` 写表驱动测试 | M |
