# assets/certs/ — 内置共享京东 SM2 公钥证书

本目录存放京东 AI 付外场开放接口分发的京东 SM2 公钥证书（**公开材料**，非商户私钥/密钥，允许内置在 skill 中）。

## jd-sm2-pub.b64（pre / prod 通用）

- **用途**：商户侧构造 SM2 证书信封（`encType=SM2`）时，用京东公钥加密请求业务报文（`bizContent`）。
- **适用环境**：`pre`（预发）与 `prod`（生产）通用同一份。
- **使用方式**：接入方**无需再提供**该公钥。`scripts/render_server_example.sh` 在 KV 配置未提供 `sm2_jd_pub` 时自动按环境读取：pre/prod 用本文件，sandbox 用 `jd-sm2-pub-sandbox.b64`；用户显式提供 `sm2_jd_pub` 时以提供值为准。
- **格式**：Base64（DER）单行文本，读取时会剥离全部空白字符。

### 证书元信息（用于到期轮换核对）

| 项 | 值 |
| --- | --- |
| Subject CN | 京东集团-京东科技-金融科技群-金融科技研发部-支付平台研发部-支付交易研发组(AKS00000AKS) |
| Subject | C=CN, OU=jr sm2 company, O=JDD |
| Issuer | C=CN, O=北京天威诚信电子商务服务有限公司, OU=SM2证书系统, CN=天威诚信数字认证中心CA |
| 签名算法 | SM3WithSM2（OID 1.2.156.10197.1.501） |
| 有效期 | 2026-06-08 ~ 2027-06-08（UTC） |
| 密钥用法 | Digital Signature, Non Repudiation, Key Encipherment, Data Encipherment |
| 序列号 | 1b:6d:af:49:f9:28:33:f0:db:37:fe:fb:91:b1:51:62:8c:30:38:f5 |

## jd-sm2-pub-sandbox.b64（sandbox 专用）

- **用途**：同上，用于**沙箱环境**（网关 `https://fpitest.jd.com`）的 SM2 证书信封加密。
- **适用环境**：仅 `env=sandbox`；与 pre/prod 共享证书**不是同一把**，选沙箱环境时由渲染脚本自动选用。
- **格式**：Base64（DER）单行文本。

### 证书元信息（用于到期轮换核对）

| 项 | 值 |
| --- | --- |
| Subject CN | 京东集团-京东科技-金融科技事业群-金融科技研发部-基础研发部(AKS00000AKS) |
| Subject | C=CN, OU=jr sm2 company, O=JDD |
| Issuer | C=CN, O=北京天威诚信电子商务服务有限公司, OU=SM2证书系统, CN=天威诚信数字认证中心CA |
| 签名算法 | SM3WithSM2（OID 1.2.156.10197.1.501） |
| 有效期 | 2026-07-20 ~ 2027-07-20（UTC） |
| 序列号 | 47:c4:cb:86:e3:ea:f1:2c:67:e9:5f:f4:58:c1:79:d2:78:42:64:8c |

## jd-merchant-pfx-sandbox.b64（sandbox 商户测试私钥）

- **用途**：沙箱环境构造 SM2 证书信封时，商户侧签名所用的**测试私钥证书**（PKCS#12/PFX 的 Base64，密码同样内置在渲染脚本中，均为京东分发的沙箱公共测试物料）。
- **内嵌证书**：CN=金脉智测(AKS00001AKS)，序列号 51777790E943529D8FC7059DF166BA7202D72643，有效期 2026-07-20 ~ 2027-07-20。
- **适用环境**：仅 `env=sandbox`；选沙箱环境时由渲染脚本自动注入，商户无需准备任何证书。**严禁用于 pre/prod 生产环境**。

> **到期提醒**：三份物料分别于 2027-06-08（pre/prod 京东公钥）、2027-07-20（sandbox 京东公钥、sandbox 商户测试私钥）到期，到期前需由京东侧换发并更新本目录，同时核对 Subject/Issuer 是否变化。
