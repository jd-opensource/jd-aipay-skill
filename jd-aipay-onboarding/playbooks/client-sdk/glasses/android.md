# 眼镜支付 Android 接入 — JD AI Pay (外场AI付 · 眼镜端)

> ⚠️ **预留 / 待填充（TODO）**。眼镜端 Android SDK 与接入手册尚未提供，本文件为占位。
> `jdaipay-glasses.md` 主流程在眼镜能力上线后路由到这里。

在拿到眼镜端 Android SDK 与官方接入手册前，**不要臆造依赖、类名或调用**。按
`jdaipay-glasses.md` 的说明向用户告知现状。

## 待填充章节（上线时按实际手册补齐，结构对齐标品 `../standard/android.md`）

1. **环境要求** — JDK / API level / 眼镜端设备与系统要求。
2. **添加 SDK** — 眼镜端 AAR / 依赖 + gradle 配置（含 D8 / multidex / 脱糖处理，按报错开）。
3. **初始化 SDK** — `Application.onCreate` 注册服务 + debug 日志开关。
4. **API surface** — 眼镜端特有：**设备绑定**、**生物特征鉴权**、**收银台唤起** 的方法与参数模型。
5. **可复用封装** — `JDAIPayHelper`（眼镜端）暴露 init / 绑定 / 鉴权 / pay / 回调。
6. **示例调用 + 回调处理** — 结果回传机制（onActivityResult 或眼镜端特有通道，待手册确定）。
7. **构建 + 日志验证** — `./gradlew assembleDebug` + 捕获初始化日志关键字。
8. **注意事项** — 眼镜端权限（麦克风 / 生物特征）、设备绑定失败排查、token 来自服务端等。

共享故障排查见 `../notes.md`。
