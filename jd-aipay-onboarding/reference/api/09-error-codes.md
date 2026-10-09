# 错误码与附录

> 版本：v1.5 | 更新时间：2026-09-14
>
> **【v1.5 变更说明 2026-09-14】**createOrder 业务错误码新增 `NEED_CASHIER_PAY`（见 3.1，响应包含 bizContent）。

### 变更记录

| 版本 | 日期       | 变更内容                                                                 |
| ---- | ---------- | ------------------------------------------------------------------------ |
| v1.5 | 2026-09-14 | createOrder 业务错误码新增 `NEED_CASHIER_PAY`（已开通已绑定但当笔超额或风控不通过，降级外单 tradeType=GEN 扫码支付；响应包含 bizContent，含 cashierUrl/guideText） |
| v1.4 | 2026-07-22 | 配合协议总览 v1.5（SM2 证书数字信封）：公共错误码新增 `ENVELOPE_DECRYPT_FAIL`、`CERT_INVALID` |
| v1.3 | 2026-07-20 | 按实际代码归纳错误码：拆分为公共错误码与业务错误码，仅保留代码中实际返回的枚举值；文档编号由 07 调整为 09（置于 `08-refund-notify` 之后） |
| v1.2 | 2026-07-13 | 新增 `queryRefundResult` 业务结果码；扩充 `refund` 业务结果码（对齐下游能力） |
| v1.1 | 2026-05-14 | 补充业务结果码 bizContent 携带规则                                       |
| v1.0 | 2026-05-12 | 初始版本                                                                 |

---

## 1. 网关响应码

> 网关响应码通过响应体外层 `code` 字段返回，表示网关层面的处理结果。

| 响应码 | 描述 |
|--------|------|
| `00000` | 成功：网关处理成功，继续解析 `data.content` |
| 其他 | 失败：网关异常，读取 `msg` 获取原因 |

---

## 2. 公共错误码

> 公共错误码适用于**所有** OpenAPI 接口（`createOrder` / `queryPayResult` / `refund` / `queryRefundResult`），通过公共响应层的 `resultCode` 字段返回。这些码由公共验签切面 `OpenApiSignAspect`、Controller 参数校验以及各 Service 的通用分支产生。
>
> **bizContent 规则**：
> - `resultCode = SUCCESS` 时，响应包含 `sign` / `signType` / `bizContent`，商户须先验签再解码 bizContent
> - 其他公共错误码属于前置校验或系统异常，响应**不包含** `sign` / `signType` / `bizContent`，商户直接读取 `resultDesc` 获取原因

| 响应码 | 说明 | 解决方案 |
|--------|------|----------|
| `SUCCESS` | 成功 | 正常解析业务响应 |
| `PARAM_ERROR` | 请求参数错误 | 对照接口文档检查必填字段、格式与 `bizContent` 编码 |
| `SIGN_KEY_FAIL` | 未查询到商户签名密钥 | 联系京东技术支持确认商户密钥配置 |
| `SIGN_INVALID` | 签名校验不通过 | 检查签名算法、密钥与参与签名的字段是否正确 |
| `ENVELOPE_DECRYPT_FAIL` | 信封解密或信封签名验签失败（`encType=SM2` 时） | 检查信封使用的商户私钥/京东公钥证书是否匹配、证书是否过期 |
| `CERT_INVALID` | 证书无效、过期或未登记（`encType=SM2` 时） | 联系京东技术支持确认商户公钥证书登记状态与有效期 |
| `SYSTEM_ERROR` | 系统异常 | 稍后重试；持续出现请联系京东技术支持 |

---

## 3. 业务错误码

> 业务错误码为**接口专有**的失败场景，与第 2 章公共错误码互斥。除 `NOT_AUTHORIZED`、`NEED_CASHIER_PAY` 外，返回时响应均**不包含** `sign` / `signType` / `bizContent`。

### 3.1 createOrder 业务错误码

| 响应码 | 说明               | 解决方案 |
|--------|------------------|----------|
| `NOT_AUTHORIZED` | 用户未绑定京东账号或未开通AI付 | **响应包含 bizContent**，验签后解码，引导用户按其中的授权链接完成绑定 |
| `NEED_CASHIER_PAY` | 已开通已绑定，但当笔超过免密额度或风控不通过 | **响应包含 bizContent**，验签后解码，业务侧将 `cashierUrl` 渲染为二维码，用户京东App扫码在外单收银台完成支付；支付结果经异步通知推送 |
| `CREATE_ORDER_EXCEPTION` | 下单异常          | 稍后重试；持续出现请联系京东技术支持 |

### 3.2 queryPayResult 业务错误码

