# 标品支付主流程 — JD AI Pay (京东外场AI付)

This is the **standard-payment** integration flow. `SKILL.md` routes here for 标品 / 收银台 /
未指定平台的接入请求。It wires the **JD AI Pay SDK** (外场AI付 / `JDAIPay`) into a host mobile
app and verifies the integration.

The same payment model exists on both platforms — `appId`, `userId`, `aiTokenParam` (issued by
your server), unified callback `status` (`JDAIPAY_PAY_SUCCESS` / `JDAIPAY_PAY_FAIL` /
`JDAIPAY_PAY_CANCEL`) — but the packaging, init point, and result-delivery mechanism differ per
platform, so this flow detects the platform and routes to the correct per-platform reference.

> 说明：本流程的操作指令用英文（对 coding agent 更稳定），面向最终用户的提示与注意事项用中文。

## Workflow overview

Do these in order. Each step gates the next — don't generate integration code before you
know the platform and have located the SDK binaries.

1. **Detect** the target platform (iOS / Android / both / unknown).
2. **Locate** the SDK binaries the user has provided locally.
3. **Integrate** using the platform reference (one platform, or both independently).
4. **Smoke-test**: build, then run an init + mock sign/pay call and capture the debug log.
5. **Report** 注意事项 and the test results to the user.

---

## Step 1 — Detect the target platform

Run the bundled detector against the user's project root (or the directory they point at):

```bash
bash <skill-dir>/scripts/client-sdk-scripts/detect_env.sh [project-root]   # defaults to current directory
```

It prints one of `android`, `ios`, `both`, or `none`, plus the root path it found for each
platform (e.g. a Flutter/React-Native repo has both `android/` and `ios/` subfolders).

Routing rules:

- **Exactly one platform** → integrate that one.
- **Both platforms** → integrate **each one independently** in its own root. Treat them as
  two separate integrations; don't share build files or init code between them.
- **None detected** → don't guess. Ask the user which platform to integrate, and where the
  project root is. 用中文询问，例如：「没有识别到 iOS 或 Android 工程，请问要接入哪一端？工程根目录在哪里？」

If detection finds a platform but you're unsure it's the *right* root (monorepo, multiple
modules), confirm the target module with the user before editing build files.

---

## Step 2 — Obtain & locate the SDK binaries

The SDK binaries must be provided by the user. Ask the user for the SDK download address
first, then download/extract if needed, and finally verify the files are in place.

### 2.1 Ask the user for the SDK source

用中文询问用户：「请提供 JD AI 付 SDK 的下载地址（支持 HTTP/HTTPS 链接或本地 zip 文件路径）。
下载文件为 zip 包，内含 AI 付 SDK 及其依赖 SDK。」

The user will provide one of:
- **HTTP/HTTPS URL** — a network download link (e.g. OSS direct link)
- **Local file path** — a zip file already on disk

### 2.2 Download and extract

Run the bundled download script against the user's project root:

```bash
bash <skill-dir>/scripts/client-sdk-scripts/download_sdk.sh "<url_or_path>" [project-root]
```

The script will:
1. If the source is an HTTP(S) URL → `curl` download the zip file
2. If the source is a local path → use it directly
3. Extract (unzip) into the project root directory
4. Report which SDK files (`.aar` / `JDPay.xcframework`) were found after extraction

If the download or extraction fails, relay the error to the user and ask them to verify
the URL or file.

### 2.3 Verify SDK files are in place

After extraction, search the project to confirm the correct files exist:

**Android** — look for the AAR (typically in `app/libs/` or `<module>/libs/`):
- `*.aar` (the JD AI Pay AAR, e.g. `JDAIPay-*.aar`)

**iOS** — look for the frameworks and resources:
- `JDPay.xcframework`
- `CryptoLib.xcframework`
- `JDBBaseInfoModule.xcframework`
- `JDCNRecord.xcframework`
- `openssl.xcframework`
- `JDPay.xcframework/.../Res/` containing fonts `JDPay-JR-Bold.otf`, `JDPay-JR-Medium.otf`,
  `JDPay-JR-Regular.otf` and bundles `JDPay.bundle`, `JDPayComponents.bundle`,
  `JDPayNetwork.bundle`, `JDPayUIKit.bundle`

If the expected binaries are still not found after extraction, stop and inform the user.
用中文说明，例如：「SDK 下载/解压完成，但未找到预期的 SDK 文件（Android 需要 .aar；iOS 需要
JDPay.xcframework 及 Res 资源）。请确认 zip 包内容是否正确。」
The SDK is mandatory — do not stub or fabricate it.

You'll also need, before a real (non-mock) payment can run — these come from the business
side, not from the SDK:
- `appId` — 京东 AI 付分配的 APPID
- `userId` — 业务方用户唯一标识
- `aiTokenParam` / `signParam` / `payParam` — issued by **your server**, not generated on-device

For the smoke test you may use clearly-labelled placeholders for these; flag them in the
final report so the user replaces them.

---

## Step 3 — Integrate

Read the matching reference and follow it. It contains the exact build config, init code,
the reusable wrapper, the sample call site, and platform-specific gotchas.

- **Android** → read `android.md`
- **iOS** → read `ios.md`

