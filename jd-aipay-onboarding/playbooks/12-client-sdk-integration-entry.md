# 客户端 SDK 集成入口

当用户明确需要客户端 SDK 集成（iOS / Android）时，按以下流程推进：

## Step 1: 识别支付类型和平台

### 支付类型
- **标品支付**（收银台 / 声纹支付）→ 使用 `client-sdk/standard/` 文档
- **眼镜支付**（智能眼镜端）→ 使用 `client-sdk/glasses/` 文档

### 平台识别
- **iOS**：检测 `.xcodeproj`、`.xcworkspace`、`Podfile`、Swift/Objective-C 文件
- **Android**：检测 `build.gradle`、`settings.gradle`、Java/Kotlin 文件

## Step 2: 读取对应平台的主流程文档

### 标品支付
- iOS：`playbooks/client-sdk/standard/ios.md`
- Android：`playbooks/client-sdk/standard/android.md`
- 或先读取主流程：`playbooks/client-sdk/standard/jdaipay-integration.md`

### 眼镜支付
- iOS：`playbooks/client-sdk/glasses/ios.md`
- Android：`playbooks/client-sdk/glasses/android.md`
- 或先读取主流程：`playbooks/client-sdk/glasses/jdaipay-glasses.md`

## Step 3: 按流程执行

流程文档会指引你：
1. 使用 `scripts/client-sdk-scripts/detect_env.sh` 自动检测平台
2. 使用 `scripts/client-sdk-scripts/download_sdk.sh` 下载 SDK（如果用户未提供）
3. 集成 SDK（导入 framework / AAR）
4. 添加初始化代码
5. 添加支付调用代码
6. 冒烟测试（编译 + 验证初始化）
7. 输出中文集成报告

## Step 4: 故障排查

如遇到问题，参考：
- `playbooks/client-sdk/notes.md` - 跨平台故障排查与手册不一致点

## 注意事项

1. 客户端 SDK 集成只是支付流程的一部分，真实支付需要：
   - 服务端已完成接入（createOrder 接口）
   - 服务端生成订单并返回 token 给客户端
   - 客户端调用 SDK 支付接口，传入 token
   - 服务端接收支付回调并处理业务逻辑

2. 如果用户只完成客户端集成，提醒：
   - 需要服务端配合提供订单创建接口
   - 需要完成服务端的沙箱联调和生产配置
   - 建议按全栈接入流程推进（服务端 → 客户端 → 端到端联调）

3. 用商户友好的语言说明集成步骤，避免使用过于技术化的术语。
