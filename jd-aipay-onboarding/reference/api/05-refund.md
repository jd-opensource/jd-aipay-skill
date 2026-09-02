# 退款接口 - refund

> 版本：v1.2 | 更新时间：2026-07-13

### 变更记录

| 版本 | 日期       | 变更内容                                                                 |
| ---- | ---------- | ------------------------------------------------------------------------ |
| v1.2 | 2026-07-13 | 正式对外发布：对齐下游服务商退款接口能力，补齐字段与示例；`originalOutTradeNo` 必填替换 `outTradeNo/tradeNo 二选一` 的历史占位描述；新增 `currency` 必填字段与 `returnParams` 透传字段 |
| v1.1 | 2026-06-18 | 公共请求/响应参数新增必填字段 `agentId`（Agent 身份标识，详见协议总览 4.2 / 4.3） |
| v1.0 | 2026-05-12 | 接口首次发布（TRD 占位预留）                                             |

---

## 1. 接口说明

| 项           | 说明                          |
| ------------ | ----------------------------- |
| 功能         | 商户对已支付成功订单发起退款  |
| 路径         | `/pay-ai-agent/refund`        |
| 方法         | POST                          |
| 通信方式     | HTTPS 同步响应                |
| Content-Type | application/json              |

### 1.1 功能描述

商户对已支付成功的原订单发起退款申请，AI 付服务端对接下游 `SpUnifiedRefundApiResource#refund`。支持：

- **全额退款**：`refundAmount` = 原订单支付金额
- **部分退款**：`refundAmount` < 原订单可退余额
- **多次退款**：同一原订单可拆分多次退款，累计不超过可退余额

退款受理成功后为异步处理，最终结果通过 `queryRefundResult`（见 [06-refund-query.md](06-refund-query.md)）主动查询获取。

### 1.2 幂等设计

- 同一 `refundNo` 多次调用：服务端保证幂等，不重复退款
- 相同 `reqNo` 重复请求直接返回首次处理结果
- 幂等键为 `acqMerchantNo + refundNo`

---

## 2. 业务请求参数

> 以下参数为 `bizContent` 解码后的 JSON 对象字段。

| 参数名           | 编码                 | 必填 | 类型         | 说明                                                                                     |
| ---------------- | -------------------- | ---- | ------------ | ---------------------------------------------------------------------------------------- |
| 收单商户号       | acqMerchantNo        | 是   | String(32)   | 收单商户号。非服务商模式传与外层 merchantNo 相同的值；服务商模式传子商户实际入驻的商户号 |
| 原商户订单号     | originalOutTradeNo   | 是   | String(64)   | 被退款的原支付订单的商户订单号                                                           |
| 商户退款单号     | refundNo             | 是   | String(64)   | 商户侧退款流水号，全局唯一，作为幂等键                                                   |
| 退款金额         | refundAmount         | 是   | Long         | 退款金额，单位：分，大于 0，累计不超过原订单可退余额                                     |
| 货币种类         | currency             | 是   | String(8)    | 固定值：`CNY`                                                                            |
| 退款原因         | refundReason         | 否   | String(256)  | 退款原因描述，透传给下游作为退款主题                                                     |
| 退款异步通知地址 | refundNotifyUrl      | 否   | String(256)  | 退款最终结果异步通知 URL（后续版本启用；本版本请使用 `queryRefundResult` 主动查询）      |
| 回传信息         | returnParams         | 否   | String(500)  | 商户自定义回传信息，退款查询接口原样返回                                                 |

> 注：本接口的 `originalOutTradeNo` 是**原支付订单**的商户订单号（对应 `createOrder` 传入的 `outTradeNo`），`refundNo` 是**本次退款**的商户流水号，两者用途不同，请勿混淆。

---

## 3. 业务响应参数

