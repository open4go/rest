# rest — REST 接口公共工具（查询解析、统一响应、登录签发、字段加密）

> 仓库：https://github.com/open4go/rest ｜ 模块：`github.com/open4go/rest` ｜ Go 1.24.0 ｜ 最近提交：2026-08-20（最新 tag v0.2.24）

## 项目简介
给 react-admin 风格的前端（`range=[0,9]`、`sort=["id","ASC"]`、`filter={}`）提供后端工具：把查询参数解析成 Mongo 过滤条件和分页、统一成功/错误响应、`Content-Range` 列表响应、bcrypt 校验密码并签发 Cookie JWT，以及手机号这类敏感字段的 AES-GCM 加密、脱敏和哈希。

## 目录结构
```
rest/
├── query.go    # QueryParams、LoadQuery、AsMongoFilter、ToRange（_gte/_lte/_sec/_hash 后缀）
├── render.go   # MakeResponse（201/200/202/500）
├── error.go    # ErrResp、MakeError
├── list.go     # GetContentRange、RenderForList
├── login.go    # RenderLogin（bcrypt + HS256 JWT Cookie）
├── secret.go   # EncryptData/DecryptData/DataMatch/GetDataSuffix/GetDataSec/GetDataHash/SetDataEncryptKey
└── convert.go  # ToString
```

## 使用示例
```go
q := rest.LoadQuery(c)                         // 从 X-Tenant-ID 读取 MerchantId
filter, opt := q.AsMongoFilter([]string{"name", "id=>_id", "phone_sec"}, q.Filter)
total, err := m.GetListWithOpt(filter, &list, &opt)
rest.RenderForList(c, q.RangeContent(total), total, list)

rest.MakeResponse(c, err, "created", id)      // id 是 string 时返回 201 并写 TargetId 头
rest.MakeError(c, "参数错误", err.Error(), 400)
rest.RenderLogin(c, loginInfo, hashFromDB, req.Password, jwtKey, host, roles, toolbar)
```
租户隔离：`MerchantId` 非空时追加条件 `_.meta.merchant_id ∈ [MerchantId, "share"]`。

## 依赖
`gin v1.10.0`、`dgrijalva/jwt-go`、`x/crypto/bcrypt`、`mongo-driver v1.17.3`、`logrus`、`open4go/{log v0.0.16, middle v0.3.25}` 等

## 构建
`go build` 通过；`go vet` 有 1 条提示：`query.go:237` 的 `primitive.E` 用了无键字段。没有测试。

## 与其他仓库的关系
依赖 `middle`（`LoginInfo`/`DumpLoginInfo`），签发的 Cookie 由 `middle.JWTMiddleware` 校验；加密实现和 `util9s/crypto` 重复；过滤路径 `_.meta.merchant_id` 依赖 `auth` 这类模型的 `bson:"_"` 嵌入约定。
