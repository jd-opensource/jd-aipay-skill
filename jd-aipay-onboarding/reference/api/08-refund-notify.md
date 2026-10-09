# 退款结果异步通知

> 版本：v1.0 | 更新时间：2026-07-20

---

## 1. 接口说明

| 项 | 说明 |
|----|------|
| 功能 | AI付主动推送**退款成功**结果到商户 refundNotifyUrl |
| 方向 | AI付服务端 → 商户服务端（被动接收） |
| 方法 | POST |
| 通信方式 | HTTPS 同步 |
| Content-Type | application/json |
| 触发范围 | **仅在退款成功时触发**；退款处理中、退款失败**不推送**，商户须通过 `queryRefundResult` 主动查询 |

### 1.1 使用说明

异步通知是指一笔退款到账后，AI付服务端会将该笔退款的结果，沿着商户调用 `refund` 时传入的异步通知地址 `refundNotifyUrl`，通过 POST 请求将退款结果作为参数通知到商户系统。

商户 `refundNotifyUrl` 的响应 HTTP 状态码为 `200` 且响应体 `code=success` 时，AI付判定通知成功并停止重试；返回其他 HTTP 状态码（如 `404`、`500`）、非 200 状态、超时，或响应体 `code != success`，均判定通知失败，AI付会按第 4 章的阶梯策略继续重试，直到达到最大次数为止。

**特别说明**：当前版本**只推送退款成功**（`refundStatus=SUCCESS`）。以下场景**不会**触发通知，商户必须通过轮询 `queryRefundResult` 获取：

- 退款处理中（`PROCESSING`）
- 退款失败（`FAIL`）

本接口与支付结果通知（`04-pay-notify`）在协议结构、签名规则、重试策略上完全一致，仅业务数据（bizContent）不同。

### 1.2 注意事项

- **验签必做**：商户服务端**必须先验签再处理业务**，防止伪造通知导致错误入账
- **快速响应**：收到通知后应尽快返回 HTTP 200 + `code=success`，**不要在回调线程内做耗时业务处理**（账户回冲、通知用户等业务动作请异步做）
- **幂等处理**：网关可能因未收到 `success` 响应而重复通知，商户须按退款单维度（`refundNo` / `refundTradeNo`）幂等处理，避免重复回冲
- **必须搭配轮询**：本通知**只覆盖退款成功场景**，处理中 / 失败必须通过 `queryRefundResult` 主动轮询兜底，不能仅依赖本通知作为唯一的退款结果获取途径
- **URL 与支付通知区分**：**强烈建议**商户将 `refundNotifyUrl`（`refund` 请求传入）与 `notifyUrl`（`createOrder` 请求传入）配置为不同的 URL，便于路由、幂等、监控隔离，详见 6.7 节

> 详细规范（含公网可达要求、URL 复用的兼容处理等）见第 6 章「注意事项」。

---

## 2. 通知请求参数

> AI付服务端向商户 `refundNotifyUrl` 发送的请求体。
> **本接口为 AI付主动发起的 HTTP 请求**（非响应），采用**请求侧参数结构与请求侧签名规则**。
> 为便于商户复用同一套响应处理框架，报文额外携带固定的 `resultCode` / `resultDesc` 作为信封结果码——**当前版本通知仅在退款成功场景发起，因此两者恒为成功值**（`resultCode=SUCCESS`、`resultDesc=成功`）。这两个字段**参与签名**，防止外层结果码被篡改。

