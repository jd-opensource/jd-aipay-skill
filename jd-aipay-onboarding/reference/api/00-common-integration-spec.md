# AI付开放接口 - 公共接入规范

> 适用对象：接入京东AI付开放接口的商户/平台方  
> 适用范围：下单、查询、退款、通知等所有开放接口  
> 版本：v1.0

---

## 1. 接入说明

AI付开放接口采用 **HTTPS + JSON** 方式调用。商户请求分为两层：

| 层级 | 说明 | 商户需要关注 |
| --- | --- | --- |
| 网关通信层 | 商户请求先进入京东SSE网关，由网关完成路由和基础鉴权 | 请求 URL、Header、外层请求结构 |
| AI付业务层 | AI付业务参数放在 `data.content` 内，由AI付服务完成验签、解码和业务处理 | 公共参数、`bizContent`、签名、业务字段 |

> **注意**：接口文档中的公共请求参数不是 HTTP 请求体根对象，实际请求时必须放入 `data.content` 中。

---

## 2. 环境与地址

### 2.1 环境地址

**预发环境（公网）**

```text
https://ridepassfront-pre.jd.com/api/
```

用于商户联调测试。

**生产环境**

```text
https://sse.jd.com/api/
```

用于正式线上交易。

### 2.2 通信要求

- 请求方式：`POST`
- 数据格式：`application/json`
- 编码格式：`UTF-8`
- 传输协议：生产环境必须使用 `HTTPS`

### 2.3 接口地址拼接

完整接口地址由“环境基础地址 + 接口路径”组成。以下单接口为例：

```text
https://sse.jd.com/api/pay-ai-agent/createOrder
```

---

## 3. 请求 Header

| 参数名称     | 参数编码        | 是否必填 | 参数类型   | 描述                                                                     |
| ------------ | --------------- | -------- | ---------- | ------------------------------------------------------------------------ |
| 网关应用ID   | app-id          | 是       | String     | 网关侧分配的应用标识，用于网关鉴权和路由。示例：`AI_PAY_GATEWAY_001`     |
| 内容类型     | Content-Type    | 是       | String     | 固定值：`application/json`                                               |
| 网关加密类型 | encrypt-type    | 是       | String     | 网关报文加密方式，当前固定值：`NONE`                                     |
| 登录类型     | login-type      | 是       | String     | `0`：不登录；`1`：Cookie登录；`2`：Token登录                             |
| 来源类型     | source-type     | 是       | String     | `H5`：H5页面；`APP`：APP端；`PC`：PC端                                   |
| 密钥类型     | secret-key-type | 是       | String     | 当前固定值：`1`                                                          |
| 登录态凭证   | cookie          | 否       | String     | 当 `login-type=1` 时必传，示例：`pt_key=***`                             |
| 流式输出标识 | stream-type     | 否       | String     | 是否流式输出，普通接口建议传 `false`                                     |

> Header 中的 `app-id` 是网关侧身份标识；请求体中的 `appId` 是AI付业务侧应用标识。两者不是同一个字段，以实际分配值为准。

---

## 4. 公共请求参数

以下字段位于 `data.content` 内，所有接口通用。

| 参数名称     | 参数编码   | 是否必填 | 参数类型    | 描述                                                                     |
| ------------ | ---------- | -------- | ----------- | ------------------------------------------------------------------------ |
| 应用ID       | appId      | 是       | String(32)  | AI付分配的应用标识                                                       |
| 二级商户号   | merchantNo | 是       | String(32)  | 商户号（12位数字，由京东侧分配）                                         |
| Agent标识    | agentId    | 是       | String(64)  | 调用方 Agent 唯一标识，用于 Agent 身份识别、策略路由、风控与数据统计     |
| 请求唯一标识 | reqNo      | 是       | String(64)  | 商户侧生成，全局唯一                                                     |
| 时间戳       | timestamp  | 是       | Long        | 请求时间戳，毫秒级（13位），有效期±5分钟                                 |
| 随机字符串   | nonce      | 是       | String(32)  | 防重放攻击，每次请求唯一                                                 |
| 协议版本     | version    | 是       | String(4)   | 固定值：`1.0`                                                            |
| 加密类型     | encType    | 是       | String(8)   | 固定值：`SM2`，表示 `bizContent` 为 SM2 证书信封密文                     |
| 签名类型     | signType   | 是       | String(8)   | 固定值：`SM3`                                                            |
| 签名         | sign       | 是       | String(128) | 按第 7 节签名规则生成                                                    |
| 业务数据     | bizContent | 是       | String      | 业务参数的传输载体，内容为业务 JSON 经证书信封处理后的密文               |

---

## 5. 请求体外层结构

商户按第 4 节组装公共请求参数后，需将其整体放入 `data.content` 中，形成最终 HTTP 请求体。

