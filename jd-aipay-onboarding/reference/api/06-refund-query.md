# 退款结果查询接口 - queryRefundResult

> 版本：v1.0 | 更新时间：2026-07-13

### 变更记录

| 版本 | 日期       | 变更内容 |
| ---- | ---------- | -------- |
| v1.0 | 2026-07-13 | 初始版本，对接下游 `SpUnifiedRefundApiResource#refundQuery` |

---

## 1. 接口说明

| 项           | 说明                              |
| ------------ | --------------------------------- |
| 功能         | 查询退款交易结果                  |
| 路径         | `/pay-ai-agent/queryRefundResult` |
| 方法         | POST                              |
| 通信方式     | HTTPS 同步响应                    |
| Content-Type | application/json                  |

### 1.1 功能描述

商户通过 `queryRefundResult` 主动查询退款交易的处理状态，用于：

- `refund` 接口返回 `PROCESSING` 后轮询确认最终退款结果
- 与后续版本的退款异步通知互为兜底，确保商户最终获得退款结果
- 纯查询操作，任意次调用不产生副作用

### 1.2 调用建议

- **轮询间隔**：建议 3~5 秒一次
- **最长轮询时长**：5 分钟（超时后仍未终态，请通过对账文件核实）
- **终态判断**：收到 `SUCCESS` / `FAIL` 后停止轮询

---

## 2. 业务请求参数

> 以下参数为 `bizContent` 解码后的 JSON 对象字段。

| 参数名       | 编码          | 必填 | 类型       | 说明                                                                                     |
| ------------ | ------------- | ---- | ---------- | ---------------------------------------------------------------------------------------- |
| 收单商户号   | acqMerchantNo | 是   | String(32) | 收单商户号。非服务商模式传与外层 merchantNo 相同的值；服务商模式传子商户实际入驻的商户号 |
| 商户退款单号 | refundNo      | 是   | String(64) | `refund` 接口请求时使用的商户退款流水号                                                  |

> 注：本接口通过 `refundNo` 定位唯一的退款交易。原订单可能存在多次退款，因此不支持仅传 `originalOutTradeNo` 查询。

---

## 3. 业务响应参数

> 以下参数为响应 `bizContent` 解码后的 JSON 对象字段。`resultCode` 和 `resultDesc` 已提升至公共响应层（见协议总览 4.3），bizContent 内不再包含。
>
> **注意**：仅当公共响应层 `resultCode = SUCCESS` 时，响应中才包含 `sign`、`bizContent` 字段。其他 `resultCode`（如 `REFUND_NOT_FOUND`、`PARAM_ERROR` 等）属于前置校验失败，响应中无 `sign` 和 `bizContent`，商户无需验签。

| 参数名           | 编码                | 必填 | 类型        | 说明                                                          |
| ---------------- | ------------------- | ---- | ----------- | ------------------------------------------------------------- |
| 收单商户号       | acqMerchantNo       | 是   | String(32)  | 与请求一致                                                    |
| 原商户订单号     | originalOutTradeNo  | 是   | String(64)  | 被退款的原支付订单商户订单号                                  |
| 京东原交易单号   | originalTradeNo     | 是   | String(64)  | 被退款的原支付订单京东侧交易流水号                            |
| 商户退款单号     | refundNo            | 是   | String(64)  | 与请求一致                                                    |
| 京东退款流水号   | refundTradeNo       | 是   | String(64)  | 京东侧退款交易流水号                                          |
| 退款金额         | refundAmount        | 是   | Long        | 退款金额，单位：分                                            |
| 货币种类         | currency            | 是   | String(8)   | 货币类型                                                      |
| 退款状态         | refundStatus        | 是   | String(16)  | 退款状态码，见 3.1                                            |
| 退款状态描述     | refundStatusDesc    | 是   | String(64)  | 退款状态描述                                                  |
| 退款完成时间     | refundFinishTime    | 否   | String(14)  | 退款完成时间，格式：yyyyMMddHHmmss，仅 `REFUND_SUCCESS` 时返回 |
| 优惠金额         | discountAmount      | 否   | Long        | 本次退款关联的优惠金额（单位：分），无优惠时不返回或返回 0    |
| 利息费           | interestFee         | 否   | Long        | 本次退款关联的利息费（单位：分），适用于分期/白条场景         |
| 回传信息         | returnParams        | 否   | String(500) | 商户退款申请时传入的回传信息，原样返回                        |

### 3.1 refundStatus 枚举值

| 枚举值       | 含义       | 说明                                             |
| ------------ | ---------- | ------------------------------------------------ |
| `PROCESSING` | 退款处理中 | 非终态，下游处理中，需继续轮询                   |
| `SUCCESS`    | 退款成功   | 终态，退款已完成                                 |
| `FAIL`       | 退款失败   | 终态，退款处理失败，可读取 `resultDesc` 获取原因 |

