# Android Integration — JD AI Pay (外场AI付)

Source: 外场AI付Android接入手册. Follow the steps in order. Adapt package names, language
(Kotlin/Java), and Activity wiring to the host project — don't paste blindly.

## Table of contents
1. Environment requirements
2. Add the SDK (AAR + gradle)
3. Initialize the SDK
4. API surface
5. Reusable wrapper (`JDAIPayHelper`)
6. Sample call site + result handling
7. Build + log verification
8. 注意事项 (cautions)

---

## 1. Environment requirements

- JDK 1.7+
- Android API 21+ recommended by the manual's environment section.
- Build config in the manual: `compileSdk 35`, `targetSdk 35`, `minSdk 19`.

> ⚠️ The manual is internally inconsistent: the environment section says "Android 21+",
> while the sample `build.gradle` sets `minSdk 19`. Default to the host app's existing
> `minSdk`. If the app sets `minSdk` below 19, keep it but warn the user that the JD AI Pay
> SDK targets 19+ (21+ per the environment note) and lower devices are unsupported.

## 2. Add the SDK (AAR + gradle)

1. Place the JD AI Pay `*.aar` in the module's `libs/` directory (create `app/libs/` if it
   doesn't exist).
2. In `app/build.gradle` (or `build.gradle.kts`), ensure the SDK config and the `libs` AAR
   dependency:

```gradle
android {
    compileSdk 35
    defaultConfig {
        minSdk 19          // keep the host app's existing minSdk if higher
        targetSdk 35
        // ...
    }
}

dependencies {
    // pulls every .aar from libs/ — includes the JD AI Pay AAR
    implementation fileTree(include: ['*.aar'], dir: 'libs')
}
```

Kotlin DSL (`build.gradle.kts`) equivalent for the dependency:

```kotlin
implementation(fileTree(mapOf("dir" to "libs", "include" to listOf("*.aar"))))
```

If the project already uses a `libs/` fileTree include, you don't need to add it twice —
just confirm the AAR is in `libs/`. Sync Gradle after editing.

### D8 / dexing 处理（通用）

接入第三方 AAR 后最常见的两类构建报错都来自 **D8**（Android 的 dex 编译器）：方法数超限
（`minSdk < 21` 触发）和 Java 8+ 字节码脱糖。JD AI Pay 把 `minSdk` 压到 19，两者都可能踩到。
按报错决定开哪个，不要预防性全开。

**Multidex（`minSdk < 21` 时基本必做）** — 低于 API 21 的单个 dex 有 64K 方法上限，接入 SDK 后
极易超限，报：

```
com.android.tools.r8.D8: Cannot fit requested classes in a single dex file (# methods: 65536 > 65536)
```

处理：

```gradle
android {
    defaultConfig {
        minSdk 19
        multiDexEnabled true          // minSdk < 21 时开启
    }
}

dependencies {
    implementation 'androidx.multidex:multidex:2.0.1'
}
```

并让 `Application` 继承 `MultiDexApplication`（或已有基类时在 `attachBaseContext` 里
`MultiDex.install(this)`），这也正好是调用 `registeredService` 的入口（见 §3）：

```java
public class App extends androidx.multidex.MultiDexApplication {
    @Override public void onCreate() {
        super.onCreate();
        JDAIPay.setDebugMode(BuildConfig.DEBUG);
        JDAIPay.registeredService(this);
    }
}
```

> `minSdk >= 21` 的工程 D8 原生支持 multidex，无需以上配置，直接跳过。

**Java 8 脱糖（desugaring）** — 若编译期报 `Default interface methods are only supported
starting with Android N` 或 `Invoke-customs are only supported starting with Android O`，说明
AAR 用了 Java 8 字节码，需要脱糖：

```gradle
android {
    compileOptions {
        sourceCompatibility JavaVersion.VERSION_1_8
        targetCompatibility JavaVersion.VERSION_1_8
    }
    kotlinOptions { jvmTarget = '1.8' }   // Kotlin 工程需对齐 jvmTarget
}
```

