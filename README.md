# jd-aipay-skill

京东 AI付一站式接入 Skill —— 让 AI 帮你完成支付接入。

京东 AI付是面向 AI 时代的智能支付解决方案：让支付成为 AI 交易链路中的自然一环。京东 AI付不是简单地把传统收银台搬到 AI 场景中，而是围绕"智能体交易"重新组织支付交互、支付决策和支付安全能力。

## ✨ 这是什么

这是一个 **AI 编程工具 Skill**，为商户提供京东 AI付**服务端 + 客户端**的一站式 AI 引导体验。只需在 AI 编程工具中说 **"帮我接入 AI付"**，即可自动完成代码集成、沙箱联调、生产配置替换和上线验证。

## 🤖 支持的平台

Claude Code · Codex · Cursor · GitHub Copilot 等主流 AI 编程工具，一条命令自动安装：

```bash
npx skills add jd-opensource/jd-aipay-skill
```

## 🚀 核心能力

### 服务端接入
| 能力 | 说明 |
| --- | --- |
| **服务端代码集成** | 自动识别 Java / Node.js / Python 项目，在现有工程中完成支付接口代码集成 |
| **沙箱联调** | 引导沙箱环境配置与接口联调，记录请求结果和排障建议 |
| **产品开通指引** | 代码集成完成后并行提醒产品开通，提供实名认证、证书密钥获取等平台操作指引 |
| **生产配置替换** | 协助将沙箱配置切换为生产环境，敏感信息自动脱敏 |
| **上线验证** | 引导小额真实交易验证，确保支付链路端到端跑通 |
| **接入答疑** | 覆盖接口字段、签名加密、状态码、错误码等常见问题 |

### 客户端 SDK 集成
| 能力 | 说明 |
| --- | --- |
| **iOS SDK 集成** | 自动检测 Xcode 项目，集成 JDPay.xcframework，配置初始化和支付调用 |
| **Android SDK 集成** | 自动检测 Android 项目，集成 AI付 AAR，配置 Gradle 依赖和支付流程 |
| **标品支付** | 支持收银台支付和声纹支付两种模式 |
| **眼镜支付** | 支持智能眼镜端支付（预留，建设中） |
| **冒烟测试** | 自动生成测试代码，验证 SDK 初始化和支付调用 |
| **故障排查** | 提供构建失败、运行时错误等常见问题的解决方案 |

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
│   ├── 12-client-sdk-integration-entry.md # 客户端 SDK 集成入口
│   └── client-sdk/                   # 客户端 SDK 集成文档
│       ├── standard/                 # 标品支付（iOS/Android）
│       │   ├── jdaipay-integration.md # 主流程
│       │   ├── ios.md                # iOS 详细步骤
│       │   └── android.md            # Android 详细步骤
│       ├── glasses/                  # 眼镜支付（预留）
│       │   ├── jdaipay-glasses.md    # 主流程
│       │   ├── ios.md                # iOS 步骤
│       │   └── android.md            # Android 步骤
│       └── notes.md                  # 故障排查与注意事项
├── reference/                        # 知识库
│   ├── api/                          # 接口协议、签名、加密、参数、状态码
│   └── product/                      # 产品认知、开通、实名、证书、密钥 QA
├── assets/                           # 集成资产与示例工程
│   └── server-examples/
│       ├── java-quickstart/          # Java 示例工程
│       ├── nodejs-quickstart/        # Node.js 示例工程
│       └── python-quickstart/        # Python 示例工程
└── scripts/                          # 辅助脚本
    ├── render_server_example.sh      # 服务端示例渲染
    └── client-sdk-scripts/           # 客户端 SDK 脚本
        ├── detect_env.sh             # 平台检测
        └── download_sdk.sh           # SDK 下载
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

#### 服务端接入

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
3. **产品开通提醒** — 并行提醒在官网完成产品开通和实名认证
4. **沙箱联调** — 配置沙箱环境，逐一验证接口调用
5. **证书密钥指引** — 引导获取生产环境证书和密钥
6. **生产配置替换** — 将沙箱配置安全替换为生产配置
7. **上线验证** — 小额真实交易验证支付链路

#### 客户端 SDK 集成

在你的 iOS 或 Android 项目目录下打开 AI 编程工具，说：

```
iOS App 接入京东 AI付
```

或：

```
Android 项目集成 JD Pay SDK
```

Skill 会按以下流程自动推进：

1. **平台检测** — 识别 iOS（Xcode）或 Android（Gradle）项目
2. **SDK 下载** — 自动下载或引导提供 JDPay.xcframework / AAR
3. **SDK 集成** — 自动添加依赖、配置 framework、修改 build.gradle
4. **初始化配置** — 生成 SDK 初始化代码（AppDelegate / Application）
5. **支付调用** — 生成支付接口调用示例代码
6. **冒烟测试** — 编译项目并验证 SDK 初始化成功
7. **中文报告** — 输出集成报告和后续联调建议

#### 全栈接入

如果需要同时完成服务端和客户端接入，说：

```
完整接入京东 AI付，包括 Spring Boot 后端和 iOS App
```

Skill 会按顺序推进：服务端接入 → 客户端 SDK 集成 → 端到端联调建议

### 仅咨询问题

不需要做代码集成时，也可以直接提问：

**服务端相关**：
```
AI付的签名规则是什么？
createOrder 接口的 bizContent 怎么加密？
沙箱环境报 ILLEGAL_SIGN 怎么排查？
```

**客户端相关**：
```
iOS SDK 的 registeredService 方法怎么用？
Android AAR 怎么导入到项目？
JDPay.xcframework 编译失败怎么办？
```

## 🔗 支持的能力

### 服务端接口

| 接口 | 说明 |
| --- | --- |
| `createOrder` | 创建支付订单 |
| `queryPayResult` | 查询支付结果 |
| `refund` | 发起退款 |
| `queryRefundResult` | 查询退款结果 |

### 客户端 SDK

| 平台 | SDK | 支持能力 |
| --- | --- | --- |
| iOS | JDPay.xcframework | 标品支付（收银台/声纹）、眼镜支付（预留） |
| Android | JDAIPaySDK.aar | 标品支付（收银台/声纹）、眼镜支付（预留） |

## 🌐 常用链接

- 🏠 AI付官网：https://aipay.jdpay.com/
- ⚡ 快速接入：https://aipay.jdpay.com/developers
- 🔧 一站式接入：https://aipay.jdpay.com/onboarding?product=AI_PAY
- 📞 技术支持：jdpay-bd@jd.com / 400-098-8500

## 📄 License

[MIT](LICENSE) © JD.com
