# 证书密钥指引

## 标准规则

- 实名认证是创建商户证书的前置条件。
- 证书创建不依赖产品开通完成；产品审核中也可以先创建证书。
- 生产密钥的创建与获取依赖产品开通完成并完成合同签约。
- 商户证书、私钥文件、证书密码、SM3 密钥都属于敏感信息，需要保存在商户自己的服务端安全环境中。
- 完成证书密钥后，要主动引导商户进入生产配置替换；不要只回答“已完成”。

## 证书密钥入口

- 证书密钥步骤：`https://aipay.jdpay.com/process?productId=ai_pay&activeProcessStep=CERT_KEY`
- 入口路径：AI付官网 → 快速接入 → 选择产品 → 传统接入方式 → 立即接入/继续接入 → 证书密钥获取。

## 获取证书

### 什么时候做

商户完成实名认证后即可申请证书，不必等待产品开通审核完成。

### 标准话术

实名认证通过后，可以先把商户证书准备好。证书不需要等产品开通完成，这样后续产品开通和签约完成后，就能更快进入生产配置替换和线上验证。

请进入证书密钥步骤：
https://aipay.jdpay.com/process?productId=ai_pay&activeProcessStep=CERT_KEY

如果页面提示暂无有效证书，按页面引导前往国密证书管理/企业站申请证书。证书生成过程中会涉及私钥、P10/CSR、公钥证书 Base64 和最终 PFX 文件，请妥善保存，不要提交到代码仓库或公开发送。

### 操作概要

1. 打开「国密证书管理」页面，点击「申请证书」。
2. 在安装国密证书页面下载「证书导出工具」。
3. 用 Codex / Cursor 等 AI 编辑器打开证书导出工具，输入：`调用此工具生成证书`。
4. 工具生成本地私钥文件和 P10/CSR 文件。
5. 将 P10 文件内容粘贴到安装国密证书页面，点击「下一步」。
6. 复制页面返回的「公钥证书」Base64 编码。
7. 将公钥证书 Base64 提供给 AI 编辑器，合成最终商户证书 PFX 文件。

### 生成过程中可能出现的文件

- 私钥文件：`[商户名称]_pri.key`
- P10/CSR 文件：`[商户名称]_p10.csr`
- 公钥证书：`[商户名称]_public_cert.base64`
- 最终商户证书：`[商户名称].pfx`

最终生产配置通常只需要：

- 商户证书文件：`[商户名称].pfx`
- 证书密码

## 生成和获取密钥

### 什么时候做

产品开通申请审核通过并完成合同签约后，再生成或获取生产密钥。

### 标准话术

产品开通完成并签约后，请进入证书密钥步骤获取生产密钥：
https://aipay.jdpay.com/process?productId=ai_pay&activeProcessStep=CERT_KEY

如页面提示还没有密钥，可按页面引导进入京东金融企业版/企业站的「产品协议」页面，找到「AI付」产品，点击「创建密钥」，在「产品密钥配置」页面依次生成 SM3 密钥、shaKey 密钥并提交。完成后回到 AI付官网「证书密钥获取」步骤，复制生产 SM3 密钥。

### 敏感信息处理

- 不要在对话中粘贴完整 SM3 密钥、shaKey、证书密码或私钥内容。
- 需要我协助配置时，优先让我写入本地环境变量文件或安全配置文件。
- 回复中只回显脱敏信息，例如 `abcd****wxyz`，并说明完整值由商户自行核对。

## 证书转 Base64

用户从企业站下载证书后（常见：`xxx.pfx`/`xxx.p12` 商户私钥证书、`xxx.pem.cer`/`xxx.cer` 京东公钥证书），说“把证书转成 base64”“帮我转一下证书”时，用脚本完成转换并回填：

```bash
# pfx（可加 --password 校验密码正确性）
bash <skill根目录>/scripts/cert_to_base64.sh /path/to/商户证书.pfx --password <证书密码>

# 京东公钥证书（PEM/DER 均可）
bash <skill根目录>/scripts/cert_to_base64.sh /path/to/京东公钥.pem.cer

# 直接写入工程的 aipay.env（推荐，避免超长文本在对话中复制出错）
bash <skill根目录>/scripts/cert_to_base64.sh /path/to/商户证书.pfx --password <密码> \
  --env-file /path/to/工程/aipay.env --env-key AIPAY_MERCHANT_PFX
```

**执行要求：**

1. pfx 类：优先建议加 `--password` 校验，密码错误立即提示，不产出无效 Base64。
2. 结果键名对应：pfx → `AIPAY_MERCHANT_PFX`；京东公钥证书 → `AIPAY_PUBLIC_KEY`。
3. **展示给用户时只回显头尾各 4 位 + 总长度**用于核对（如 `MIIH****AgIEAA==（2512 字符）`），完整 Base64 优先写文件或直接写入 aipay.env。
4. 商户私钥证书的 Base64 视同敏感信息：提醒用户不要提交到代码仓库、不要贴到公开渠道。

## 完成后的主动下一步

- **只完成了证书，产品未开通**：引导去产品开通页提交/查看申请，入口 `https://aipay.jdpay.com/process?productId=ai_pay&activeProcessStep=PRODUCT_OPEN`。
- **产品已开通但密钥未生成**：引导继续在证书密钥页生成/获取密钥。
- **证书和密钥均完成**：引导进入生产配置替换，读取 `playbooks/09-production-cutover.md`。
- **生产配置已替换**：引导进入线上验证，入口 `https://aipay.jdpay.com/process?productId=ai_pay&activeProcessStep=ONLINE_VERIFY`。
