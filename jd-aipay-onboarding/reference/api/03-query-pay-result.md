# 支付结果查询接口 - queryPayResult

> 版本：v1.3 | 更新时间：2026-07-20

### 变更记录

| 版本 | 日期       | 变更内容                                                                 |
| ---- | ---------- | ------------------------------------------------------------------------ |
| v1.3 | 2026-07-20 | 业务响应参数新增必填字段 `acqMerchantNo`（收单商户号，与 `04-pay-notify` 保持同名同义）；业务响应参数新增可选字段 `bankSubmitNo`（银行交易流水号，String(128)）、`discountAmount`（优惠金额，Long，单位分），两者仅在 `payStatus=SUCCESS` 时可能返回；修正 4.1 请求示例 JSON 语法错误 |
| v1.2 | 2026-06-18 | 公共请求/响应参数新增必填字段 `agentId`（Agent 身份标识，详见协议总览 4.2 / 4.3） |
| v1.1 | 2026-06-16 | 业务响应参数新增 `channelResultCode` / `channelResultDesc`（底层支付渠道结果） |
| v1.0 | 2026-05-12 | 初始版本                                                                 |

---

## 1. 接口说明


| 项           | 说明                           |
| ------------ | ------------------------------ |
| 功能         | 查询订单支付结果               |
| 路径         | `/pay-ai-agent/queryPayResult` |
| 方法         | POST                           |
| 通信方式     | HTTPS 同步响应                 |
| Content-Type | application/json               |

### 1.1 功能描述

商户通过 `queryPayResult` 主动查询订单的支付状态，用于：

- SDK 唤起后轮询确认最终支付结果
- 与异步通知互为兜底，确保商户最终获得支付结果
- 纯查询操作，任意次调用不产生副作用

### 1.2 调用建议

- **轮询间隔**：建议 2~3 秒一次
- **最长轮询时长**：60 秒
- **终态判断**：收到 `SUCCESS` / `FAIL` / `CLOSED` 后停止轮询

---

## 2. 业务请求参数

> 以下参数为 `bizContent` 解码后的 JSON 对象字段。


| 参数名       | 编码          | 必填   | 类型       | 说明                                                                                     |
| ------------ | ------------- | ------ | ---------- | ---------------------------------------------------------------------------------------- |
| 收单商户号   | acqMerchantNo | 是     | String(32) | 收单商户号。非服务商模式传与外层 merchantNo 相同的值；服务商模式传子商户实际入驻的商户号 |
| 商户订单号   | outTradeNo    | 二选一 | String(32) | 商户订单号，与 tradeNo 二选一                                                            |
| 京东交易单号 | tradeNo       | 二选一 | String(32) | 京东侧订单号，与 outTradeNo 二选一                                                       |

> 注：`outTradeNo` 和 `tradeNo` 至少传一个，同时传入时以 `tradeNo` 为准。

---

## 3. 业务响应参数

> 以下参数为响应 `bizContent` 解码后的 JSON 对象字段。`resultCode` 和 `resultDesc` 已提升至公共响应层（见协议总览 4.3），bizContent 内不再包含。
>
> **注意**：仅当公共响应层 `resultCode = SUCCESS` 时，响应中才包含 `sign`、`bizContent` 字段。其他 `resultCode`（如 `ORDER_NOT_FOUND`、`PARAM_ERROR` 等）属于前置校验失败，响应中无 `sign` 和 `bizContent`，商户无需验签。


| 参数名       | 编码          | 必填 | 类型        | 说明                                                 |
| ------------ | ------------- | ---- | ----------- | ---------------------------------------------------- |
| 收单商户号       | acqMerchantNo     | 是   | String(32)  | 收单商户号。非服务商模式与外层 merchantNo 相同；服务商模式为子商户实际入驻的商户号，与 `04-pay-notify` 保持同名同义 |
| 商户订单号       | outTradeNo        | 是   | String(32)  | 商户订单号                                                                  |
| 京东交易单号     | tradeNo           | 是   | String(32)  | 京东侧订单号                                                                |
| 支付状态         | payStatus         | 是   | String(16)  | 支付状态码，见下表                                                          |
| 支付状态描述     | payStatusDesc     | 是   | String(64)  | 支付状态描述                                                                |
| 交易金额         | tradeAmount       | 是   | Long        | 交易金额，单位：分                                                          |
| 货币种类         | currency          | 是   | String(8)   | 货币类型                                                                    |
| 支付完成时间     | payTime           | 否   | String(14)  | 支付成功时间，格式：yyyyMMddHHmmss，仅支付成功时返回                        |
| 银行交易流水号   | bankSubmitNo      | 否   | String(128) | 银行侧交易流水号，由银行返回，仅 `payStatus=SUCCESS` 时可能返回；部分支付工具（如余额、优惠券支付）无此字段。与 `04-pay-notify` 保持同名同义 |
| 优惠金额         | discountAmount    | 否   | Long        | 本次支付使用的优惠总金额，单位：分，仅 `payStatus=SUCCESS` 时可能返回；无优惠时不返或返 0。与 `04-pay-notify` 保持同名同义 |
| 底层渠道结果码   | channelResultCode | 否   | String(64)  | 底层支付渠道返回的结果码（如银行/钱包返回），用于商户定位失败原因，仅非成功态返回 |
| 底层渠道结果描述 | channelResultDesc | 否   | String(256) | 底层支付渠道返回的结果描述，与 `channelResultCode` 一一对应                  |
| 回传信息         | returnParams      | 否   | String(500) | 商户下单时传入的回传信息，原样返回                                          |

