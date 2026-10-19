# Go 服务端接入

## 目标

在商户现有 Go 服务端项目中接入 AI付服务端能力。

## 集成范围

- 下单、查单、退款、退款查询
- SM2 数字信封（含 CMS P7 签名）和 HMAC-SM3 签名
- 沙箱配置和后续生产配置切换

## 做法

1. 识别 Gin、Go-zero、Kratos、标准库 net/http 或其他 Go 服务端框架。
2. 遵循项目已有目录：cmd/internal/pkg/api 等。
3. 将 `assets/server-examples/go-quickstart/aipay/` 作为共享包引入（或按模块规范复制），
   避免将加密、签名、HTTP 调用散落在 handler 中。
4. 敏感配置（密钥、证书、商户号）走 `aipay.env` 或项目既有配置系统，不要硬编码。
5. 用 `go test ./aipay/` 做标准向量自测；用 `AIPAY_INTEROP_ENV=<aipay.env路径>
   go test -run TestPfxInterop ./aipay/` 做真实证书互验。
6. 使用项目现有测试/启动命令验证。

## 说明

- **零第三方依赖**：国密算法（SM2/SM3/SM4）、数字信封、PFX、CMS 均为纯 Go +
  标准库实现，不引入任何 module 依赖，`go build` 即用。
- 报文口径（2026-10-19 生产实测）：`tradeAmount`/`timestamp` 为 JSON 数字（Long），
  `deviceInfo` 为 JSON 字符串（内含紧凑 JSON，对象形态会被拒 PARAM_ERROR），
  HTTP 头名全小写（网关大小写敏感）。

## 参考资产

- 示例工程：`assets/server-examples/go-quickstart/`
- API 文档：`reference/api/`
