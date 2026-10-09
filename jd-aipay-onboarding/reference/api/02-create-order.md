# AI下单接口 - createOrder

> 版本：v1.2（草案，待协议串评） | 更新时间：2026-09-14
>
> **【v1.2 变更说明 2026-09-14】** 本次变更（详情见第 6 章迭代记录与对应章节）：① 业务请求参数新增可选 `receiverInfo`（收货人信息），并补充 `goodsInfo` / `riskInfo` 的对象结构说明（见 2.2–2.4）；② bizContent 响应新增 cashierUrl（降级收银台URL）、notAuthType（未开通细分类型）；③ tradeNo / guideText 返回条件扩展；④ 新增 resultCode NEED_CASHIER_PAY（超额或风控不通过，降级外单扫码支付）。

---

## 1. 接口说明

| 项 | 说明 |
|----|------|
| 功能 | AI付统一下单（准入校验 + 授权引导 + 下单一体） |
| 路径 | `/pay-ai-agent/createOrder` |
| 方法 | POST |
| 通信方式 | HTTPS 同步响应 |
| Content-Type | application/json |

### 1.1 功能描述

`createOrder` 是 AI 付场景下的统一下单工具，单次调用内自动完成：

1. **商户准入校验**：验证商户是否支持 AI 付
2. **用户授权状态判断**：查询用户是否已授权绑定京东账号
3. **状态自动路由**：
   - 未授权 → 返回引导文案与开通 URL
   - 已授权 → 完成下单并返回 SDK 唤起参数

### 1.2 幂等设计

- 同一 `outTradeNo` 多次调用：未授权时不创建京东侧订单，已授权时保证只创建一次订单。
- 同一 `reqNo` 重复请求直接返回首次处理结果。

---

## 2. 业务请求参数

> 以下参数为 `bizContent` 解码后的 JSON 对象字段。

> `accessType` 为本次新增必填字段。四个外场请求接口（下单、查询支付结果、退款、退款查询）均需传入该字段；AI付以请求值为准进行 RPC 路由，不反查原订单校验接入类型一致性。

| 参数名 | 编码 | 必填 | 类型 | 说明                                                       |
|--------|------|------|------|----------------------------------------------------------|
| 收单商户号 | acqMerchantNo | 是 | String(32) | 收单商户号。非服务商模式传与外层 merchantNo 相同的值；服务商模式传子商户实际入驻的商户号       |
| 商户接入类型 | accessType | 是 | String(16) | 商户接入类型：`SERVICE_MER` 服务商模式；`COMMON` 普通商户模式。AI付按该字段路由下游接口，字段参与 `bizContent` 签名/加密保护 |
| 商户订单号 | outTradeNo | 是 | String(32) | 商户唯一交易流水号（字母和数字），同一笔订单两次调用须一致                            |
| 用户ID | userId | 是 | String(64) | 商户端/平台端用户唯一标识（如Rokid用户ID）                                |
| 交易金额 | tradeAmount | 是 | Long | 订单金额，单位：分，大于0的整数                                         |
| 货币种类 | currency | 是 | String(8) | 固定值：`CNY`                                                |
| 交易类型 | tradeType | 是 | String(8) | 固定值：`AI_PAY`                                             |
| 订单创建时间 | createDate | 是 | String(14) | 格式：yyyyMMddHHmmss                                        |
| 订单失效时长 | expiryTime | 否 | String(16) | 单位：秒，默认7天，最大90天                                          |
| 交易名称 | tradeSubject | 是 | String(128) | 订单标题/商品名称，展示在用户账单上                                       |
| 交易描述 | tradeRemark | 是 | String(256) | 订单具体描述信息                                                 |
| 异步通知地址 | notifyUrl | 是 | String(256) | 支付完成后异步通知商户支付结果的URL，须公网可达。注：服务商模式下传入服务商的通知URL，由服务商负责通知商户 |
| 同步跳转地址 | pageBackUrl | 否 | String(256) | 支付成功后跳转URL（仅H5场景使用）                                      |
| 客户端类型 | clientType | 否 | String(15) | 客户端标识：`H5` / `APP`                         |
| 商品信息 | goodsInfo | 否 | String(2048) | 商品信息列表，JSON格式，结构见 2.2                                      |
| 收货人信息 | receiverInfo | 否 | String(2048) | 收货人信息，JSON格式，结构见 2.3                              |
| 用户IP | userIp | 否 | String(64) | 用户终端IP，支持IPv4/IPv6                                       |
| 回传信息 | returnParams | 否 | String(500) | 商户自定义回传信息，异步通知和查询接口原样返回                                  |
| 业务类型 | bizTp | 否 | String(6) | 业务类型码，见附录                                                |
| 业务分类码 | categoryCode | 否 | String(16) | 区分同一商户号下不同业务                                             |
| 风控信息 | riskInfo | 否 | String(1024) | 风控要求信息，JSON格式，结构见 2.4                                            |
| 设备信息 | deviceInfo | 否 | String(1024) | AI设备相关信息，JSON格式                                          |
| 扩展参数 | extendParams | 否 | String(500) | 业务扩展参数，JSON格式                                            |