### 2.1 公共参数
| 参数名称 | 参数编码 | 是否必填 | 参数类型 | 描述 |
|----------|----------|----------|----------|------|
| 应用ID | appId | 是 | String(32) | AI付分配的应用标识，与商户下单时使用的 appId 一致 |
| 二级商户号 | merchantNo | 是 | String(32) | 商户号，与商户下单时使用的 merchantNo 一致 |
| Agent标识 | agentId | 是 | String(64) | 与商户下单时传入的 agentId 一致 |
| 通知请求号 | reqNo | 是 | String(64) | 本次通知的唯一请求号 |
| 时间戳 | timestamp | 是 | Long | 通知发起时间戳，毫秒级（13 位） |
| 随机字符串 | nonce | 是 | String(32) | 每次请求唯一（含重试），用于防重放 |
| 协议版本 | version | 是 | String(4) | 固定值：`1.0` |
| 签名类型 | signType | 是 | String(8) | `SHA256` 或 `SM3`，与商户签约时约定 |
| 签名 | sign | 是 | String(128) | 按 2.4 节规则生成 |
| 结果码 | resultCode | 是 | String(16) | 通知信封结果码。当前版本通知仅在退款成功场景发起，**恒为** `SUCCESS` |
| 结果描述 | resultDesc | 是 | String(64) | 通知信封结果描述。当前版本**恒为** `成功` |
| 业务数据 | bizContent | 是 | String | 业务参数 JSON 经 Base64 编码后的字符串，见 2.2 |

### 2.2 bizContent 业务参数

> 以下参数为 `bizContent` Base64 解码后的 JSON 对象字段。

| 参数名 | 编码 | 必填 | 类型 | 说明 |
|--------|------|------|------|------|
| 收单商户号 | acqMerchantNo | 是 | String(32) | 收单商户号。非服务商模式与外层 merchantNo 相同；服务商模式为子商户实际入驻的商户号 |
| 原商户订单号 | originalOutTradeNo | 是 | String(64) | 被退款的原支付订单商户订单号 |
| 京东原交易单号 | originalTradeNo | 是 | String(64) | 被退款的原支付订单京东侧交易流水号 |
| 商户退款单号 | refundNo | 是 | String(64) | 商户侧退款流水号（`refund` 请求时传入） |
| 京东退款流水号 | refundTradeNo | 是 | String(64) | 京东侧退款交易流水号 |
| 退款金额 | refundAmount | 是 | Long | 退款金额，单位：分 |
| 货币种类 | currency | 是 | String(8) | 货币类型，如 `CNY` |
| 退款状态 | refundStatus | 是 | String(16) | 固定值：`SUCCESS`；见 2.3 |
| 退款状态描述 | refundStatusDesc | 是 | String(64) | 退款状态描述 |
| 退款完成时间 | refundFinishTime | 是 | String(14) | 退款到账时间，格式：yyyyMMddHHmmss |
| 回传信息 | returnParams | 否 | String(500) | 商户退款申请时传入的回传信息，原样返回 |

### 2.3 refundStatus 枚举值

| 枚举值       | 含义       | 说明                                                  |
| ------------ | ---------- | ----------------------------------------------------- |
| `SUCCESS`    | 退款成功   | 推送本接口                                            |
| `PROCESSING` | 退款处理中 | 不推送本接口，商户通过 `queryRefundResult` 查询获取   |
| `FAIL`       | 退款失败   | 不推送本接口，商户通过 `queryRefundResult` 查询获取   |

> 当前版本推送到商户 `refundNotifyUrl` 的 `refundStatus` 只会是 `SUCCESS`。其他枚举值仅在 `queryRefundResult` 响应中出现；此处列出是为了与 `05-refund` / `06-refund-query` 保持同一份状态码定义，便于商户复用解析代码。

### 2.4 签名规则

本接口签名规则**与协议总览 5.2 节的请求侧签名规则完全一致**（notify 语义上是 AI付发起的请求），也与 `04-pay-notify` § 2.4 一致：

**参与签名的字段**：`appId`、`merchantNo`、`agentId`、`reqNo`、`timestamp`、`nonce`、`version`、`resultCode`、`resultDesc`、`bizContent`（`sign` 与 `signType` 不参与）。

**待签名字符串**（按参数名 ASCII 升序拼接，`key=value` 用 `&` 连接，空值不参与）：

```
appId=AI_PAY_001&agentId=AGENT_ROKID_001&bizContent=eyJhY3FNZXJj...&merchantNo=220000000001&nonce=r1e2f3n4o5t6&reqNo=REFUND_NOTIFY20260713121030001&resultCode=SUCCESS&resultDesc=成功&timestamp=1721278830000&version=1.0
```