若 SDK 还用到 `java.time`、`java.util.stream` 等库 API 且 `minSdk < 26`，再额外开启**核心库脱糖
（core library desugaring）**：

```gradle
android {
    compileOptions {
        coreLibraryDesugaringEnabled true
    }
}
dependencies {
    coreLibraryDesugaring 'com.android.tools:desugar_jdk_libs:2.0.4'
}
```

> ⚠️ 是否需要脱糖取决于 AAR 实际用到的字节码版本与 API，先按报错开，不要一次全开——无谓的脱糖会拖慢
> 构建。`multidex` / `desugar_jdk_libs` 的版本以宿主工程的 AGP 兼容性为准，别照抄写死。

**R8（release 的 D8 超集）** — release 构建走 R8 时若出现反射/`ClassNotFound`，需为 JD SDK 加
`-keep` 规则，详见 §8 注意事项。

## 3. Initialize the SDK

Call `registeredService` once at app startup — the best place is your `Application.onCreate`
(create/register a custom `Application` if the app doesn't have one and wire it in
`AndroidManifest.xml` via `android:name`).

```java
// In your Application subclass
@Override
public void onCreate() {
    super.onCreate();
    JDAIPay.setDebugMode(BuildConfig.DEBUG);   // debug log switch, default off
    JDAIPay.registeredService(this);           // initialize the SDK
}
```

## 4. API surface

From the manual (`JDAIPay` static methods):

```java
// 初始化 sdk
public static void registeredService(Context context);
// 签约
public static void sign(Context context, JDAISignParam param);
// 支付
public static void pay(Context context, JDAIPayParam param);
// 打开支付设置
public static void openPaySetting(Context context, JDAISettingParam param);
// 获取 SDK 版本号
public static String getSdkVersion();
// debug 日志开关，默认关闭
public static void setDebugMode(boolean enable);
```

**`JDAIPayParam` fields** (shared param model):

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `appId` | String | App 标识 |
| `userId` | String | 用户标识 |
| `aiTokenParam` | String | AI 付 Token 参数（**服务端下发**） |
| `appLanguage` | String | 应用语言标识（如 `zh-CN`、`en-US`） |
| `style` | String(枚举) | UI 样式：`default`（默认）、`silent`（纯接口无 UI）、`card`（卡片样式） |
| `extParams` | Map | 业务方透传参数 |
| `voiceString` | String | 声纹数据 |
| `externalProcessBlock` | Callback | 外部处理回调（SDK 需要外部处理数据时触发） |

Param setters seen in the manual: `setSignParam(...)`, `setPayParam(...)`,
`setSettingParam(...)`, `setExtParams(...)`, `setVoicePath(...)`, `setVoiceSign(...)`.

**Callback status** (delivered via `onActivityResult`, see §6):

| 字段 | 类型 | 必选 | 说明 |
| --- | --- | --- | --- |
| `status` | String | 是 | `JDAIPAY_PAY_SUCCESS` / `JDAIPAY_PAY_FAIL` / `JDAIPAY_PAY_CANCEL` |
| `extraData` | Map | 否 | 扩展数据（业务方透传回传） |
| `errorCode` | String | 否 | 错误码（失败时存在） |
| `errorMessage` | String | 否 | 错误描述（失败时存在） |

## 5. Reusable wrapper (`JDAIPayHelper`)

Add one thin wrapper so call sites stay clean and the param model is centralized. Place it
in the app's package (adjust `package`). Kotlin version:

