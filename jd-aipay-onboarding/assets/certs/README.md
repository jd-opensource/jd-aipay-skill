# assets/certs/ — 内置共享京东 SM2 公钥证书

本目录存放京东 AI 付外场开放接口统一分发的京东 SM2 公钥证书（**公开材料**，非商户私钥/密钥）。

## jd-sm2-pub.b64

- **用途**：商户侧构造 SM2 证书信封（`encType=SM2`）时，用京东公钥加密请求业务报文（`bizContent`）。
- **适用环境**：`pre`（预发）与 `prod`（生产）通用同一份；沙箱环境待京东发布后另行补充。
- **使用方式**：接入方**无需再提供**该公钥。`scripts/render_server_example.sh` 在 KV 配置未提供 `sm2_jd_pub` 时自动读取本文件注入示例工程的 `__SM2_JD_PUB__` 占位符；显式提供 `sm2_jd_pub` 时以提供值为准。
- **格式**：Base64（DER）单行文本，读取时会剥离全部空白字符。

## 证书元信息（用于到期轮换核对）

| 项 | 值 |
| --- | --- |
| Subject CN | 京东集团-京东科技-金融科技群-金融科技研发部-支付平台研发部-支付交易研发组(AKS00000AKS) |
| Subject | C=CN, OU=jr sm2 company, O=JDD |
| Issuer | C=CN, O=北京天威诚信电子商务服务有限公司, OU=SM2证书系统, CN=天威诚信数字认证中心CA |
| 签名算法 | SM3WithSM2（OID 1.2.156.10197.1.501） |
| 有效期 | 2026-06-08 ~ 2027-06-08（UTC） |
| 密钥用法 | Digital Signature, Non Repudiation, Key Encipherment, Data Encipherment |
| 序列号 | 1b:6d:af:49:f9:28:33:f0:db:37:fe:fb:91:b1:51:62:8c:30:38:f5 |

> **到期提醒**：证书 2027-06-08 到期，到期前需由京东侧换发新共享证书并更新本目录，同时核对 Subject/Issuer 是否变化。