### 3.1 payStatus 枚举值

| 枚举值    | 含义     | 说明                                 |
| --------- | -------- | ------------------------------------ |
| `PAYING`  | 支付中   | 非终态，用户正在进行支付，需继续轮询 |
| `SUCCESS` | 支付成功 | 终态，支付已完成                     |
| `FAIL`    | 支付失败 | 终态，支付失败                       |
| `CLOSED`  | 订单关闭 | 终态，订单已关闭（超时未支付等）     |

---

## 4. 完整调用示例

### 4.1 请求示例

```json
{
  "appId": "AI_PAY_001",
  "merchantNo": "220000000001",
  "agentId": "AGENT_ROKID_001",
  "reqNo": "QUERY20260512120000001",
  "timestamp": 1715500860000,
  "nonce": "q1w2e3r4",
  "version": "1.0",
  "signType": "SHA256",
  "sign": "a1b2c3d4e5f6...",
  "bizContent": "eyJvdXRUcmFkZU5vIjoiTUVSMjAyNjA1MTIwMDEifQ=="
}
```

**bizContent 解码后：**

```json
{
  "acqMerchantNo": "220000000001",
  "outTradeNo": "MER20260512001"
}
```

### 4.2 响应示例（支付成功）

**公共响应层（data.content JSON.parse 后）：**

```json
{
  "appId": "AI_PAY_001",
  "merchantNo": "220000000001",
  "agentId": "AGENT_ROKID_001",
  "reqNo": "QUERY20260512120000001",
  "resultCode": "SUCCESS",
  "resultDesc": "查询成功",
  "timestamp": 1715500860123,
  "signType": "SHA256",
  "sign": "f6e5d4c3b2a1...",
  "bizContent": "eyJvdXRUcmFkZU5vIjoiTUVSMjAyNjA1MTIwMDEiLCJ0cmFkZU5vIjoiSkQyMDI2MDUxMjAwMDAwMSIsInBheVN0YXR1cyI6IlNVQ0NFU1MiLC4uLn0="
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
  "discountAmount": 100
}
```

### 4.3 响应示例（支付中）

**公共响应层（data.content JSON.parse 后）：**

```json
{
  "appId": "AI_PAY_001",
  "merchantNo": "220000000001",
  "agentId": "AGENT_ROKID_001",
  "reqNo": "QUERY20260512120000002",
  "resultCode": "SUCCESS",
  "resultDesc": "查询成功",
  "timestamp": 1715500870123,
  "signType": "SHA256",
  "sign": "b1c2d3e4f5a6...",
  "bizContent": "eyJvdXRUcmFkZU5vIjoiTUVSMjAyNjA1MTIwMDEiLCJ0cmFkZU5vIjoiSkQyMDI2MDUxMjAwMDAwMSIsInBheVN0YXR1cyI6IlBBWUlORyIsLi4ufQ=="
}
```

**bizContent 解码后：**

```json
{
  "outTradeNo": "MER20260512001",
  "tradeNo": "JD20260512000001",
  "payStatus": "PAYING",
  "payStatusDesc": "用户支付中",
  "tradeAmount": 9900,
  "currency": "CNY"
}
```

### 4.4 响应示例（支付失败，含底层渠道错误）

**bizContent 解码后：**

```json
{
  "outTradeNo": "MER20260512001",
  "tradeNo": "JD20260512000001",
  "payStatus": "FAIL",
  "payStatusDesc": "支付失败",
  "tradeAmount": 9900,
  "currency": "CNY",
  "channelResultCode": "BALANCE_INSUFFICIENT",
  "channelResultDesc": "账户余额不足"
}
```

### 4.5 响应示例（订单不存在，无签名无bizContent）

**公共响应层（data.content JSON.parse 后）：**

```json
{
  "appId": "AI_PAY_001",
  "merchantNo": "220000000001",
  "agentId": "AGENT_ROKID_001",
  "reqNo": "QUERY20260512120000003",
  "resultCode": "ORDER_NOT_FOUND",
  "resultDesc": "未找到对应订单",
  "timestamp": 1715500880123
}
```

> 前置校验失败时，响应中无 `sign`、`signType`、`bizContent` 字段，商户无需验签。

---

## 5. 注意事项

1. **幂等安全**：该接口为纯查询操作，多次调用不产生副作用。
2. **与异步通知互补**：异步通知可能因网络抖动丢失，轮询查询保证商户最终能拿到结果。
3. **验签一致**：与 `createOrder` 共用同一套签名密钥和验签逻辑。
4. **终态处理**：收到终态（SUCCESS/FAIL/CLOSED）后应立即停止轮询，避免无效请求。