签名算法（HMAC-SHA256 / HMAC-SM3）与密钥使用规则见协议总览 5.2 节。**签名密钥与 `refund` / `queryRefundResult` 共用同一套**，商户无需为退款通知单独维护密钥。

**商户侧必须先验签再处理业务**，验签失败一律返回非 `success` 让 AI付重试，或直接丢弃并报警（不要静默吞掉）。

---

## 3. 商户响应要求

### 3.1 响应结构

商户服务端收到通知后，须返回 HTTP 200 状态码，响应体使用 `application/json`，仅需返回以下字段：

| 字段 | 必填 | 类型 | 说明 |
|------|------|------|------|
| code | 是 | String(16) | 处理结果码，见 3.2 |
| msg | 否 | String(128) | 结果描述，便于排查 |

**响应体本身无需签名**（AI付仅通过响应内容判定商户处理结果，不校验商户响应签名）。

### 3.2 处理结果码

| code | 含义 | AI付后续行为 |
|------|------|--------------|
| `success` | 商户已成功接收并处理（含幂等命中） | 停止重试 |
| 其他任意值 / 非 200 / 超时 | 商户未成功处理 | 按 4 章阶梯重试 |

> 商户**不应**返回 `PROCESSING` 之类的"中间态"来阻止重试——重试的成本远低于业务不一致的风险。若确需异步处理，请遵循 6 章"快速响应"要求。

### 3.3 成功响应示例

```json
{
  "code": "success",
  "msg": "处理成功"
}
```

---

## 4. 重试机制

| 项 | 说明 |
|----|------|
| 触发条件 | 商户服务端未返回 HTTP 200，或响应体 `code != success`，或请求超时 |
| 重试策略 | 阶梯间隔：15s → 30s → 1min → 5min → 15min → 30min → 1h → 2h → 6h → 24h |
| 最大重试次数 | 10 次（含首次共 11 次投递） |
| 单次超时 | 5 秒 |
| 总窗口 | 约 35 小时后彻底放弃 |
| reqNo 语义 | **同一笔退款的所有重试共用同一 `reqNo`**；每次重试的 `nonce` 与 `timestamp` 会刷新 |
| nonce 语义 | 每次投递（含重试）生成新的 `nonce`，防止商户侧防重放中间件误拦重试 |

> **商户幂等键推荐使用 `refundNo` 或 `refundTradeNo`**。`reqNo` 在重试期间不变，也可作为幂等键，但退款单维度的键更能兜住"轮询查询 + 通知"两条路径共同触达时的重复处理。

---

## 5. 完整示例

### 5.1 通知请求示例

```json
{
  "appId": "AI_PAY_001",
  "merchantNo": "220000000001",
  "agentId": "AGENT_ROKID_001",
  "reqNo": "REFUND_NOTIFY20260713121030001",
  "timestamp": 1721278830000,
  "nonce": "r1e2f3n4o5t6",
  "version": "1.0",
  "signType": "SHA256",
  "sign": "8b7c6d5e4f3a...",
  "resultCode": "SUCCESS",
  "resultDesc": "成功",
  "bizContent": "eyJhY3FNZXJjaGFudE5vIjoiMjIwMDAwMDAwMDAxIiwib3JpZ2luYWxPdXRUcmFkZU5vIjoiTUVSMjAyNjA1MTIwMDEiLCJvcmlnaW5hbFRyYWRlTm8iOiJKRDIwMjYwNTEyMDAwMDAxIiwicmVmdW5kTm8iOiJSRUZVTkQyMDI2MDcxMzAwMSIsInJlZnVuZFRyYWRlTm8iOiJKRFJGMjAyNjA3MTMwMDAwMDEiLCJyZWZ1bmRBbW91bnQiOjk5MDAsImN1cnJlbmN5IjoiQ05ZIiwicmVmdW5kU3RhdHVzIjoiU1VDQ0VTUyIsInJlZnVuZFN0YXR1c0Rlc2MiOiLpgIDmrL7miJDlip8iLCJyZWZ1bmRGaW5pc2hUaW1lIjoiMjAyNjA3MTMxMjEwMzAifQ=="
}
```

**bizContent 解码后：**

