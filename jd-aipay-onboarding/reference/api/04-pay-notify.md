# 支付结果异步通知

> 版本：v1.3 | 更新时间：2026-09-14
>
> **【v1.3 变更说明 2026-09-14】**bizContent 新增可选字段 `payToolName` / `payToolType`（支付工具名称与类型）与 `aiPayLogo`（京东AI付品牌LOGO；评审明确非支付工具LOGO），`payStatus=SUCCESS` 时返回；详见 2.2 与第 7 章迭代记录。

---

## 1. 接口说明

| 项 | 说明 |
|----|------|
| 功能 | AI付主动推送**支付成功**结果到商户 notifyUrl |
| 方向 | AI付服务端 → 商户服务端（被动接收） |
| 方法 | POST |
| 通信方式 | HTTPS 同步 |
| Content-Type | application/json |
| 触发范围 | **仅在支付成功时触发**；支付失败、订单关闭**不推送**，商户须通过 `queryPayResult` 主动查询 |

### 1.1 使用说明

异步通知是指一笔订单支付成功后，AI付服务端会将该笔订单的支付结果，沿着商户调用 `createOrder` 时传入的异步通知地址 `notifyUrl`，通过 POST 请求将支付结果作为参数通知到商户系统。

商户 `notifyUrl` 的响应 HTTP 状态码为 `200` 且响应体 `code=SUCCESS` 时，AI付判定通知成功并停止重试；返回其他 HTTP 状态码（如 `404`、`500`）、非 200 状态、超时，或响应体 `code != SUCCESS`，均判定通知失败，AI付会按第 4 章的阶梯策略继续重试，直到达到最大次数为止。

**特别说明**：当前版本**只推送支付成功**。以下场景**不会**触发通知，商户必须通过轮询 `queryPayResult` 获取：

- 支付失败（用户放弃、余额不足、风控拦截等）
- 订单超时关闭
- 退款结果（走独立退款通知接口）

### 1.2 注意事项

- **验签必做**：商户服务端**必须先验签再处理业务**，防止伪造通知导致错误发货
- **快速响应**：收到通知后应尽快返回 HTTP 200 + `code=success`，**不要在回调线程内做耗时业务处理**（发货、下发权益等业务动作请异步做）
- **幂等处理**：网关可能因未收到 `success` 响应而重复通知，商户须按订单维度（`outTradeNo` / `tradeNo`）幂等处理，避免重复发货
- **必须搭配轮询**：本通知**只覆盖支付成功场景**，失败 / 关单必须通过 `queryPayResult` 主动轮询兜底，不能仅依赖本通知作为唯一的订单结果获取途径
- **退款不走本接口**：本接口只推送支付成功结果，退款结果通过独立的退款通知接口（见 `08-refund-notify.md`）推送

> 详细规范（含 `notifyUrl` 来源、公网可达要求等）见第 6 章「注意事项」。

---

## 2. 通知请求参数

> AI付服务端向商户 `notifyUrl` 发送的请求体。
> **本接口为 AI付主动发起的 HTTP 请求**（非响应），采用**请求侧参数结构与请求侧签名规则**。
> 为便于商户复用同一套响应处理框架，报文额外携带固定的 `resultCode` / `resultDesc` 作为信封结果码——**当前版本通知仅在支付成功场景发起，因此两者恒为成功值**（`resultCode=SUCCESS`、`resultDesc=成功`）。这两个字段**参与签名**，防止外层结果码被篡改。

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
| 结果码 | resultCode | 是 | String(16) | 通知信封结果码。当前版本通知仅在支付成功场景发起，**恒为** `SUCCESS` |
| 结果描述 | resultDesc | 是 | String(64) | 通知信封结果描述。当前版本**恒为** `成功` |
| 业务数据 | bizContent | 是 | String | 业务参数 JSON 经 Base64 编码后的字符串，见 2.2 |

### 2.2 bizContent 业务参数

> 以下参数为 `bizContent` Base64 解码后的 JSON 对象字段。

