# Node.js 服务端接入

## 目标

在商户现有 Node.js 服务端项目中接入 AI付服务端能力。

## 集成范围

- 下单、查单、退款、退款查询
- SM2 bizContent 加密和 HMAC-SM3 签名
- 沙箱配置和后续生产配置切换

## 做法

1. 识别 Express、NestJS、Koa、Fastify 或其他服务端框架。
2. 遵循项目已有目录：routes/controllers/services/config/tests。
3. 封装 AI付 client，避免将加密、签名、HTTP 调用散落在路由里。
4. 敏感配置放入 `.env` 或项目既有配置系统，不提交真实密钥。
5. 使用项目现有测试/启动命令验证。

## 参考资产

- 示例工程：`assets/server-examples/nodejs-quickstart/`
- API 文档：`reference/api/`
