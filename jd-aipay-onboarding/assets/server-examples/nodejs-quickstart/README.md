# AI 付网关 Node.js 服务端示例

京东 AI 付外场接口 Node.js 版可运行服务端示例，与 Java 示例（`wyaks-security` + BC）、Python 示例（gmssl + asn1crypto）**加密协议兼容**：SM2 数字信封 + CMS SignedData(SM3WithSM2, directSignature=true) + HMAC-SM3 外层签名。

## 快速开始

```bash
# 1. 安装依赖（sm-crypto，全 npm 公网可装）
npm install

# 2. 运行示例（由 render_server_example.sh 生成时，占位符已替换为用户提供的环境参数）
node src/create_order_gateway_demo.js
```

## 依赖说明

- **sm-crypto** — 提供 SM2 加密/签名、SM3、SM4-CBC 基元
- **ASN.1 DER / PKCS12 / RC2-40 / PBE 派生**：均由 `src/utils/der.js` 和 `src/utils/pfx.js` 手工实现（Node 内建 crypto 不含 RC2；主流库如 node-forge / pkijs 不支持 SM2 曲线）

要求 Node.js **>= 16**。

## 目录结构

```
nodejs/
├── package.json                                   # sm-crypto 依赖
├── README.md
└── src/
    ├── create_order_gateway_demo.js
    ├── query_pay_result_gateway_demo.js
    ├── refund_gateway_demo.js
    ├── query_refund_result_gateway_demo.js
    └── utils/
        ├── der.js         # 极简 DER 编解码（SEQUENCE / SET / OID / INTEGER / OCTET STRING / CONTEXT tag）
        ├── pfx.js         # PKCS12 PFX 解析（支持 3DES / RC2-40 / PBES2 三种 PBE）
        ├── crypto.js      # SM2 数字信封 + CMS SM3WithSM2 签名（兼容 wyaks-security）
        └── common.js      # HMAC-SM3、content 组装、HTTP 调用、响应解密
```

## 加密协议要点

- **bizContent**: `encType=SM2` → 京东私有 SM2 数字信封（`SM2EnvelopUtil2`）：`version(1B=0x01) || DER(SEQ(c11,c12,c3,c2)) || IV(16B) || SM4-CBC-PKCS7(P7Sign, key, iv)`
- **P7 签名**: CMS SignedData(SM3WithSM2, `directSignature=true`, `attachFlag=true`)。签名前 `e = SM3(ZA || M)`，其中 `ZA = SM3(ENTLA || "1234567812345678" || a || b || Gx || Gy || Px || Py)`——本示例委托 sm-crypto `doSignature({ hash:true, der:true, userId })` 完成
- **外层签名**: `signType=SM3` → HMAC-SM3(k1=v1&k2=v2&..., SECRET_KEY)，字段按 ASCII 升序排列，`sign/signType` 不参与，空值不参与
- **响应解密**: 若响应 `encType=SM2`，用商户私钥解信封 → 剥 CMS `encapContentInfo.eContent`

## HTTP header 大小写陷阱（重要）

网关对 HTTP header 名**大小写敏感**：只识别全小写的 `stream-type` / `app-id` / `encrypt-type` / `source-type` / `login-type` / `cache-control` / `content-type`。若把 `stream-type` 误发成 `Stream-Type`，网关识别不到 `stream-type=false`，会当成 SSE 流式请求处理，走另一条鉴权链路，返回 `10000403 资源受权信息缺失`。

本示例直接用 Node 内建 `http` / `https` 模块，通过 `options.headers` 传对象——Node 会保留 key 大小写。**切勿**改用 axios / undici 默认代理 / superagent 等封装库，它们会规范化 header 名。
