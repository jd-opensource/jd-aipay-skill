# iOS Integration — JD AI Pay (外场AI付)

Source: 外场AI付iOS接入手册. Follow in order. Adapt to the host project's language
(Swift/Objective-C) and project layout (Xcode project vs. workspace/CocoaPods).

## Table of contents
1. System requirements
2. Add the framework + resources
3. Link dependencies & project config (Info.plist)
4. Initialize the SDK
5. API surface
6. Reusable wrapper (`YourProjectJDAIPayHelper`)
7. Sample call site
8. Build + log verification
9. 注意事项 (cautions)

---

## 1. System requirements

- Minimum iOS **11.0**
- Architectures: `arm64` (device), `arm64 + x86_64` (simulator) — covered by the
  `.xcframework`.

## 2. Add the framework + resources

1. Drag/Add the following **5 个 xcframework** into the project: right-click the project group →
   **Add Files to "…"** → select all frameworks → in *Choose options* keep "Copy items
   if needed" sensible for your setup → **Finish**. Confirm they all appear under
   **General → Frameworks, Libraries, and Embedded Content** (Embed & Sign):
   - **`JDPay.xcframework`** — AI 付主 SDK
   - **`CryptoLib.xcframework`** — 加密库
   - **`JDBBaseInfoModule.xcframework`** — 基础信息模块
   - **`JDCNRecord.xcframework`** — 声纹模块
   - **`openssl.xcframework`** — OpenSSL 依赖
2. **(重要) Add the SDK's resources.** Missing them makes the SDK fail to start:
   - From `JDPay.xcframework`'s `Res/` directory, add **all** of these to the app target:
     - Fonts: `JDPay-JR-Bold.otf`, `JDPay-JR-Medium.otf`, `JDPay-JR-Regular.otf`
     - Bundles: `JDPay.bundle`, `JDPayComponents.bundle`, `JDPayNetwork.bundle`,
       `JDPayUIKit.bundle`
   - From `CryptoLib.xcframework`'s内部目录, add:
     - **`CryptoLib.bundle`** (路径: `CryptoLib.xcframework/CryptoLib.bundle`)
   Ensure all resources are in **Target → Build Phases → Copy Bundle Resources**.

## 3. Link dependencies & project config

**3.1 Link required system libraries/frameworks** — Target → **Build Phases → Link Binary With Libraries** → `+` → add:

- **`libc++.tbd`**
- **`CoreMotion.framework`**
- **`WebKit.framework`**

After this, *Link Binary With Libraries* must contain `JDPay.xcframework`,
`CryptoLib.xcframework`, `JDBBaseInfoModule.xcframework`, `JDCNRecord.xcframework`,
`openssl.xcframework`, `libc++.tbd`, `CoreMotion.framework`, and `WebKit.framework` — 缺一不可.

**3.2 `LSApplicationQueriesSchemes`** — in `Info.plist`, add these 3 schemes (newer Xcode
labels this "Queried URL Schemes"):

```xml
<key>LSApplicationQueriesSchemes</key>
<array>
    <string>jdpay</string>
    <string>jdpayopen</string>
    <string>jdmobilejdpay</string>
</array>
```

> ⚠️ 手册要求这些 scheme 必须排在前 50 个，否则不生效（iOS 对 `LSApplicationQueriesSchemes` 的查询数量有限制）。
> If the app already has many query schemes, keep the JD ones within the first 50 entries.

**3.3 Permissions** (`Info.plist`):

| Key | 用途 | 备注 |
| --- | --- | --- |
| `NSMicrophoneUsageDescription` (Privacy - Microphone Usage Description) | 声纹验证 | **必填** |
| `NSFaceIDUsageDescription` (Privacy - Face ID Usage Description) | FaceID 验证 | 本期未使用，可不填 |
| `NSCameraUsageDescription` (Privacy - Camera Usage Description) | 摄像头 | 本期未使用，可不填 |

Add at least the microphone key with a human description, or 声纹支付 will fail at runtime.

## 4. Initialize the SDK

In `AppDelegate`, register the service at launch and forward URL callbacks:

```objc
#import <JDPay/JDPay.h>

@implementation AppDelegate

- (BOOL)application:(UIApplication *)application
        didFinishLaunchingWithOptions:(NSDictionary *)launchOptions {
    [JDAIPaySDK_mainModule() registeredService];   // initialize the SDK
    [JDAIPaySDK_mainModule() setDebugMode:YES];     // debug log (turn off for release)
    return YES;
}

- (BOOL)application:(UIApplication *)app
            openURL:(NSURL *)url
            options:(NSDictionary<UIApplicationOpenURLOptionsKey, id> *)options {
    if ([JDAIPay canHandleURL:url options:options]) {
        [JDAIPaySDK_mainModule() handleURL:url options:options completionHandler:nil];
        return YES;
    }
    return NO;
}

@end
```

Swift (bridge `JDPay/JDPay.h` via the bridging header):

```swift
func application(_ application: UIApplication,
                 didFinishLaunchingWithOptions launchOptions: [UIApplication.LaunchOptionsKey: Any]?) -> Bool {
    JDAIPaySDK_mainModule().registeredService()
    JDAIPaySDK_mainModule().setDebugMode(true)
    return true
}

func application(_ app: UIApplication, open url: URL,
                 options: [UIApplication.OpenURLOptionsKey : Any] = [:]) -> Bool {
    if JDAIPay.canHandleURL(url, options: options) {
        JDAIPaySDK_mainModule().handleURL(url, options: options, completionHandler: nil)
        return true
    }
    return false
}
```

> ⚠️ **手册存在书写不一致**，以实际头文件为准：init 处写作 `JDAIPaySDK_mainModule()`（带括号，像取单例的函数/宏），
> 而部分示例写作 `[JDAIPaySDK_mainModule pay:...]`（不带括号，像对象）。支付/签约示例里既出现 `complete:` 又出现
> `completionHandler:`。**Verify against the real `JDPay.h`** that the user's SDK ships and use the
> actual symbol names. Don't fight the header — read it.

## 5. API surface

Public params (manual §4.1): `appId` (京东 AI 付分配的 APPID, 必填), `userId` (业务方用户唯一标识, 必填).

**Unified callback** — every business API returns an `NSDictionary` via its completion block:

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `status` | String | 是 | `JDAIPAY_PAY_SUCCESS` / `JDAIPAY_PAY_FAIL` / `JDAIPAY_PAY_CANCEL` |
| `extraData` | NSDictionary | 否 | 扩展数据（业务方透传回传） |
| `errorCode` | String | 否 | 错误码（失败时存在） |
| `errorMessage` | String | 否 | 错误描述（失败时存在） |

> Note: §4.2 lists the generic statuses as `JDAIPAY_SUCCESS / JDAIPAY_FAIL / JDAIPAY_CANCEL`
> while §4.4–4.5 use the `JDAIPAY_PAY_*` variants. Compare against the SDK headers and match
> whatever constants it actually exports; handle all three semantic outcomes
> (success / fail / cancel) regardless of the exact spelling.

**Pay** (`JDAIPayParam` — same fields as Android: `appId`, `userId`, `aiTokenParam`,
`appLanguage`, `style` ∈ {`default`, `silent`, `card`}, `extParams`, `voiceString`,
`externalProcessBlock`):

```objc
- (void)pay:(JDAIPayParam *)param
        completionHandler:(JDAIPayCompletionBlock _Nullable)completionHandler;

// 调用示例
JDAIPayParam *payInfo = [[JDAIPayParam alloc] init];
payInfo.payParam = payParam;            // server-issued
[JDAIPaySDK_mainModule() pay:payInfo complete:^(NSDictionary *result) {
    // status / errorCode / errorMessage
}];
```

**Sign** (`JDAIPaySignParam` — `appId`, `userId`, `aiTokenParam`, `extParams`):

```objc
- (void)sign:(JDAIPaySignParam *)signParam complete:(JDAIPayCompletionBlock)complete;

// 调用示例
JDAIPaySignParam *signParam = [[JDAIPaySignParam alloc] init];
signParam.appId  = @"xxxx";
signParam.userId = @"xxxx";
[JDAIPaySDK_mainModule() sign:signParam complete:^(NSDictionary *result) {
    // ....
}];
```

**Pay setting** (`JDAIPayAccountParam`):

```objc
- (void)openAccount:(JDAIPayAccountParam *)param
                    complete:(void (^ _Nullable)(NSDictionary *result));

// 调用示例
[JDAIPaySDK_mainModule() openAccount:param complete:^(NSDictionary *result) {
    // ....
}];
```

## 6. Reusable wrapper (`YourProjectJDAIPayHelper`)

Centralize the calls. Swift version (assumes the bridging header imports `<JDPay/JDPay.h>`):