| 参数名 | 编码 | 必填 | 类型 | 说明 |
|--------|------|------|------|------|
| 收单商户号 | acqMerchantNo | 是 | String(32) | 收单商户号。非服务商模式与外层 merchantNo 相同；服务商模式为子商户实际入驻的商户号 |
| 商户订单号 | outTradeNo | 是 | String(32) | 商户订单号 |
| 京东交易单号 | tradeNo | 是 | String(32) | 京东侧订单号 |
| 支付状态 | payStatus | 是 | String(16) | 支付状态码，见 2.3 |
| 支付状态描述 | payStatusDesc | 是 | String(64) | 支付状态描述 |
| 交易金额 | tradeAmount | 是 | Long | 交易金额，单位：分 |
| 货币种类 | currency | 是 | String(8) | 货币类型，如 `CNY` |
| 支付完成时间 | payTime | 否 | String(14) | 支付成功时间，格式：yyyyMMddHHmmss；`payStatus=SUCCESS` 时必返 |
| 银行交易流水号 | bankSubmitNo | 否 | String(128) | 银行侧交易流水号，由银行返回；部分支付工具（如余额、优惠券支付）无此字段。与 `queryPayResult` 保持同名同义 |
| 支付工具名称 | payToolName | 否 | String(128) | 本次支付实际使用的支付工具名称，如"京东AI付-招商银行储蓄卡"；`payStatus=SUCCESS` 时返回，供商户结果页展示 |
| 支付工具类型 | payToolType | 否 | String(32) | 支付工具类型，如 `DEBIT_CARD` 储蓄卡 / `CREDIT_CARD` 信用卡 / `BT` 白条 / `GREY` 小金库；无法识别类型时不返 |
| 京东AI付品牌LOGO | aiPayLogo | 否 | String(256) | 京东AI付品牌 LOGO 图片 URL，用于结果页展示；评审明确：下发京东AI付品牌LOGO，非支付工具LOGO；无时不返 |
| 优惠金额 | discountAmount | 否 | Long | 本次支付使用的优惠总金额，单位：分。无优惠时不返或返 0。与 `queryPayResult` 保持同名同义 |
| 回传信息 | returnParams | 否 | String(500) | 商户下单时传入的回传信息，原样返回 |

### 2.3 payStatus 枚举值

| 枚举值    | 含义     | 说明                                             |
| --------- | -------- | ------------------------------------------------ |
| `SUCCESS` | 支付成功 | 推送本接口                                       |
| `FAIL`    | 支付失败 | 不推送本接口，商户通过 `queryPayResult` 查询获取 |
| `CLOSED`  | 订单关闭 | 不推送本接口，商户通过 `queryPayResult` 查询获取 |

> 当前版本推送到商户 `notifyUrl` 的 `payStatus` 只会是 `SUCCESS`。`FAIL` / `CLOSED` 枚举值仅在 `queryPayResult` 响应中出现；此处列出是为了与查询接口保持同一份状态码定义，便于商户复用解析代码。
>
> **退款结果不通过本接口通知**。退款完成会通过独立的退款通知接口推送，或由商户主动调用 `queryRefundResult` 查询。

### 2.4 签名规则

本接口签名规则**与协议总览 5.2 节的请求侧签名规则完全一致**（notify 语义上是 AI付发起的请求）：

**参与签名的字段**：`appId`、`merchantNo`、`agentId`、`reqNo`、`timestamp`、`nonce`、`version`、`resultCode`、`resultDesc`、`bizContent`（`sign` 与 `signType` 不参与）。

**待签名字符串**（按参数名 ASCII 升序拼接，`key=value` 用 `&` 连接，空值不参与）：

```
appId=AI_PAY_001&agentId=AGENT_ROKID_001&bizContent=eyJvdXRUcm...&merchantNo=220000000001&nonce=n1o2t3i4f5y6&reqNo=NOTIFY20260512120530001&resultCode=SUCCESS&resultDesc=成功&timestamp=1715500930000&version=1.0
```

签名算法（HMAC-SHA256 / HMAC-SM3）与密钥使用规则见协议总览 5.2 节。

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

| code               | 含义 | AI付后续行为 |
|--------------------|------|--------------|
| `success`          | 商户已成功接收并处理（含幂等命中） | 停止重试 |
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
| reqNo 语义 | **同一笔订单的所有重试共用同一 `reqNo`**；每次重试的 `nonce` 与 `timestamp` 会刷新 |
| nonce 语义 | 每次投递（含重试）生成新的 `nonce`，防止商户侧防重放中间件误拦重试 |

> **商户幂等键推荐使用 `outTradeNo` 或 `tradeNo`**。`reqNo` 在重试期间不变，也可作为幂等键，但订单维度的键更能兜住"轮询查询 + 通知"两条路径共同触达时的重复处理。

---

## 5. 完整示例

### 5.1 通知请求示例

