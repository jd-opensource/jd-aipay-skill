# 眼镜支付 iOS 接入 — JD AI Pay (外场AI付 · 眼镜端)

> ⚠️ **预留 / 待填充（TODO）**。眼镜端 iOS SDK 与接入手册尚未提供，本文件为占位。
> `jdaipay-glasses.md` 主流程在眼镜能力上线后路由到这里。

在拿到眼镜端 iOS SDK（framework + 资源）与官方接入手册前，**不要臆造 framework、符号或调用**。按
`jdaipay-glasses.md` 的说明向用户告知现状。

## 待填充章节（上线时按实际手册补齐，结构对齐标品 `../standard/ios.md`）

1. **系统要求** — 最低 iOS 版本 / 架构（device + simulator slice）。
2. **添加 framework + 资源** — 眼镜端 `.xcframework` + `Res/` 字体与 bundle（全引入，缺一不启动）。
3. **链接依赖 & 工程配置** — 系统库、`LSApplicationQueriesSchemes`、`Info.plist` 权限
   （麦克风 / 生物特征鉴权描述）。
4. **初始化 SDK** — `AppDelegate` 注册服务 + `handleURL` 回调转发 + debug 日志开关。
5. **API surface** — 眼镜端特有：**设备绑定**、**生物特征鉴权**、**收银台唤起** 的
   completion-block 接口与参数模型。
6. **可复用封装** — 眼镜端 `JDAIPayHelper` 暴露 init / 绑定 / 鉴权 / pay。
7. **示例调用** — 唤起收银台 + 统一回调 `status` 处理。
8. **构建 + 日志验证** — `xcodebuild ... -sdk iphonesimulator` 优先模拟器，必要时真机，捕获初始化日志。
9. **注意事项** — 资源全引入、系统库缺一不可、麦克风/生物特征权限、`openURL` 回调、token 来自服务端等。

共享故障排查见 `../notes.md`。
