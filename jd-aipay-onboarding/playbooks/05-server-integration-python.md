# Python 服务端接入

## 目标

在商户现有 Python 服务端项目中接入 AI付服务端能力。

## 集成范围

- 下单、查单、退款、退款查询
- SM2 bizContent 加密和 HMAC-SM3 签名
- 沙箱配置和后续生产配置切换

## 做法

1. 识别 FastAPI、Flask、Django 或其他 Python 服务端框架。
2. 遵循项目已有目录：routers/views/services/config/tests。
3. 封装 AI付 client/service，避免将加密、签名、HTTP 调用散落在视图函数中。
4. 敏感配置放入环境变量或项目既有配置系统。
5. 使用项目现有测试/启动命令验证。

## 参考资产

- 示例工程：`assets/server-examples/python-quickstart/`
- API 文档：`reference/api/`
