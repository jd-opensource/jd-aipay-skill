# Java 服务端接入

## 目标

在商户现有 Java 服务端项目中接入 AI付服务端能力，而不是另起一个孤立示例工程。

## 集成范围

- 下单 `createOrder`
- 支付结果查询 `queryPayResult`
- 退款 `refund`
- 退款结果查询 `queryRefundResult`
- SM2 bizContent 加密和 HMAC-SM3 签名
- 沙箱配置和后续生产配置切换

## 做法

1. 识别项目构建工具、框架、配置文件和测试命令。
2. 优先遵循项目已有分层：client/service/controller/config/test。
3. 将 AI付接口调用封装成独立 client 或 service，避免把签名、加密、HTTP 调用散落在业务代码里。
4. 配置项放入项目现有配置体系；敏感值使用环境变量或安全配置占位，不写死到代码。
5. 增加最小可验证测试或启动检查。

## 参考资产

- 示例工程：`assets/server-examples/java-quickstart/`
- API 文档：`reference/api/`