### 2.1 deviceInfo 结构（AI场景扩展）

| 参数名      | 编码            | 必填 | 类型 | 说明         |
|----------|---------------|------|------|------------|
| 账户标识     | deviceAccount | 是 | String(64) | 设备绑定的账户标识  |
| 设备账户     | deviceSn      | 是 | String(64) | 设备账户       |
| 服务商      | vendor        | 是 | String(64) | 服务商标识      |
| 设备类型     | deviceType    | 是 | String(64) | 设备类型       |
| 设备型号     | deviceModel   | 是 | String(64) | 设备具体型号     |
| 绑定设备显示名称 | deviceName    | 是 | String(64) | 绑定设备的显示名称  |
| 设备唯一标识   | deviceId      | 是 | String(64) | 设备唯一标识ID   |

### 2.2 GoodsInfo 结构（商品信息）

`goodsInfo` 为商品信息 JSON 数组序列化后的字符串，单个商品对象结构如下：

| 参数名称 | 参数编码 | 是否必填 | 参数类型 | 描述               |
|------------|----------|----------|------------|--------------------|
| 商品id     | id       | 是       | String(32) | 商品唯一标识     |
| 商品一级类目 | cat1     | 是       | String(32) | 商品所属一级类目 |
| 商品二级类目 | cat2     | 否       | String(32) | 商品所属二级类目 |
| 商品三级类目 | cat3     | 否       | String(32) | 商品所属三级类目 |
| 商品单价     | price    | 是       | String(32) | 商品单价，单位：分，整数大于0 |
| 商品名称     | name     | 是       | String(32) | 商品名称         |
| 商品数量     | num      | 是       | String(32) | 商品数量         |
| 商品类型     | type     | 是       | String(32) | 商品类型         |
| 门店ID      | storeId  | 否       | String(32) | 商品所属门店唯一标识 |

### 2.3 ReceiverInfo 结构（收货人信息）

`receiverInfo` 为收货人信息 JSON 对象序列化后的字符串，结构如下：

| 参数名称   | 参数编码 | 是否必填 | 参数类型   | 描述               |
|------------|----------|----------|------------|--------------------|
| 收货人姓名 | name     | 是       | String(100) | 收货人姓名         |
| 收货地址   | address  | 是       | String(500) | 收货地址           |
| 收货手机号 | mobile   | 否       | String(30)  | 收货人手机号码     |
| 收货人座机 | phone    | 否       | String(20)  | 收货人座机号码     |
| 电子邮箱   | email    | 否       | String(100) | 收货人电子邮箱     |
| 省份ID     | province | 是       | String(30)  | 收货地址所属省份ID |
| 城市ID     | city     | 是       | String(30)  | 收货地址所属城市ID |
| 区县ID     | country  | 是       | String(30)  | 收货地址所属区县ID |

### 2.4 riskInfo 结构（风控信息）

`riskInfo` 为风控信息 JSON 对象序列化后的字符串，字段均选填、原样透传风控系统，结构如下：

