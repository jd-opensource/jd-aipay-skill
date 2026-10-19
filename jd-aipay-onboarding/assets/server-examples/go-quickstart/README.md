# AI 付服务端示例工程（Go 版）

与 Java / Python / Node.js 版完全等价的 createOrder 等四个网关接口示例。
**零第三方依赖**：SM2/SM3/SM4/数字信封/PFX/CMS 全部为纯 Go 实现（仅用标准库），
`go build` 即可编译，无需拉取任何模块、无需 vendor 目录。

## 目录结构

```
go-quickstart/
├── go.mod                          # module aipay-quickstart（无第三方依赖）
├── aipay/                          # 共享包：国密算法 + 网关协议
│   ├── sm3.go / sm4.go / sm2.go    # 国密算法（GB/T 32905/32907/32918）
│   ├── der.go / cert.go            # 极简 DER 与 X.509 解析（支持国密证书）
│   ├── pkcs12.go / rc2.go          # 商户 PFX 解析（3DES/RC2-40/PBES2）
│   ├── cms.go                      # CMS(P7) SM3WithSM2 签名
│   ├── envelope.go                 # 京东私有 SM2 数字信封
│   ├── common.go                   # HMAC-SM3、content 组装、HTTP、响应解密
│   ├── config.go                   # aipay.env 加载
│   └── aipay_test.go               # 标准向量 + 往返自测
└── cmd/
    ├── create_order_gateway_demo/          # 下单
    ├── query_pay_result_gateway_demo/      # 支付结果查询
    ├── refund_gateway_demo/                # 退款
    └── query_refund_result_gateway_demo/   # 退款结果查询
```

## 运行

```bash
cd <工程根目录>          # aipay.env 所在目录
go build -o aipay-demo ./cmd/create_order_gateway_demo
./aipay-demo
# 或直接：
go run ./cmd/create_order_gateway_demo
```

全部可替换配置（环境、密钥、证书、商户号）集中在工程根目录 `aipay.env`，
与 Java/Python/Node.js 版同一份文件、同一批键名，改完即生效，无需重新编译。

## 与其他语言版的协议口径对齐

| 项 | 口径 |
| --- | --- |
| bizContent | `信封( CMS(P7)(业务JSON) )`——明文先做 SM3WithSM2 P7 签名再进京东 SM2 数字信封 |
| tradeAmount / timestamp | JSON 数字（Long），沙箱网关严格校验数字类型 |
| deviceInfo | JSON 字符串（内含紧凑 JSON，生产实测口径；2026-10-19 生产实测对象形态会报 PARAM_ERROR） |
| content 层签名 | HMAC-SM3，字段 ASCII 升序（sign/signType 不参与，空值不参与） |
| HTTP 头 | 全小写（`app-id` / `stream-type` 等，网关大小写敏感） |
| 响应 bizContent | 商户私钥解信封 + 剥 CMS 层取业务明文 |

## 自测

```bash
go test ./aipay/                       # 标准向量（GB/T）+ 算法往返 + 签名原文格式
# 与本地真实 aipay.env 联调互验（PFX 解析、P7 签名、信封全链路往返）：
AIPAY_INTEROP_ENV=/path/to/aipay.env go test -v -run TestPfxInterop ./aipay/
```

互验测试在未设置 `AIPAY_INTEROP_ENV` 时自动跳过，不会读取任何本地路径。