---

## 4. 完整调用示例

### 4.1 请求示例

**公共请求参数（data.content 内）：**

```json
{
  "appId": "AI_PAY_001",
  "merchantNo": "220000000001",
  "agentId": "AGENT_ROKID_001",
  "reqNo": "REFUND_QUERY20260713120000001",
  "timestamp": 1721278860000,
  "nonce": "q1u2e3r4y5r6",
  "version": "1.0",
  "signType": "SHA256",
  "sign": "d4e5f6a7b8c9...",
  "bizContent": "eyJhY3FNZXJjaGFudE5vIjoiMjIwMDAwMDAwMDAxIiwicmVmdW5kTm8iOiJSRUZVTkQyMDI2MDcxMzAwMSJ9"
}
```

**bizContent 解码后：**

```json
{
  "acqMerchantNo": "220000000001",
  "refundNo": "REFUND20260713001"
}
```

### 4.2 响应示例（退款成功）

**公共响应层（data.content JSON.parse 后）：**

```json
{
  "appId": "AI_PAY_001",
  "merchantNo": "220000000001",
  "agentId": "AGENT_ROKID_001",
  "reqNo": "REFUND_QUERY20260713120000001",
  "resultCode": "SUCCESS",
  "resultDesc": "查询成功",
  "timestamp": 1721278860123,
  "signType": "SHA256",
  "sign": "e5f6a7b8c9d0...",
  "bizContent": "eyJhY3FNZXJjaGFudE5vIjoiMjIwMDAwMDAwMDAxIiwib3JpZ2luYWxPdXRUcmFkZU5vIjoiTUVSMjAyNjA1MTIwMDEiLCJvcmlnaW5hbFRyYWRlTm8iOiJKRDIwMjYwNTEyMDAwMDAxIiwicmVmdW5kTm8iOiJSRUZVTkQyMDI2MDcxMzAwMSIsInJlZnVuZFRyYWRlTm8iOiJKRFJGMjAyNjA3MTMwMDAwMDEiLCJyZWZ1bmRBbW91bnQiOjk5MDAsImN1cnJlbmN5IjoiQ05ZIiwicmVmdW5kU3RhdHVzIjoiUkVGVU5EX1NVQ0NFU1MiLCJyZWZ1bmRTdGF0dXNEZXNjIjoi6YCA5qy+5oiQ5Yqf5oOFIiwicmVmdW5kRmluaXNoVGltZSI6IjIwMjYwNzEzMTIwMDA1In0="
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
  "refundStatus": "REFUND_SUCCESS",
  "refundStatusDesc": "退款成功",
  "refundFinishTime": "20260713120005"
}
```

### 4.3 响应示例（退款处理中）

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
  "refundStatus": "REFUND_PROCESSING",
  "refundStatusDesc": "退款处理中"
}
```

### 4.4 响应示例（含优惠金额与利息费）

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
  "refundStatus": "REFUND_SUCCESS",
  "refundStatusDesc": "退款成功",
  "refundFinishTime": "20260713120005",
  "discountAmount": 100,
  "interestFee": 50,
  "returnParams": "custom_biz_tag_xyz"
}
```

### 4.5 响应示例（退款单不存在，无签名无 bizContent）

**公共响应层（data.content JSON.parse 后）：**

```json
{
  "appId": "AI_PAY_001",
  "merchantNo": "220000000001",
  "agentId": "AGENT_ROKID_001",
  "reqNo": "REFUND_QUERY20260713120000002",
  "resultCode": "REFUND_NOT_FOUND",
  "resultDesc": "未找到对应退款单",
  "timestamp": 1721278860456
}
```

> 前置校验失败时，响应中无 `sign`、`signType`、`bizContent` 字段，商户无需验签。

---

## 5. 注意事项

1. **幂等安全**：该接口为纯查询操作，多次调用不产生副作用。
2. **与异步通知互补**：后续版本的退款异步通知可能因网络抖动丢失，轮询查询保证商户最终能拿到结果。
3. **验签一致**：与 `refund` 共用同一套签名密钥和验签逻辑。
4. **终态处理**：收到终态（`REFUND_SUCCESS` / `REFUND_FAIL`）后应立即停止轮询，避免无效请求。
5. **查询窗口**：退款单在退款成功后保留 **6 个月**，超期后建议通过对账文件核实。

---

## 6. 结果码

见 [09-error-codes.md](09-error-codes.md)：第 2 章「公共错误码」通用；第 3.4 节「queryRefundResult 业务错误码」为本接口专有。