> 以下参数为响应 `bizContent` 解码后的 JSON 对象字段。`resultCode` 和 `resultDesc` 已提升至公共响应层（见协议总览 4.3），bizContent 内不再包含。
>
> **注意**：仅当公共响应层 `resultCode = SUCCESS` 时，响应中才包含 `sign`、`bizContent` 字段。其他 `resultCode` 属于前置校验失败，响应中无 `sign` 和 `bizContent`，商户无需验签。

| 参数名           | 编码                | 必填 | 类型        | 说明                                                          |
| ---------------- | ------------------- | ---- | ----------- | ------------------------------------------------------------- |
| 收单商户号       | acqMerchantNo       | 是   | String(32)  | 与请求一致                                                    |
| 原商户订单号     | originalOutTradeNo  | 是   | String(64)  | 与请求一致                                                    |
| 商户退款单号     | refundNo            | 是   | String(64)  | 与请求一致                                                    |
| 京东退款流水号   | refundTradeNo       | 是   | String(64)  | 京东侧退款交易流水号，用于查询与对账                          |
| 退款金额         | refundAmount        | 是   | Long        | 退款金额，单位：分                                            |
| 货币种类         | currency            | 是   | String(8)   | 与请求一致                                                    |
| 退款状态         | refundStatus        | 是   | String(16)  | 退款状态码，见 3.1                                            |
| 退款状态描述     | refundStatusDesc    | 是   | String(64)  | 退款状态描述                                                  |
| 退款完成时间     | refundFinishTime    | 否   | String(14)  | 退款完成时间，格式：yyyyMMddHHmmss，仅 `REFUND_SUCCESS` 时返回 |
| 回传信息         | returnParams        | 否   | String(500) | 商户下单时传入的回传信息，原样返回                            |

### 3.1 refundStatus 枚举值

| 枚举值       | 含义       | 说明                                                            |
| ------------ | ---------- | --------------------------------------------------------------- |
| `PROCESSING` | 退款处理中 | 非终态，退款已受理，下游处理中，请稍后通过 `queryRefundResult` 查询 |
| `SUCCESS`    | 退款成功   | 终态，退款已完成                                                |
| `FAIL`       | 退款失败   | 终态，退款处理失败，可读取 `resultDesc` 获取原因                |

> **调用建议**：同步响应绝大多数为 `REFUND_PROCESSING`（受理成功），最终成功/失败状态请通过 `queryRefundResult` 轮询获取。轮询间隔建议 3~5 秒，最长 5 分钟。

---

## 4. 完整调用示例

### 4.1 请求示例

**公共请求参数（data.content 内）：**

```json
{
  "appId": "AI_PAY_001",
  "merchantNo": "220000000001",
  "agentId": "AGENT_ROKID_001",
  "reqNo": "REFUND_REQ20260713120000001",
  "timestamp": 1721278800000,
  "nonce": "r1e2f3u4n5d6",
  "version": "1.0",
  "signType": "SHA256",
  "sign": "c3d4e5f6a7b8...",
  "bizContent": "eyJhY3FNZXJjaGFudE5vIjoiMjIwMDAwMDAwMDAxIiwib3JpZ2luYWxPdXRUcmFkZU5vIjoiTUVSMjAyNjA1MTIwMDEiLCJyZWZ1bmRObyI6IlJFRlVORDIwMjYwNzEzMDAxIiwicmVmdW5kQW1vdW50Ijo5OTAwLCJjdXJyZW5jeSI6IkNOWSIsInJlZnVuZFJlYXNvbiI6IueUqOaIt+eUs+ivt+mAgOasvSJ9"
}
```

**bizContent 解码后：**

```json
{
  "acqMerchantNo": "220000000001",
  "originalOutTradeNo": "MER20260512001",
  "refundNo": "REFUND20260713001",
  "refundAmount": 9900,
  "currency": "CNY",
  "refundReason": "用户申请退款",
  "returnParams": "custom_biz_tag_xyz"
}
```

### 4.2 响应示例（退款受理成功）

**公共响应层（data.content JSON.parse 后）：**

