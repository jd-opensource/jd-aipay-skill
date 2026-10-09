# AI付准入接口 - queryAiPayAccess

> 版本：v0.1（草案，待协议串评） | 更新时间：2026-09-14
>
> **【v0.1 新增接口 2026-09-14】** 查询用户在当前场景的AI付准入状态（`accessStatus` 1 准入/0 未准入），准入时返回授权管理 H5 URL；仅查询账号绑定与AI付开通状态，不报送风控。


---

## 1. 接口说明

| 项 | 说明 |
|----|------|
| 功能 | AI付准入查询：查询外部场景用户（车机、智能终端等）在当前场景的AI付准入状态；已准入返回场景授权管理 H5 URL，供"AI付管理页"入口展示与跳转 |
| 路径 | `/pay-ai-agent/queryAiPayAccess` |
| 方法 | POST |
| 通信方式 | HTTPS 同步响应 |
| Content-Type | application/json |
| 公共参数/签名 | 与[公共接入规范](00-common-integration-spec.md)完全一致（Header、外层结构、SM2 证书信封、HMAC 签名、防重放） |

### 1.1 功能描述

本接口为**只读查询接口**，单次调用内完成：

1. **账号绑定校验**：查询外部用户（appId + userId）是否已绑定京东账号
2. **AI付开通校验**：查询该用户是否已开通京东AI付
3. **准入状态返回**：
   - 已绑定且已开通 → `accessStatus=1`（准入），并返回当前场景当前用户的授权管理 H5 URL（`manageUrl`），调用方展示"AI付管理"入口
   - 未满足 → `accessStatus=0`（未准入）+ 准入细分类型（`notAuthType`），调用方**不展示管理入口**

### 1.2 接口边界（重要）

| 项 | 说明 |
|----|------|
| 不做支付链路校验 | 不校验可用支付工具（支付时由网银在线底层拦截）、不做当笔免密额度判断，上述能力仅在下单接口（createOrder）四级准入与支付链路中生效 |
| **不报送风控** | 本接口链路不接入风控系统、不产生风控流水 |
| 无资金动作 | 纯只读查询，不创建订单、不发起支付 |
| 结果口径 | 查询成功即返回公共层 `SUCCESS`，用户是否准入以 bizContent 中 `accessStatus` 为准（查询成功 ≠ 准入） |

---

## 2. 业务请求参数

> 以下参数为 `bizContent` 解密后的 JSON 对象字段。公共请求参数（appId、merchantNo、agentId、reqNo、timestamp、nonce、version、encType、signType、sign、bizContent）见公共接入规范，本接口无额外公共参数要求。

| 参数名 | 编码 | 必填 | 类型 | 说明 |
|--------|------|------|------|------|
| 商户接入类型 | accessType | 是 | String(16) | `SERVICE_MER` / `COMMON`，与下单接口同枚举同校验规则；本接口不路由下游，仅保持协议一致性 |
| 收单商户号 | acqMerchantNo | 是 | String(32) | 收单商户号，与下单接口传入值一致；非服务商模式传与外层 merchantNo 相同的值，服务商模式传子商户实际入驻的商户号 |
| 用户ID | userId | 是 | String(64) | 商户端/平台端用户唯一标识（如商户App用户ID），须与下单接口传入值一致 |

---

## 3. 业务响应参数

> 以下参数为响应 `bizContent` 解密后的 JSON 对象字段。`resultCode` 和 `resultDesc` 已提升至公共响应层（见协议总览 4.3），bizContent 内不再包含。
>
> **注意**：仅当公共响应层 `resultCode = SUCCESS` 时，响应中才包含 `sign`、`signType`、`bizContent` 字段；前置校验失败（如 `PARAM_ERROR`、`SIGN_INVALID`）时无 `sign` 和 `bizContent`，商户无需验签。

### 3.1 bizContent 响应字段

| 参数名 | 编码 | 必填 | 类型 | 返回条件 | 说明 |
|--------|------|------|------|----------|------|
| 用户ID | userId | 是 | String(64) | 所有场景 | 与请求一致 |
| AI付准入状态 | accessStatus | 是 | String(1) | 所有场景 | `1`：准入（已绑定账号且已开通AI付）；`0`：未准入 |
| 准入细分类型 | notAuthType | 否 | String(32) | `accessStatus=0` | 未准入原因细分：`NOT_BOUND` 未绑定账号 / `NOT_OPENED` 未开通AI付；按"绑定＞开通"优先级取首个缺失项，仅用于调用方提示文案，不作为开通流程路由依据 |
| 授权管理URL | manageUrl | 否 | String(256) | `accessStatus=1` | 当前场景当前用户的场景授权管理 H5 URL。页面本体由京东侧提供（来源待评审会确认）；AI付服务端从 DUCC 模板（`ai_pay_manage_url_template`）按 appId+outerUserId 拼接生成；京东登录态校验后可查看/解绑授权 |

### 3.2 结果码与响应分支

