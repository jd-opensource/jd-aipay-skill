# jd-aipay-skill

京东 AI付一站式接入 Skill —— 让 AI 帮你完成支付接入。

京东 AI付是面向 AI 时代的智能支付解决方案：让支付成为 AI 交易链路中的自然一环。京东 AI付不是简单地把传统收银台搬到 AI 场景中，而是围绕"智能体交易"重新组织支付交互、支付决策和支付安全能力。

## ✨ 这是什么

这是一个 **AI 编程工具 Skill**，为商户提供京东 AI付服务端接入的一站式 AI 引导体验。只需在 AI 编程工具中说 **"帮我接入 AI付"**，即可自动完成代码集成、沙箱联调、生产配置替换和上线验证。

## 🤖 支持的平台

Claude Code · Codex · Cursor · GitHub Copilot 等主流 AI 编程工具，一条命令自动安装：

```bash
npx skills add jd-opensource/jd-aipay-skill
```

## 🚀 核心能力

| 能力 | 说明 |
| --- | --- |
| **服务端代码集成** | 自动识别 Java / Node.js / Python 项目，在现有工程中完成支付接口代码集成 |
| **沙箱联调** | 引导沙箱环境配置与接口联调，记录请求结果和排障建议；沙箱密钥/证书全部内置，仅需提供沙箱实例 ID |
| **主动下一步引导** | 每完成代码集成、沙箱联调、实名、产品开通、证书密钥、生产配置等任务后，主动提示商户下一步该做什么 |
| **产品开通/实名/证书密钥指引** | 按商户状态引导完成实名认证、产品开通申请与签约、证书申请、SM3 密钥生成/获取 |
| **证书转 Base64** | 企业站下载的商户证书（pfx/p12）与京东公钥证书（pem/cer）一键转 Base64，可直接写入工程根目录的 aipay.env |
| **生产配置替换** | 协助将沙箱配置切换为生产环境，敏感信息自动脱敏 |
| **上线验证** | 引导小额真实交易验证，确保支付链路端到端跑通 |
| **接入答疑** | 覆盖接口字段、签名加密、状态码、错误码等常见问题 |

## 📁 项目结构

```
jd-aipay-onboarding/
├── SKILL.md                          # Skill 定义（入口）
├── playbooks/                        # 接入编排剧本
│   ├── 00-response-style.md          # 回复语气规范
│   ├── 01-start-integration.md       # 开始接入
│   ├── 02-detect-server-project.md   # 项目检测
│   ├── 03-server-integration-java.md # Java 服务端接入
│   ├── 04-server-integration-nodejs.md # Node.js 服务端接入
│   ├── 05-server-integration-python.md # Python 服务端接入
│   ├── 06-product-opening-parallel-reminder.md # 产品开通并行提醒
│   ├── 07-sandbox-joint-debug.md     # 沙箱联调
│   ├── 08-certificate-secret-guide.md # 证书密钥指引
│   ├── 09-production-cutover.md      # 生产配置替换
│   ├── 10-go-live-verification.md    # 上线验证
│   ├── 11-client-sdk-guidance.md     # 客户端 SDK 指引
│   ├── 13-real-name-auth-guide.md    # 实名认证引导
│   ├── 14-product-opening-guide.md   # 产品开通详细引导
│   └── 15-platform-next-step-router.md # 平台侧主动下一步路由
├── reference/                        # 知识库
│   ├── api/                          # 接口协议、签名、加密、参数、状态码
│   └── product/                      # 产品认知、开通、实名、证书、密钥 QA
├── assets/                           # 集成资产与示例工程
│   └── server-examples/
│       ├── java-quickstart/          # Java 示例工程
│       ├── nodejs-quickstart/        # Node.js 示例工程
│       └── python-quickstart/        # Python 示例工程
└── scripts/                          # 辅助脚本
    └── render_server_example.sh      # 服务端示例渲染
```

## 📖 使用方式

### 一键安装

确保已安装 Node.js（≥14），运行：

```bash
# 安装（自动检测当前 AI 编程工具）
npx skills add jd-opensource/jd-aipay-skill

# 安装到所有平台
npx skills add jd-opensource/jd-aipay-skill -g
```

> 卸载：`npx skills remove jd-aipay-onboarding`（可加 `-g` 全局卸载）

### 接入流程

在你的服务端项目目录下打开 AI 编程工具，说：

```
帮我接入 AI付
```

或明确指定：

```
Spring Boot 项目接入京东 AI付
```

Skill 会按以下流程自动推进：

1. **项目检测** — 识别当前工作区的语言和框架（Java / Node.js / Python）
2. **代码集成** — 在现有项目中集成下单、支付查询、退款、退款查询接口
3. **主动下一步引导** — 代码集成完成后主动提示产品开通/实名认证并行推进
4. **沙箱联调** — 配置沙箱环境，逐一验证接口调用
5. **产品开通/实名/证书密钥指引** — 按状态引导完成实名、申请开通、签约、证书申请和 SM3 密钥获取
6. **生产配置替换** — 将沙箱配置安全替换为生产配置
7. **线上验证** — 小额真实交易验证支付链路，并在官网完成交易号验证

### 仅咨询问题

不需要做代码集成时，也可以直接提问：

```
AI付的签名规则是什么？
createOrder 接口的 bizContent 怎么加密？
沙箱环境报 ILLEGAL_SIGN 怎么排查？
```

## 🔗 支持的接口

| 接口 | 说明 |
| --- | --- |
| `createOrder` | 创建支付订单 |
| `queryPayResult` | 查询支付结果 |
| `refund` | 发起退款 |
| `queryRefundResult` | 查询退款结果 |

## 🌐 常用链接

- 🏠 AI付官网：https://aipay.jdpay.com/
- ⚡ 快速接入：https://aipay.jdpay.com/developers
- 🔧 一站式接入：https://aipay.jdpay.com/process?productId=ai_pay
- 🧾 产品开通：https://aipay.jdpay.com/process?productId=ai_pay&activeProcessStep=PRODUCT_OPEN
- 🪪 去实名认证：https://qy-web.jdpay.com/select?source=QYZ&channel=00006&scene=SMRZ&bussCode=REAL_NAME&merchantno={merchantno}&r={callback}
  - `merchantno` 为商户号，`callback` 为认证完成后的回跳地址；参数不明确时，从产品开通页点击「去实名」。
- 🔐 证书密钥：https://aipay.jdpay.com/process?productId=ai_pay&activeProcessStep=CERT_KEY
- ✅ 线上验证：https://aipay.jdpay.com/process?productId=ai_pay&activeProcessStep=ONLINE_VERIFY
- 📞 技术支持：jdpay-bd@jd.com / 400-098-8500

## 📄 License

[MIT](LICENSE) © JD.com