```json
{
  "data": {
    "content": {
      "appId": "AI_PAY_001",
      "merchantNo": "220000000001",
      "agentId": "AGENT_ROKID_001",
      "reqNo": "REQ20260512120000001",
      "timestamp": 1715500800000,
      "nonce": "a1b2c3d4e5f6",
      "version": "1.0",
      "encType": "SM2",
      "signType": "SM3",
      "sign": "3a5b8c9d2e1f4a5b6c7d8e9f...",
      "bizContent": "MIIBnjCCAYagAwIBAgIU...(SM2证书信封密文)..."
    }
  }
}
```

---

## 6. bizContent 生成规则

商户先按具体接口文档组装业务参数 JSON，再使用证书信封生成 `bizContent`。

### 6.1 证书信封模式：encType=SM2

对外商户统一使用证书信封模式，`encType` 固定传 `SM2`。`bizContent` 为业务参数 JSON 经 SM2 证书信封处理后的密文字符串。

![证书信封模式](diagrams/06-certificate-envelope.png)

> 图源文件：[`diagrams/06-certificate-envelope.drawio`](diagrams/06-certificate-envelope.drawio)（用 [drawio](https://app.diagrams.net/) 打开编辑后，导出同名 PNG 到 `diagrams/` 目录即可自动渲染）

业务参数 JSON 示例：

```json
{
  "outTradeNo": "MER20260512001",
  "tradeAmount": 9900,
  "currency": "CNY"
}
```

生成后请求中只传证书信封密文：

```text
bizContent=MIIBnjCCAYagAwIBAgIU...(SM2证书信封密文)...
```

服务端收到请求后，使用京东私钥解密，并使用商户公钥证书验签，得到原始业务参数 JSON。

### 6.2 证书准备

| 证书/密钥 | 持有方 | 用途 |
| --- | --- | --- |
| 商户私钥 | 商户自持 | 生成请求信封签名 |
| 商户公钥证书 | 提交京东 | 京东服务端验证信封签名 |
| 京东公钥证书 | 京东提供给商户 | 商户加密请求业务报文 |
| 京东私钥 | 京东服务端持有 | 京东服务端解密请求业务报文 |

> 商户私钥、证书密码等敏感信息必须由商户自行安全保管，禁止写入前端代码、日志或公开仓库。

---

## 7. 签名规则

### 7.1 签名字段

请求签名使用 `data.content` 内的公共请求参数。

参与签名字段：

```text
appId、merchantNo、agentId、reqNo、timestamp、nonce、version、encType、bizContent
```

不参与签名字段：

```text
sign、signType
```

`encType` 固定传 `SM2`，并参与待签名字符串。

### 7.2 生成步骤

1. 去除 `sign`、`signType` 和空值字段。
2. 按参数名 ASCII 码从小到大排序。
3. 使用 `key=value` 格式拼接，并用 `&` 连接。
4. 使用商户签名密钥 `signKey` 计算 HMAC 签名。
5. 将签名结果转为小写 16 进制字符串，放入 `sign` 字段。

待签名字符串示例：

```text
agentId=AGENT_ROKID_001&appId=AI_PAY_001&bizContent=MIIBnjCCAYagAwIBAgIU...(SM2证书信封密文)...&encType=SM2&merchantNo=220000000001&nonce=a1b2c3d4e5f6&reqNo=REQ20260512120000001&timestamp=1715500800000&version=1.0
```

签名算法：

| signType | 算法 | 说明 |
| --- | --- | --- |
| `SM3` | `HmacSM3` | 对外商户固定使用 |

> 签名密钥由京东通过线下安全渠道分发。签名密钥与证书私钥均禁止写入前端代码、日志或公开仓库。

---

## 8. 响应体结构

网关响应外层结构如下：

```json
{
  "code": "00000",
  "msg": "成功",
  "data": {
    "content": "{\"appId\":\"AI_PAY_001\",\"merchantNo\":\"220000000001\",\"resultCode\":\"SUCCESS\",\"bizContent\":\"...\"}",
    "status": "FINISHED",
    "encryptType": "NONE"
  }
}
```

| 参数 | 类型 | 说明 |
| --- | --- | --- |
| `code` | String | 网关响应码，`00000` 表示网关处理成功 |
| `msg` | String | 网关响应描述 |
| `data.content` | String | AI付业务响应 JSON 字符串，需要先 `JSON.parse` |
| `data.status` | String | 处理状态，通常为 `FINISHED` |
| `data.encryptType` | String | 当前固定值 `NONE` |

> `code=00000` 只表示网关处理成功，不代表AI付业务处理成功。业务是否成功以 `data.content` 解析后的 `resultCode` 为准。

---

## 9. 公共响应参数

商户需要先对 `data.content` 做 `JSON.parse`，得到AI付业务响应对象。

| 参数名       | 参数编码   | 必填 | 类型        | 说明                                                                                 |
| ------------ | ---------- | ---- | ----------- | ------------------------------------------------------------------------------------ |
| 应用ID       | appId      | 是   | String(32)  | 与请求一致                                                                           |
| 二级商户号   | merchantNo | 是   | String(32)  | 与请求一致                                                                           |
| Agent标识    | agentId    | 是   | String(64)  | 与请求一致，用于商户侧确认响应所属 Agent                                             |
| 请求唯一标识 | reqNo      | 是   | String(64)  | 与请求一致                                                                           |
| 业务结果码   | resultCode | 是   | String(16)  | `SUCCESS`：业务处理成功；其他值见错误码表                                            |
| 业务结果描述 | resultDesc | 是   | String(128) | 结果描述信息，失败时返回具体原因                                                     |
| 时间戳       | timestamp  | 是   | Long        | 响应时间戳                                                                           |
| 签名类型     | signType   | 否   | String(8)   | 业务成功时返回，与请求一致                                                           |
| 签名         | sign       | 否   | String(128) | 业务成功时返回，商户须验签                                                           |
| 业务数据     | bizContent | 否   | String      | 业务成功时返回，业务响应 JSON 经 Base64 编码后的字符串（各接口不同，见具体接口文档） |

响应解析规则：

| 场景 | 商户处理方式 |
| --- | --- |
| `resultCode=SUCCESS` | 验证响应签名，通过后按接口文档解析 `bizContent`，读取业务结果 |
| `resultCode!=SUCCESS` | 响应中通常无 `sign`、`signType`、`bizContent`，商户无需验签，直接读取 `resultDesc` 定位问题 |

---

## 10. 商户接入流程

商户调用任一AI付接口时，按以下步骤处理：

![商户接入流程](diagrams/10-merchant-integration.png)

> 图源文件：[`diagrams/10-merchant-integration.drawio`](diagrams/10-merchant-integration.drawio)（用 [drawio](https://app.diagrams.net/) 打开编辑后，导出同名 PNG 到 `diagrams/` 目录即可自动渲染）

1. 按接口文档组装业务参数 JSON。
2. 使用商户私钥和京东公钥证书生成 SM2 证书信封，得到 `bizContent`。
3. 组装公共请求参数：`appId`、`merchantNo`、`agentId`、`reqNo`、`timestamp`、`nonce`、`version`、`encType=SM2`、`signType=SM3`、`bizContent`。
4. 按签名规则生成 `sign`。
5. 将公共请求参数放入 `data.content`，并设置请求 Header。
6. 发起 HTTP `POST` 请求。
7. 判断网关响应 `code` 是否为 `00000`。
8. 对 `data.content` 执行 `JSON.parse`，读取业务 `resultCode`。
9. 如 `resultCode=SUCCESS`，先验签，再按接口文档解析 `bizContent`。
10. 按具体接口文档处理业务结果。

---

## 11. 完整请求示例

以下单接口为例。

```bash
curl -X POST 'https://ridepassfront-pre.jd.com/api/pay-ai-agent/createOrder' \
  -H 'app-id: AI_PAY_GATEWAY_001' \
  -H 'Content-Type: application/json' \
  -H 'encrypt-type: NONE' \
  -H 'login-type: 0' \
  -H 'source-type: H5' \
  -H 'secret-key-type: 1' \
  -H 'stream-type: false' \
  -d '{
    "data": {
      "content": {
        "appId": "AI_PAY_001",
        "merchantNo": "220000000001",
        "agentId": "AGENT_ROKID_001",
        "reqNo": "REQ20260512120000001",
        "timestamp": 1715500800000,
        "nonce": "a1b2c3d4e5f6",
        "version": "1.0",
        "encType": "SM2",
        "signType": "SM3",
        "sign": "3a5b8c9d2e1f4a5b6c7d8e9f...",
        "bizContent": "MIIBnjCCAYagAwIBAgIU...(SM2证书信封密文)..."
      }
    }
  }'
```

---

## 12. 接入注意事项

- `reqNo` 必须全局唯一；重试同一请求时可使用相同 `reqNo`，服务端按幂等处理。
- `nonce` 每次请求必须唯一，禁止重复使用。
- `timestamp` 使用毫秒级13位时间戳，有效期为服务端时间 ±5 分钟。
- `bizContent` 加密前的业务 JSON 字段以各接口文档为准。
- 生成签名前不要对证书信封密文做二次 URL 编码或格式化改写。
- 商户必须校验响应签名后再使用响应 `bizContent` 中的业务数据。
- 商户私钥、证书密码、签名密钥、签名串、token、用户敏感信息不得打印明文日志。
- 生产环境必须使用 `https://sse.jd.com/api/`，不得使用预发地址。