```json
{
  "acqMerchantNo": "220000000001",
  "originalOutTradeNo": "MER20260512001",
  "originalTradeNo": "JD20260512000001",
  "refundNo": "REFUND20260713001",
  "refundTradeNo": "JDRF20260713000001",
  "refundAmount": 9900,
  "currency": "CNY",
  "refundStatus": "SUCCESS",
  "refundStatusDesc": "退款成功",
  "refundFinishTime": "20260713121030",
  "discountAmount": 100,
  "interestFee": 50,
  "returnParams": "custom_biz_tag_xyz"
}
```

### 5.2 商户响应示例

```json
{
  "code": "success",
  "msg": "处理成功"
}
```

---

## 6. 注意事项

1. **refundNotifyUrl 来源**：AI付使用商户在 `refund`（见 `05-refund.md`）请求时传入的 `refundNotifyUrl` 作为推送地址，商户退款申请时未传或传入 `null` 则不发起通知。地址一旦与退款单绑定，重试期间不再变更。**强烈建议 `refundNotifyUrl` 与下单时的 `notifyUrl` 配置为不同的 URL**，详见 6.7 节。
2. **验签必做**：回调报文由 AI付服务端使用签名密钥签名，商户服务端**必须先验签再处理业务**，防止伪造通知导致错误入账。签名规则见 2.4 节。
3. **幂等处理**：网关可能因未收到 `success` 响应而重复通知，商户须按退款单维度（`refundNo` / `refundTradeNo`）幂等——已处理过的退款单直接返回 `success`，不重复变更业务状态（如不重复给用户账户回冲、不重复发短信）。
4. **快速响应**：收到通知后**尽快返回 HTTP 200 + `code=success`**，耗时业务逻辑应异步处理（放入消息队列等），避免超时导致 AI付误判为失败而重复通知。**验签与退款单查表判重可以同步做，业务动作（账户回冲、通知用户等）应异步做**。
5. **公网可达**：`refundNotifyUrl` 必须公网可达，支持 HTTPS POST 请求，推荐 TLS 1.2+。
6. **必须搭配轮询使用**：本通知**只覆盖退款成功场景**，商户必须通过 `queryRefundResult` 主动轮询来获取失败、处理中等状态；不能仅依赖本通知作为唯一的退款结果获取途径。两条路径同时触达时，幂等键（退款单号）保证只处理一次。
7. **与支付通知的 URL 区分**（强烈建议）：本接口只推送退款成功。支付结果（支付成功）通过 `04-pay-notify` 推送。虽然协议层允许商户把 `refundNotifyUrl`（`refund` 请求传入）和 `notifyUrl`（`createOrder` 请求传入）配成同一个 URL，但**强烈建议商户为两类通知配置不同的 URL**，理由：
   - **路由清晰**：接收方无需通过 bizContent 内容再次分辨是支付通知还是退款通知，直接按 URL 分发到不同处理器
   - **权限与幂等隔离**：支付入账和退款回冲通常由不同业务模块处理，用不同 URL 天然隔离了幂等键命名空间（`outTradeNo` vs `refundNo`），避免误判
   - **监控与告警分开**：支付通知与退款通知的量级、SLA、告警阈值通常不同，独立 URL 便于分别观测
   - **风险面收敛**：某一类通知出现攻击或异常流量时，可单独限流或临时下线，不影响另一类

   若商户确因架构限制必须复用同一 URL，接收方**必须**先解析 bizContent 判断字段（存在 `refundNo` / `refundTradeNo` 即为退款通知），再分派处理逻辑；不能仅凭"HTTP 请求路径相同"就走同一套业务分支。

---

## 7. 版本迭代记录

| 版本 | 日期 | 变更内容 |
|------|------|----------|
| v1.0 | 2026-07-20 | 接口首次发布：结构对齐 `04-pay-notify`，仅在退款成功（`refundStatus=SUCCESS`）时推送；bizContent 字段与 `queryRefundResult` 保持同名同义；公共参数包含 `resultCode` / `resultDesc` 信封结果码（恒为 `SUCCESS` / `成功`），参与签名；强烈建议商户将 `refundNotifyUrl` 与 `notifyUrl` 配成不同的 URL |