| 响应码 | 说明 | 解决方案 |
|--------|------|----------|
| `ORDER_NOT_FOUND` | 未找到对应订单 | 核对 `outTradeNo` 是否为已成功创建的商户订单号 |
| `ORDER_QUERY_FAIL` | 订单查询失败 | 稍后重试；持续出现请联系京东技术支持 |

> 说明：`ORDER_NOT_FOUND` 仅在下游明确返回订单不存在（`JMPT100029`）时返回；下游返回其它业务失败（非成功且非订单不存在）时返回 `ORDER_QUERY_FAIL`；仅当本方 RPC 传输异常或系统级失败时才归为公共错误码 `SYSTEM_ERROR`。

### 3.3 refund 业务错误码

| 响应码 | 说明 | 解决方案 |
|--------|------|----------|
| `REFUND_ORIGINAL_QUERY_FAIL` | 原订单查询失败 | 稍后重试；持续出现请联系京东技术支持 |
| `REFUND_ORIGINAL_NOT_FOUND` | 原订单不存在 | 核对 `originalOutTradeNo` 是否为已支付成功的商户订单号 |
| `REFUND_AMOUNT_EXCEED_ORIGINAL` | 退款金额超过原订单金额 | 核对 `refundAmount`（单位：分）是否超过原订单支付金额 |

### 3.4 queryRefundResult 业务错误码

| 响应码 | 说明 | 解决方案 |
|--------|------|----------|
| `REFUND_NOT_FOUND` | 退款单不存在 | 核对 `refundNo` 是否为已成功发起过的退款单号 |

---

## 4. 附录

### 4.1 GoodsInfo 商品信息结构

> 用于 `createOrder` 接口的 `goodsInfo` 字段，以 `List<GoodsInfo>` 的 JSON 格式提交。

| 参数名 | 编码 | 必填 | 类型 | 说明 |
|--------|------|------|------|------|
| 商品ID | id | 是 | String(32) | 商品唯一标识 |
| 商品名称 | name | 是 | String(128) | 商品名称 |
| 商品单价 | price | 是 | Long | 商品单价，单位：分 |
| 商品数量 | num | 是 | String(8) | 商品数量 |
| 商品类型 | type | 否 | String(32) | 商品类型 |
| 一级类目 | cat1 | 否 | String(32) | 商品一级类目 |
| 二级类目 | cat2 | 否 | String(32) | 商品二级类目 |
| 三级类目 | cat3 | 否 | String(32) | 商品三级类目 |

**示例：**

```json
[
  {
    "id": "GOODS001",
    "name": "Rokid Max Pro 智能眼镜",
    "price": 399900,
    "num": "1",
    "type": "SMART_DEVICE",
    "cat1": "智能硬件"
  }
]
```

### 4.2 deviceInfo AI设备信息结构

> 用于 `createOrder` 接口的 `deviceInfo` 字段。

| 参数名 | 编码 | 必填 | 类型 | 说明 |
|--------|------|------|------|------|
| 设备类型 | deviceType | 是 | String(32) | 见设备类型枚举 |
| 设备ID | deviceId | 否 | String(64) | 设备唯一标识 |
| 设备品牌 | deviceBrand | 否 | String(32) | 设备品牌 |
| 设备型号 | deviceModel | 否 | String(32) | 设备具体型号 |
| 操作系统 | os | 否 | String(32) | 设备操作系统 |

**设备类型枚举：**

| deviceType | 说明 |
|-----------|------|
| `SMART_GLASSES` | 智能眼镜 |
| `SMART_SPEAKER` | 智能音箱 |
| `SMART_SCREEN` | 智能屏 |
| `SMART_WATCH` | 智能手表 |
| `SMART_CAR` | 智能车机 |
| `OTHER` | 其他设备 |

**示例：**

```json
{
  "deviceType": "SMART_GLASSES",
  "deviceId": "ROKID_DEVICE_ABC123",
  "deviceBrand": "Rokid",
  "deviceModel": "Max Pro",
  "os": "YodaOS"
}
```

### 4.3 riskInfo 风控信息结构

> 用于 `createOrder` 接口的 `riskInfo` 字段，以 `Map<String,String>` 的 JSON 格式提交。具体字段根据行业不同有所差异，请联系京东侧技术支持获取行业对应的风控字段要求。

### 4.4 客户端类型枚举

| clientType | 说明 |
|-----------|------|
| `AI_DEVICE` | AI 智能设备（眼镜、音箱等） |
| `H5` | H5 网页 |
| `APP` | 原生应用 |
| `PC` | PC 端 |
| `MINI_PROGRAM` | 小程序 |

### 4.5 货币类型枚举

| currency | 说明 |
|---------|------|
| `CNY` | 人民币（当前仅支持） |

---

## 5. 联系方式

如有接口对接问题，请联系京东AI付技术支持团队。