```swift
import Foundation

/// Thin wrapper over the JD AI Pay SDK. Replace `YourProject` with your actual project prefix
/// and verify symbol names against JDPay.h.
enum YourProjectJDAIPayHelper {

    static func initialize(debug: Bool = false) {
        JDAIPaySDK_mainModule().registeredService()
        JDAIPaySDK_mainModule().setDebugMode(debug)
    }

    /// 签约 — signParam.aiTokenParam 由服务端下发
    static func sign(appId: String, userId: String,
                     completion: @escaping ([AnyHashable: Any]) -> Void) {
        let p = JDAIPaySignParam()
        p.appId = appId
        p.userId = userId
        JDAIPaySDK_mainModule().sign(p) { result in completion(result ?? [:]) }
    }

    /// 支付 — payParam 由服务端下发
    static func pay(payParam: String,
                    completion: @escaping ([AnyHashable: Any]) -> Void) {
        let p = JDAIPayParam()
        p.payParam = payParam
        JDAIPaySDK_mainModule().pay(p) { result in completion(result ?? [:]) }
    }

    /// 打开支付设置
    static func openSetting(_ param: JDAIPayAccountParam,
                            completion: @escaping ([AnyHashable: Any]) -> Void) {
        JDAIPaySDK_mainModule().openAccount(with: param) { result in completion(result ?? [:]) }
    }
}
```

> Swift selector mapping (`sign:complete:`, `pay:complete:`/`pay:completionHandler:`,
> `openAccountWithParam:complete:`) depends on the framework's actual ObjC signatures. If Swift
> can't see a method, check the generated interface (Xcode → jump to definition on the module)
> and adjust. Same caveat for ObjC: use the exact selectors from `JDPay.h`.

## 7. Sample call site

```swift
// DEMO ONLY — replace `YourProject` with your real project prefix, and replace payParam
// with a server-issued value before release
YourProjectJDAIPayHelper.pay(payParam: "MOCK_PAY_PARAM_FROM_SERVER") { result in
    let status = result["status"] as? String   // JDAIPAY_PAY_SUCCESS / _FAIL / _CANCEL
    print("JD AI Pay result: \(result)")
}
```

## 8. Build + log verification

Prefer validating on the **iOS Simulator first**. Only fall back to a **real device** if the
simulator cannot complete the verification you need (for example, the app can't launch there,
the SDK path you're checking isn't observable there, or the init-success log can't be captured).

```bash
xcodebuild -scheme <Scheme> -sdk iphonesimulator -configuration Debug \
  -destination 'generic/platform=iOS Simulator' build
```

(For a workspace/CocoaPods project use `-workspace <App>.xcworkspace` instead of the implicit
project.) Run on the **iOS Simulator first** and watch the Xcode console — with
`setDebugMode:YES`, successful init prints
`[JDAIPaySDK][INIT] registeredService success` (the manual: "出现以下 log 则代表初始化成功").
If you cannot verify this successfully on the simulator, repeat the run on a **real device**
before concluding the smoke test result.

## 9. 注意事项 (cautions)

- **所有 xcframework 必须全部引入**：`JDPay.xcframework`、`CryptoLib.xcframework`、`JDBBaseInfoModule.xcframework`、`JDCNRecord.xcframework`、`openssl.xcframework` 缺一不可，否则链接报错或 SDK 无法启动。
- **资源必须全引入**：`JDPay.xcframework/Res/` 下 3 个字体 + 4 个 bundle，以及 `CryptoLib.xcframework/CryptoLib.bundle`，一个都不能少，否则 SDK 无法启动。
- **`libc++.tbd`、`CoreMotion.framework`、`WebKit.framework` 必须同时链接**，缺一不可。
- **`LSApplicationQueriesSchemes` 的 3 个 scheme 要在前 50 个**，否则跳转失败。
- **麦克风权限 `NSMicrophoneUsageDescription` 必填**（声纹验证），否则声纹支付失败。
- **`openURL` 回调要接好**：在 `AppDelegate` 转发给 `handleURL`，否则从京东 App 跳回时丢结果。
- **手册符号不一致**：`JDAIPaySDK_mainModule` 的带括号/不带括号写法、`complete:` vs `completionHandler:` 以实际头文件为准。
- **token 来自服务端**：`payParam`/`aiTokenParam`/sign 参数由服务端下发，客户端勿伪造。