| 参数名称 | 参数编码 | 是否必填 | 参数类型 | 描述 |
|------------|----------|----------|----------|------|
| 下单设备号 | orderEid | 否 | String | 需是设备在商户侧唯一标识；用于检验是否存在下单设备聚集 |
| 下单ip | orderIp | 否 | String | ipv4或ipv6，例：180.208.56.7；用于计算是否存在ip位置聚集，判断ip是否存在异常 |
| 下单账号 | orderAccount | 否 | String | 需是用户在商户侧唯一标识；用于根据风险订单，识别对应的风险下单账号 |
| 下单账号绑定手机号 | orderholderMobileNo | 否 | String | 非加密；用于辅助判断下单账号与支付账号用户是否一致 |
| 下单账号注册时间 | acctRegTime | 否 | String | 格式：yyyy-mm-dd hh:mm:ss；用于判断下单账号近期是否有风险操作行为，可用于制定针对性的策略 |
| 下单账号最近一次登录的时间 | recentLogtime | 否 | String | 格式：yyyy-mm-dd hh:mm:ss；用于判断下单账号近期是否有异常行为 |
| 下单环境UA | userAgent | 否 | String | 用户代理环境；例：Mozilla/5.0 (Phone; OpenHarmony 5.0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 |
| 订单生成时间 | orderTime | 否 | String | 格式：2023-01-01 10:00:00 |
| 首次开通AI付的设备号 | openEid | 否 | String | 用户第一次开通京东AI付的设备号；用于和京东域内数据关联 |
| 订单类型 | orderType | 否 | String | 枚举值：加油/停车/洗车等 |
| 服务门店编号 | shopId | 否 | String | 用于识别订单是否异常 |
| 服务门店地址 | shopAdd | 否 | String | 服务门店地址 |


---

## 3. 业务响应参数

> 以下参数为响应 `bizContent` 解码后的 JSON 对象字段。`resultCode` 和 `resultDesc` 已提升至公共响应层（见协议总览 4.3），bizContent 内不再包含。
>
> **注意**：仅当公共响应层 `resultCode = SUCCESS` 时，响应中才包含 `sign`、`bizContent` 字段。其他 `resultCode`（如 `PARAM_ERROR`、`SIGN_INVALID` 等）属于前置校验失败，响应中无 `sign` 和 `bizContent`，商户无需验签，直接读取公共层 `resultDesc` 即可。

### 3.1 bizContent 响应字段

| 参数名 | 编码 | 必填 | 类型 | 返回条件 | 说明 |
|--------|------|------|------|----------|------|
| 商户订单号 | outTradeNo | 是 | String(32) | 所有场景 | 与请求一致 |
| 京东交易单号 | tradeNo | 否 | String(32) | SUCCESS / ORDER_EXIST / NEED_CASHIER_PAY | 京东侧生成的订单号；`NEED_CASHIER_PAY` 时为降级外单（tradeType=GEN）订单号 |
| SDK唤起参数 | aiTokenParam | 否 | String(512) | SUCCESS | 用于唤起AI付SDK的token参数 |
| SDK应用ID | sdkAppId | 否 | String(32) | SUCCESS | SDK唤起所需的appId |
| 引导文案 | guideText | 否 | String(512) | NOT_AUTHORIZED / NOT_SUPPORTED / NEED_CASHIER_PAY | 引导用户授权或场景相关引导文案，可直接展示/播报给用户；降级场景返回"使用京东App扫码支付"类文案 |
| 开通URL | openUrl | 否 | String(256) | NOT_AUTHORIZED | 用户授权页面URL，格式：`JoyGoRokid://open?url=***`，引导用户在手机端打开完成授权 |
| 降级收银台URL | cashierUrl | 否 | String(512) | NEED_CASHIER_PAY | 降级外单收银台 URL，业务侧直接渲染二维码，用户京东App扫码完成支付；URL 原样透传，纳入响应签名保护 |
| 未开通细分类型 | notAuthType | 否 | String(32) | NOT_AUTHORIZED | 准入缺失项细分：`NOT_BOUND` 未绑定账号 / `NOT_OPENED` 未开通AI付；按"绑定＞开通"优先级取首个缺失项，供调用方渲染差异化文案与按钮（如"开通并支付"）；开通流程路由仍以 `openUrl` 拼参为准，本字段仅做展示辅助 |

### 3.2 结果码与响应分支

> 以下 `resultCode` 通过公共响应层返回，不同 resultCode 决定 bizContent 中包含哪些字段。

| resultCode               | 含义       | bizContent 中包含字段                            | 说明 |
|--------------------------|----------|---------------------------------------------|------|
| `SUCCESS`                | 下单成功     | outTradeNo, tradeNo, aiTokenParam, sdkAppId | 已授权，下单成功，返回SDK唤起参数 |
| `NOT_AUTHORIZED`         | 用户未绑定京东账号或未开通AI付 | outTradeNo, guideText, openUrl, notAuthType | 需引导用户按 openUrl 完成授权 |
| `NEED_CASHIER_PAY`       | 已开通已绑定，当笔超额或风控不通过 | outTradeNo, tradeNo, cashierUrl, guideText | 降级外单（tradeType=GEN）下单成功，返回收银台 URL；业务侧渲染二维码，用户京东App扫码在外单收银台完成支付，结果经异步通知推送 |
| `CREATE_ORDER_EXCEPTION` | 下单异常     | **无 bizContent**        |  |
| `SIGN_KEY_FAIL`          | 未查询到商户签名密钥 | **无 bizContent **   |  |
| `SIGN_INVALID`           | 签名校验不通过   | **无 bizContent**     | 签名校验不通过，无需验签 |
| `PARAM_ERROR`            | 请求参数错误     | **无 bizContent**       | 请求参数校验不通过，无需验签 |
| `SYSTEM_ERROR`           | 系统异常     | **无 bizContent**        | 系统内部错误，建议重试，无需验签 |


---

## 4. 完整调用示例

### 4.1 请求示例

```json
{
  "appId": "AI_PAY_001",
  "merchantNo": "220000000001",
  "agentId": "AGENT_ROKID_001",
  "reqNo": "REQ20260512120000001",
  "timestamp": 1715500800000,
  "nonce": "x7y8z9w0",
  "version": "1.0",
  "signType": "SHA256",
  "sign": "3a5b8c9d2e1f4a5b6c7d8e9f...",
  "bizContent": "eyJvdXRUcmFkZU5vIjoiTUVSMjAyNjA1MTIwMDEiLCJ1c2VySWQiOiJST0tJRF9VU0VSXzEyMyIsInRyYWRlQW1vdW50IjoiOTkwMCIsImN1cnJlbmN5IjoiQ05ZIiwidHJhZGVUeXBlIjoiQUlfUEFZIiwiY3JlYXRlRGF0ZSI6IjIwMjYwNTEyMTIwMDAwIiwidHJhZGVTdWJqZWN0Ijoi5pm66IO955y86ZWcLeacjeWKoeiuouWNlSIsInRyYWRlUmVtYXJrIjoi55So5oi36LSt5LmwUm9raWTmnI3liqEiLCJub3RpZnlVcmwiOiJodHRwczovL21lcmNoYW50LmNvbS9ub3RpZnkiLCJjbGllbnRUeXBlIjoiQUlfREVWSUNFIn0="
}
```

**bizContent 解码后：**

```json
{
  "accessType": "COMMON",
  "outTradeNo": "MER20260512001",
  "userId": "ROKID_USER_123",
  "tradeAmount": 9900,
  "currency": "CNY",
  "tradeType": "AI_PAY",
  "createDate": "20260512120000",
  "tradeSubject": "智能眼镜-服务订单",
  "tradeRemark": "用户购买Rokid服务",
  "notifyUrl": "https://merchant.com/notify",
  "clientType": "AI_DEVICE"
}
```

### 4.2 响应示例（下单成功）

**公共响应层（data.content JSON.parse 后）：**

```json
{
  "appId": "AI_PAY_001",
  "merchantNo": "220000000001",
  "agentId": "AGENT_ROKID_001",
  "reqNo": "REQ20260512120000001",
  "resultCode": "SUCCESS",
  "resultDesc": "下单成功",
  "timestamp": 1715500800123,
  "signType": "SHA256",
  "sign": "7f8e9d0c1b2a3e4f...",
  "bizContent": "eyJvdXRUcmFkZU5vIjoiTUVSMjAyNjA1MTIwMDEiLCJ0cmFkZU5vIjoiSkQyMDI2MDUxMjAwMDAwMSIsImFpVG9rZW5QYXJhbSI6InRva2VuX3h4eCIsInNka0FwcElkIjoiYWlwYXlfYXBwXzAwMSJ9"
}
```

**bizContent 解码后：**

```json
{
  "outTradeNo": "MER20260512001",
  "tradeNo": "JD20260512000001",
  "aiTokenParam": "token_xxx",
  "sdkAppId": "aipay_app_001"
}
```

### 4.3 响应示例（未授权）

**公共响应层（data.content JSON.parse 后）：**

```json
{
  "appId": "AI_PAY_001",
  "merchantNo": "220000000001",
  "agentId": "AGENT_ROKID_001",
  "reqNo": "REQ20260512120000001",
  "resultCode": "NOT_AUTHORIZED",
  "resultDesc": "用户尚未授权绑定京东账号",
  "timestamp": 1715500800123,
  "signType": "SHA256",
  "sign": "a2b3c4d5e6f7...",
  "bizContent": "eyJvdXRUcmFkZU5vIjoiTUVSMjAyNjA1MTIwMDEiLCJndWlkZVRleHQiOiLmgqjlsJrmnKrlvIDpgJrkuqzkuJxBSeS7mO..."
}
```

**bizContent 解码后：**

```json
{
  "outTradeNo": "MER20260512001",
  "guideText": "您尚未开通京东AI付，请使用手机扫描二维码或打开链接完成授权绑定，授权后即可使用语音支付。",
  "openUrl": "JoyGoRokid://open?url=***"
}
```

### 4.4 响应示例（降级扫码 NEED_CASHIER_PAY）

**公共响应层（data.content JSON.parse 后）：**

```json
{
  "appId": "AI_PAY_001",
  "merchantNo": "220000000001",
  "agentId": "AGENT_DEMO_001",
  "reqNo": "REQ20260914120000001",
  "resultCode": "NEED_CASHIER_PAY",
  "resultDesc": "本笔订单超过免密额度，请扫码支付",
  "timestamp": 1765728000123,
  "signType": "SM3",
  "sign": "b3c4d5e6f7a8...",
  "bizContent": "eyJvdXRUcmFkZU5vIjoiTUVSMjAyNjA5MTQwMDEiLCJ0cmFkZU5vIjoiSkQyMDI2MDkxNDAwMDAwMSIsImNhc2hpZXJVcmwiOiJodHRwczovL2Nhc2hpZXIuamQuY29tL2V4dGVybmFsL29yZGVyP3RyYWRlTm89SkQyMDI2MDkxNDAwMDAwMSIsImd1aWRlVGV4dCI6IuacrOeslOiuouWNlei2hei/h+WFjeWvhumineW6pu+8jOivt+S9v+eUqOS6rOS4nEFwcOaJq+eggeaUr+S7mCJ9"
}
```

**bizContent 解码后：**

```json
{
  "outTradeNo": "MER20260914001",
  "tradeNo": "JD20260914000001",
  "cashierUrl": "https://cashier.jd.com/external/order?tradeNo=JD20260914000001",
  "guideText": "本笔订单超过免密额度，请使用京东App扫码支付"
}
```

> 业务侧处理：渲染 `cashierUrl` 为二维码 → 用户京东App扫码进入外单收银台支付 → AI付通过异步通知（见 `04-pay-notify.md`）推送支付结果，报文含本次使用的支付工具。

### 4.5 响应示例（参数错误，无签名无bizContent）

**公共响应层（data.content JSON.parse 后）：**

```json
{
  "appId": "AI_PAY_001",
  "reqNo": "REQ20260512120000001",
  "resultCode": "PARAM_ERROR",
  "resultDesc": "merchantNo不能为空",
  "timestamp": 1715500800456
}
```

> 前置校验失败时，响应中无 `sign`、`signType`、`bizContent` 字段，商户无需验签。
```

---

## 5. 调用时序

### 5.1 典型两次调用流程

1. **首次调用**：用户发起购买，商户调用 `createOrder` → 服务端检测用户未授权 → 返回 `NOT_AUTHORIZED` + 引导文案 + 授权URL
2. **用户授权**：用户在手机端完成授权绑定
3. **二次调用**：用户确认支付，商户使用**相同 outTradeNo** 再次调用 `createOrder` → 服务端检测已授权 → 调用支付网关单 → 返回 `SUCCESS` + SDK参数
4. **唤起SDK**：商户使用 `aiTokenParam` + `sdkAppId` 唤起 AI 付 SDK 进入核验/扣款流程

### 5.2 注意事项

- 阶段二和阶段三调用的是**同一个接口**，入参完全一致，服务端根据用户授权状态自动路由不同分支。
- 商户智能体不直接持有京东密钥，由平台方（如Rokid）统一加签。

---

## 6. 版本迭代记录

| 版本 | 日期 | 变更内容 |
|------|------|----------|
| v1.2 | 2026-09-14 | 【草案待串评】bizContent 响应新增可选字段 `cashierUrl`（降级收银台URL，String(512)）、`notAuthType`（未开通细分类型：NOT_BOUND/NOT_OPENED）；`tradeNo`、`guideText` 返回条件扩展至新结果码；新增 resultCode `NEED_CASHIER_PAY`（已开通已绑定但当笔超额或风控不通过，降级外单 tradeType=GEN 扫码支付）；业务请求参数新增可选 `receiverInfo`（收货人信息，String(2048)）；新增 2.2–2.4 `goodsInfo` / `receiverInfo` / `riskInfo` 对象结构说明；新增 4.4 降级扫码响应示例 |
| v1.1 | 2026-06-18 | 公共请求/响应参数新增必填字段 `agentId`（Agent 身份标识），用于 Agent 级别的策略路由、风控与数据统计；详见协议总览 4.2 / 4.3 |
| v1.0 | 2026-05-12 | 接口首次发布 |