```json
{
  "appId": "AI_PAY_001",
  "merchantNo": "220000000001",
  "agentId": "AGENT_ROKID_001",
  "reqNo": "REFUND_REQ20260713120000001",
  "resultCode": "SUCCESS",
  "resultDesc": "退款受理成功",
  "timestamp": 1721278800123,
  "signType": "SHA256",
  "sign": "c3d4e5f6a7b8...",
  "bizContent": "eyJhY3FNZXJjaGFudE5vIjoiMjIwMDAwMDAwMDAxIiwib3JpZ2luYWxPdXRUcmFkZU5vIjoiTUVSMjAyNjA1MTIwMDEiLCJyZWZ1bmRObyI6IlJFRlVORDIwMjYwNzEzMDAxIiwicmVmdW5kVHJhZGVObyI6IkpEUkYyMDI2MDcxMzAwMDAwMSIsInJlZnVuZEFtb3VudCI6OTkwMCwiY3VycmVuY3kiOiJDTlkiLCJyZWZ1bmRTdGF0dXMiOiJSRUZVTkRfUFJPQ0VTU0lORyIsInJlZnVuZFN0YXR1c0Rlc2MiOiLpgIDmrL7lpITnkIbkuK0ifQ=="
}
```

**bizContent 解码后：**

```json
{
  "acqMerchantNo": "220000000001",
  "originalOutTradeNo": "MER20260512001",
  "refundNo": "REFUND20260713001",
  "refundTradeNo": "JDRF20260713000001",
  "refundAmount": 9900,
  "currency": "CNY",
  "refundStatus": "REFUND_PROCESSING",
  "refundStatusDesc": "退款处理中",
  "returnParams": "custom_biz_tag_xyz"
}
```

### 4.3 响应示例（退款直接成功）

**bizContent 解码后：**

```json
{
  "acqMerchantNo": "220000000001",
  "originalOutTradeNo": "MER20260512001",
  "refundNo": "REFUND20260713001",
  "refundTradeNo": "JDRF20260713000001",
  "refundAmount": 9900,
  "currency": "CNY",
  "refundStatus": "REFUND_SUCCESS",
  "refundStatusDesc": "退款成功",
  "refundFinishTime": "20260713120005"
}
```

### 4.4 响应示例（退款金额超限，无签名无 bizContent）

**公共响应层（data.content JSON.parse 后）：**

```json
{
  "appId": "AI_PAY_001",
  "merchantNo": "220000000001",
  "agentId": "AGENT_ROKID_001",
  "reqNo": "REFUND_REQ20260713120000002",
  "resultCode": "REFUND_AMOUNT_EXCEED",
  "resultDesc": "退款金额超过原订单可退余额",
  "timestamp": 1721278800456
}
```

> 前置校验失败时，响应中无 `sign`、`signType`、`bizContent` 字段，商户无需验签。

---

## 5. 注意事项

1. **原订单前置条件**：原订单必须处于 `SUCCESS`（支付成功）状态，其他状态（`PAYING` / `FAIL` / `CLOSED`）不允许退款。
2. **金额校验**：`refundAmount` 必须大于 0，且累计退款金额（含已成功的历史退款）不得超过原订单支付金额。
3. **幂等设计**：同一 `refundNo` 多次调用返回同一结果，不会重复退款；`refundNo` 全局唯一，商户须自行保证。
4. **状态查询**：同步响应仅代表**受理结果**，最终退款结果请通过 `queryRefundResult` 主动查询。
5. **异步通知**：`refundNotifyUrl` 字段为后续版本预留，当前版本请使用主动查询获取最终结果。
6. **多次退款**：同一原订单支持多次部分退款，每次退款使用不同的 `refundNo`。

---

## 6. 结果码

见 [09-error-codes.md](09-error-codes.md)：第 2 章「公共错误码」通用；第 3.3 节「refund 业务错误码」为本接口专有。
