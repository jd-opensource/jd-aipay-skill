# AI 付网关 Python 服务端示例

京东 AI 付外场接口 Python 版可运行服务端示例，与 Java 示例（`wyaks-security` + BC）**加密协议完全兼容**（已通过双向 wyaks-security 互通测试）。

## 快速开始

```bash
# 1. 建议在 venv 内安装依赖，避免污染系统 Python
python3 -m venv .venv && source .venv/bin/activate
pip install -r requirements.txt

# 2. 运行示例（由 render_server_example.sh 生成时，占位符已替换为用户提供的环境参数）
python -m aipay_demo.create_order_gateway_demo
# 或
python src/aipay_demo/create_order_gateway_demo.py
```

## 依赖说明

- **gmssl** — 提供 SM2 加密/签名与 SM3、SM4-CBC 基元
- **asn1crypto** — 手工解析 PKCS#12 (PFX)、构造 CMS SignedData
- **cryptography** — 提供 3DES/AES 用于解 PKCS#12 加密的 SafeBag（RC2 40-bit 走内置 pure-Python 实现，因为 cryptography 49+ 移除了 ARC2）

## 目录结构

```
python/
├── requirements.txt
├── src/
│   └── aipay_demo/
│       ├── create_order_gateway_demo.py
│       ├── query_pay_result_gateway_demo.py
│       ├── refund_gateway_demo.py
│       ├── query_refund_result_gateway_demo.py
│       └── utils/
│           ├── crypto.py      # SM2 数字信封 + CMS SM3WithSM2 签名（兼容 wyaks-security）
│           ├── pfx.py          # PKCS12 PFX 解析（支持 SM2 私钥/证书）
│           └── common.py       # HMAC-SM3、content 组装、HTTP 调用
└── tests/
    ├── test_roundtrip.py             # 单侧 sign/verify 往返
    ├── interop_python_encrypt.py     # Python 加密 → 让 Java wyaks-security 解
    └── interop_python_decrypt.py     # Java wyaks-security 加密 → Python 解
```

## 加密协议要点

- **bizContent**: `encType=SM2` → 京东私有 SM2 数字信封（`SM2EnvelopUtil2`）：`version(1B=0x01) || DER(SEQ(c11,c12,c3,c2)) || IV(16B) || SM4-CBC-PKCS7(P7Sign, key, iv)`
- **P7 签名**: CMS SignedData(SM3WithSM2, `directSignature=true`, `attachFlag=true`)
- **外层签名**: `signType=SM3` → HMAC-SM3(k1=v1&k2=v2&..., SECRET_KEY)，字段按 ASCII 升序排列，`sign/signType` 不参与，空值不参与
- **响应解密**: 若响应 `encType=SM2`，用商户私钥解信封 → 剥 CMS `encapContentInfo.eContent`