```kotlin
package <your.app.pkg>.payment

import android.app.Activity
import android.content.Context
import com.jdpay.aipay.JDAIPay          // adjust to the AAR's actual package if it differs
import com.jdpay.aipay.JDAIPayParam
import com.jdpay.aipay.JDAISignParam
import com.jdpay.aipay.JDAISettingParam

/** Thin wrapper over the JD AI Pay SDK. Results arrive in Activity.onActivityResult. */
object JDAIPayHelper {

    fun init(context: Context, debug: Boolean = false) {
        JDAIPay.setDebugMode(debug)
        JDAIPay.registeredService(context.applicationContext)
    }

    /** 签约 — signParam 由服务端下发 */
    fun sign(activity: Activity, signParam: String, ext: Map<String, Any>? = null) {
        val p = JDAISignParam().apply {
            setSignParam(signParam)
            ext?.let { setExtParams(it) }
        }
        JDAIPay.sign(activity, p)
    }

    /** 支付 — payParam 由服务端下发；voicePath/voiceSign 仅声纹支付需要 */
    fun pay(
        activity: Activity,
        payParam: String,
        voicePath: String? = null,
        voiceSign: String? = null,
        ext: Map<String, Any>? = null,
    ) {
        val p = JDAIPayParam().apply {
            setPayParam(payParam)
            voicePath?.let { setVoicePath(it) }
            voiceSign?.let { setVoiceSign(it) }
            ext?.let { setExtParams(it) }
        }
        JDAIPay.pay(activity, p)
    }

    /** 打开支付设置 */
    fun openSetting(activity: Activity, settingParam: String, ext: Map<String, Any>? = null) {
        val p = JDAISettingParam().apply {
            setSettingParam(settingParam)
            ext?.let { setExtParams(it) }
        }
        JDAIPay.openPaySetting(activity, p)
    }

    fun sdkVersion(): String = JDAIPay.getSdkVersion()
}
```

> The exact import package of `JDAIPay` depends on the AAR shipped to the user. If the
> compiler can't resolve `com.jdpay.aipay.JDAIPay`, inspect the AAR (`unzip -l <aar>` → look at
> `classes.jar`, or check the SDK's javadoc/demo) and fix the import. Don't invent classes.

## 6. Sample call site + result handling

Results are returned to the **calling Activity** via `onActivityResult` — **not** via a
listener. This is the single most error-prone part on Android.

```java
@Override
protected void onActivityResult(int requestCode, int resultCode, Intent data) {
    super.onActivityResult(requestCode, resultCode, data);
    if (Constants.PAY_RESPONSE_CODE == resultCode) {     // JD result resultCode
        String result = data.getStringExtra(JDPayAuthor.JDPAY_RESULT);
        // parse `result` -> status / errorCode / errorMessage
    }
}
```

Sample call site (e.g. a "Pay" button); use clearly-labelled mock params for the smoke test:

```kotlin
// DEMO ONLY — replace with server-issued params before release
JDAIPayHelper.pay(this, payParam = "MOCK_PAY_PARAM_FROM_SERVER")
```

> ⚠️ 官方提示：**若页面栈发生变化则无法准确收到回调**。Launch sign/pay from a stable Activity that owns
> `onActivityResult`; avoid finishing/recreating it mid-flow, and be careful with
> `singleTask`/`singleInstance` launch modes or nested fragments that change the back stack.

## 7. Build + log verification

```bash
./gradlew assembleDebug          # or :app:assembleDebug for the module
```

Run the app, then watch Logcat for the SDK's init/runtime output:

```bash
adb logcat | grep JDAIPaySDKLog
```

Seeing `JDAIPaySDKLog` output (with `setDebugMode(true)`) is the proof the SDK started.

## 8. 注意事项 (cautions)

- **回调走 `onActivityResult`**：必须由发起支付的 Activity 接收；页面栈变化会丢回调。
- **`minSdk` 取值**：手册环境写 21+、示例 gradle 写 19，按宿主工程实际值，低于 19 需提示用户风险。
- **AAR 包名**：`JDAIPay` 实际包名以用户提供的 AAR 为准，import 解析不了就去 AAR 里查，别臆造。
- **声纹支付需 `RECORD_AUDIO` 权限**，需要在宿主工程的 `AndroidManifest.xml` 中配置，并在运行时申请。
- **D8/dexing**：`minSdk < 21` 需开 `multiDexEnabled` 并让 `Application` 走 multidex；报 Java 8 字节码错时按需开脱糖（见 §2 的 D8 处理小节）。
- **混淆**：若开启 R8/ProGuard，可能需为 JD SDK 加 keep 规则（SDK 报 `ClassNotFound`/反射失败时排查）。
- **token 来自服务端**：`signParam`/`payParam`/`aiTokenParam` 全部由服务端下发，客户端勿伪造。