For a **both** project, complete Android fully, then iOS fully (or vice-versa). Keep the two
integrations isolated.

Scope for each platform (full wiring):
1. Add the SDK dependency + required build configuration.
2. Initialize the SDK at the app entry point (`Application` / `AppDelegate`).
3. Add a small reusable wrapper (`JDAIPayHelper` / `JDAIPayManager`) exposing
   `init / sign / pay / openSetting`.
4. Add one sample call site so the integration is demonstrably exercised.
5. Enable debug logging so the smoke test can observe the SDK starting.

Match the host project's existing language and conventions (Kotlin vs Java, Swift vs
Objective-C, view-binding style, package names). The references give both language variants —
pick the one the project already uses.

---

## Step 4 — Build + initialization smoke test

The goal is **objective proof the SDK is wired correctly** without needing a live payment
(real payments require server-issued tokens). Run as much of this as the environment allows;
if a real build can't run here, say so explicitly rather than claiming success.

1. **Build** the project:
   - Android: `./gradlew assembleDebug` (or the module's assemble task)
   - iOS: `xcodebuild -scheme <Scheme> -sdk iphonesimulator -configuration Debug build`
     and verify on the **iOS Simulator first**; if the simulator can't validate the SDK init
     path or capture the required log, retry on a **real device**.
2. **Initialize + mock-call**: ensure `registeredService` runs at startup and the sample
   call site invokes `sign`/`pay` with clearly-labelled mock params. Enable debug mode
   (`setDebugMode(true)` / `setDebugMode:YES`).
3. **Capture evidence**:
   - Android: capture the SDK's init/runtime output under the tag/keyword
     **`JDAIPaySDKLog`** from Logcat.
   - iOS: treat the exact log **`[JDAIPaySDK][INIT] registeredService success`** as the
     proof of successful init, captured from the Xcode console. Prefer the **iOS Simulator**
     first; if the simulator cannot verify it, fall back to a **real device**. You may
     filter by `JDAIPaySDK` to find it quickly.

Report the test outcome honestly using this rubric:

| Result | Meaning |
| --- | --- |
| ✅ PASS | Project builds **and** the platform-appropriate init log is observed (`JDAIPaySDKLog` on Android; `[JDAIPaySDK][INIT] registeredService success` on iOS) |
| ⚠️ PARTIAL | Builds, but the required platform-specific init log couldn't be captured in this environment |
| ❌ FAIL | Build error or SDK fails to start (see `../notes.md`) |

If anything fails, consult `../notes.md` (troubleshooting) before reporting.

---

## Step 5 — Report to the user (中文)

Finish by surfacing the cautions and the test result. Keep it concise and actionable. Use
this structure (in Chinese, since the end user is the app's business/dev owner):

```
## 京东 AI 付接入完成（<平台：iOS / Android / 双端>）

### 已完成
- <列出实际修改/新增的文件，例如 build.gradle、AppDelegate、JDAIPayHelper 等>

### ⚠️ 必要注意事项
- aiTokenParam / signParam / payParam 必须由服务端下发，当前 demo 用的是占位值，上线前替换。
- <平台相关的关键注意项，从 references 的“注意事项”小节中提取与本次接入相关的几条>

### 测试结果
- 构建：<PASS / FAIL + 关键信息>
- 初始化冒烟测试：<PASS / PARTIAL / FAIL，附 iOS 的 `[JDAIPaySDK][INIT] registeredService success` 或 Android 的 `JDAIPaySDKLog` 关键日志，或说明为何无法采集>

### 后续待办
- <用户需要自己补的东西，例如真实 token、真机验证、声纹权限弹窗文案等>
```

Always include, at minimum, these cross-platform cautions (they bite every integration):

- **Token 来自服务端**：`aiTokenParam`/`signParam`/`payParam` 由业务服务端生成下发，不要在客户端硬编码或伪造。
- **声纹权限**：用到声纹支付需申请麦克风权限（Android `RECORD_AUDIO`；iOS `NSMicrophoneUsageDescription`），否则会失败。
- **回调状态**：统一判断 `status` ∈ {`JDAIPAY_PAY_SUCCESS`, `JDAIPAY_PAY_FAIL`, `JDAIPAY_PAY_CANCEL`}，失败时读 `errorCode`/`errorMessage`。
- Platform-specific cautions live in each reference's **注意事项** section — pull the relevant ones in.

---

## Reference files

- `android.md` — Android: AAR + gradle, `JDAIPay` API,
  `onActivityResult` result handling, wrapper + sample, build/log verification, 注意事项.
- `ios.md` — iOS: `JDPay.xcframework` + `CryptoLib.xcframework` +
  `JDBBaseInfoModule.xcframework` + `JDCNRecord.xcframework` + `openssl.xcframework` +
  Res resources, `libc++`, `CoreMotion.framework`, `WebKit.framework`,
  `LSApplicationQueriesSchemes`, permissions, `JDAIPaySDK_mainModule` init + `handleURL`,
  completion-block API, wrapper + sample, build/log verification, 注意事项.
- `../notes.md` — cross-platform troubleshooting and the doc inconsistencies you'll
  hit (API name variants in the manual, minSdk vs. stated min version, etc.). Read this when
  a build or init fails, or when the manual seems to contradict itself.
