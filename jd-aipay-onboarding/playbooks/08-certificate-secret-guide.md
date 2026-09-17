# 证书密钥指引

## 标准规则

- 实名认证是创建商户证书的前置条件。
- 证书创建不依赖产品开通完成；产品审核中也可以先创建证书。
- 生产密钥创建与获取依赖产品开通完成。
- 证书私钥和生产密钥都属于敏感信息，需要保存在商户自己的服务端安全环境中。

## 标准话术

如果贵司已经完成实名认证，可以先去申请商户证书；证书不必等产品开通完成。生产密钥需要等产品开通通过后再创建或获取。证书私钥和生产密钥都属于敏感信息，请只配置到商户自己的服务端安全环境中，不要公开发送或提交到代码仓库。

## 入口

- 一站式接入页：https://aipay.jdpay.com/onboarding?product=AI_PAY
- 密钥获取步骤：https://aipay.jdpay.com/onboarding?product=AI_PAY&step=key

## 证书转 Base64

用户从企业站下载证书后（常见：`xxx.pfx`/`xxx.p12` 商户私钥证书、`xxx.pem.cer`/`xxx.cer` 京东公钥证书），说"把证书转成 base64""帮我转一下证书"时，用脚本完成转换并回填：

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

1. pfx 类：优先建议加 `--password` 校验，密码错误立即提示，不产出无效 Base64
2. 结果键名对应：pfx → `AIPAY_MERCHANT_PFX`；京东公钥证书 → `AIPAY_PUBLIC_KEY`
3. **展示给用户时只回显头尾各 4 位 + 总长度**用于核对（如 `MIIH****AgIEAA==（2512 字符）`），完整 Base64 优先写文件或直接写入 aipay.env——超长文本在对话中复制极易截断
4. 商户私钥证书的 Base64 视同敏感信息：提醒用户不要提交到代码仓库、不要贴到公开渠道