| resultCode | 含义 | bizContent 中包含字段 | 说明 |
|------------|------|------------------------|------|
| `SUCCESS` | 查询成功 | userId, accessStatus, notAuthType（未准入时）, manageUrl（准入时） | 用户准入状态以 `accessStatus` 为准 |
| `PARAM_ERROR` | 请求参数错误 | **无 bizContent** | 参数校验不通过，无需验签 |
| `SIGN_KEY_FAIL` | 未查询到商户签名密钥 | **无 bizContent** |  |
| `SIGN_INVALID` | 签名校验不通过 | **无 bizContent** | 无需验签 |
| `SYSTEM_ERROR` | 系统异常 | **无 bizContent** | 含开通状态查询异常；**系统异常不等于未准入**，调用方不应按"未准入"处理 |

> **调用方处理约定**：仅当 `resultCode=SUCCESS` 且 `accessStatus=1` 时展示"AI付管理"入口；`resultCode=SYSTEM_ERROR` 时建议下次进入时重试，不得将入口状态缓存为"未准入"。

---

## 4. 完整调用示例

### 4.1 请求示例

```json
{
  "appId": "AI_PAY_001",
  "merchantNo": "220000000001",
  "agentId": "AGENT_DEMO_001",
  "reqNo": "REQ20260913120000001",
  "timestamp": 1765641600000,
  "nonce": "a9b8c7d6e5f4",
  "version": "1.0",
  "signType": "SM3",
  "sign": "3a5b8c9d2e1f4a5b6c7d8e9f...",
  "bizContent": "eyJhY2Nlc3NUeXBlIjoiQ09NTU9OIiwiYWNxTWVyY2hhbnRObyI6IjIyMDAwMDAwMDAwMSIsInVzZXJJZCI6IkpJRE9VX1VTRVJfMTIzIn0="
}
```

**bizContent 解密后：**

```json
{
  "accessType": "COMMON",
  "acqMerchantNo": "220000000001",
  "userId": "DEMO_USER_123"
}
```

### 4.2 响应示例（已准入）

**公共响应层（data.content JSON.parse 后）：**

```json
{
  "appId": "AI_PAY_001",
  "merchantNo": "220000000001",
  "agentId": "AGENT_DEMO_001",
  "reqNo": "REQ20260913120000001",
  "resultCode": "SUCCESS",
  "resultDesc": "查询成功",
  "timestamp": 1765641600123,
  "signType": "SM3",
  "sign": "7f8e9d0c1b2a3e4f...",
  "bizContent": "eyJ1c2VySWQiOiJKSURPVV9VU0VSXzEyMyIsImFjY2Vzc1N0YXR1cyI6IjEiLCJtYW5hZ2VVcmwiOiJodHRwczovL2FpcGF5LmpkLmNvbS9tYW5hZ2U/YXBwSWQ9KioqJm91dGVyVXNlcklkPSoqKiZzY2VuZT1qaWRvdSJ9"
}
```

**bizContent 解密后：**

```json
{
  "userId": "DEMO_USER_123",
  "accessStatus": "1",
  "manageUrl": "https://aipay.jd.com/manage?appId=***&outerUserId=***&scene=demo"
}
```

### 4.3 响应示例（未准入）

**公共响应层 `resultCode=SUCCESS`，bizContent 解密后：**

```json
{
  "userId": "DEMO_USER_123",
  "accessStatus": "0",
  "notAuthType": "NOT_OPENED"
}
```

### 4.4 响应示例（系统异常，无签名无bizContent）

```json
{
  "appId": "AI_PAY_001",
  "reqNo": "REQ20260913120000001",
  "resultCode": "SYSTEM_ERROR",
  "resultDesc": "系统繁忙，请稍后重试",
  "timestamp": 1765641600456
}
```

---

## 5. 调用时序

1. 用户在终端（如车机）进入"AI付管理"入口，接入方 App 调用本接口
2. AI付服务端验签/解密后查询开通状态（账号绑定状态 + AI付开通状态）
3. 已准入（`accessStatus=1`）：返回 `manageUrl`，接入方展示管理入口，用户点击跳转授权管理页（H5 侧做京东登录态校验）
4. 未准入（`accessStatus=0`）：返回 `notAuthType` 细分原因，接入方不展示入口
5. 查询异常：返回 `SYSTEM_ERROR`，接入方不展示入口且不得缓存该结果

---

## 6. 注意事项

### 6.1 安全约定

- `userId`、`accessType` 处于 `bizContent` SM2 证书信封加密与整体签名保护范围内，防止篡改越权查询。
- `manageUrl` 按 appId+userId 维度动态拼接生成，H5 侧必须校验京东登录态与用户一致性，防止构造 URL 查看/解绑他人授权。
- 商户侧不得将 `manageUrl` 持久化存储或转发给第三方，仅限当前用户当前会话使用。

### 6.2 其他约定

- 本接口为查询类接口，天然幂等；`reqNo` 全局唯一、`nonce` 每次唯一、`timestamp` 有效期 ±5 分钟，规则与公共接入规范一致。
- 建议调用方仅在用户进入管理入口时调用，不要高频轮询。
- 授权管理页（manageUrl 对应页面）的车机安卓 UI 适配由接入方/前端负责，AI付服务端仅提供 URL 与准入判断。

---

## 7. 版本迭代记录

| 版本 | 日期 | 变更内容 |
|------|------|----------|
| v0.1 | 2026-09-13 | 接口首次发布（草案，待协议串评）：定名 AI付准入接口 `queryAiPayAccess`，响应改为准入状态 `accessStatus`（1 准入/0 未准入）+ 准入细分类型 + `manageUrl` |