```json
{
  "appId": "AI_PAY_001",
  "merchantNo": "220000000001",
  "agentId": "AGENT_ROKID_001",
  "reqNo": "NOTIFY20260512120530001",
  "timestamp": 1715500930000,
  "nonce": "n1o2t3i4f5y6",
  "version": "1.0",
  "signType": "SHA256",
  "sign": "9a8b7c6d5e4f...",
  "resultCode": "SUCCESS",
  "resultDesc": "成功",
  "bizContent": "eyJvdXRUcmFkZU5vIjoiTUVSMjAyNjA1MTIwMDEiLCJ0cmFkZU5vIjoiSkQyMDI2MDUxMjAwMDAwMSIsInBheVN0YXR1cyI6IlNVQ0NFU1MiLCJ0cmFkZUFtb3VudCI6Ijk5MDAifQ=="
}
```

**bizContent 解码后：**

```json
{
  "acqMerchantNo": "220000000001",
  "outTradeNo": "MER20260512001",
  "tradeNo": "JD20260512000001",
  "payStatus": "SUCCESS",
  "payStatusDesc": "支付成功",
  "tradeAmount": 9900,
  "currency": "CNY",
  "payTime": "20260512120530",
  "bankSubmitNo": "1234567890ABCDEF",
  "payToolName": "京东AI付-招商银行储蓄卡",
  "payToolType": "DEBIT_CARD",
  "aiPayLogo": "https://img.jd.com/aipay/logo.png",
  "discountAmount": 100,
  "returnParams": "orderSource=rokid_glass_v2"
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

1. **notifyUrl 来源**：AI付使用商户在 `createOrder`（见 `02-create-order.md`）时传入的 `notifyUrl` 作为推送地址，商户下单未传或传入 `null` 则不发起通知。地址一旦下单绑定，重试期间不再变更。
2. **验签必做**：回调报文由AI付服务端使用签名密钥签名，商户服务端**必须先验签再处理业务**，防止伪造通知导致错误发货。签名规则见 2.4 节。
3. **幂等处理**：网关可能因未收到 `success` 响应而重复通知，商户须按订单维度（`outTradeNo` / `tradeNo`）幂等——已处理过的订单直接返回 `success`，不重复变更业务状态。
4. **快速响应**：收到通知后**尽快返回 HTTP 200 + `code=success`**，耗时业务逻辑应异步处理（放入消息队列等），避免超时导致 AI付误判为失败而重复通知。**验签与订单查表判重可以同步做，业务动作（发货、下发权益等）应异步做**。
5. **公网可达**：`notifyUrl` 必须公网可达，支持 HTTPS POST 请求，推荐 TLS 1.2+。
6. **必须搭配轮询使用**：本通知**只覆盖支付成功场景**，商户必须通过 `queryPayResult` 主动轮询来获取失败、关闭等终态；不能仅依赖本通知作为唯一的订单结果获取途径。两条路径同时触达时，幂等键（订单号）保证只处理一次。
7. **退款不走本接口**：本接口只推送支付成功结果。退款结果推送使用独立通知接口，退款金额、退款单号等字段不会出现在本接口的 `bizContent` 中。

---

## 7. 版本迭代记录

| 版本 | 日期 | 变更内容 |
|------|------|----------|
| v1.3 | 2026-09-14 | bizContent 新增可选字段 `payToolName`（支付工具名称）/ `payToolType`（支付工具类型枚举）/ `aiPayLogo`（京东AI付品牌LOGO 图片 URL；评审明确非支付工具LOGO），`payStatus=SUCCESS` 时返回本次实际使用的支付工具与品牌LOGO，供商户结果页展示；工具信息取自支付网关回报的支付明细首笔；9.14 评审修正 LOGO 字段语义 |
| v1.2 | 2026-07-20 | 明确签名走请求侧规则（含 nonce/version）；明确 reqNo 重试期间不变、nonce 每次刷新；补齐 bizContent 示例字段；bizContent 新增可选字段 `bankSubmitNo`（银行交易流水号，String(128)）、`discountAmount`（优惠金额，Long，单位分），与 `queryPayResult` 保持一致；明确**当前仅推送支付成功**，失败/关单须走 `queryPayResult` 轮询；明确退款结果不通过本接口通知；公共参数新增 `resultCode` / `resultDesc` 信封结果码（恒为 `SUCCESS` / `成功`），参与签名 |
| v1.1 | 2026-06-18 | 公共参数新增必填字段 `agentId`（Agent 身份标识，详见协议总览 4.2） |
| v1.0 | 2026-05-12 | 接口首次发布 |