# Troubleshooting & doc inconsistencies — JD AI Pay

Read this when a build or init fails, or when the manual seems to contradict itself. The
official manuals are short and have a few internal inconsistencies; the rule throughout is
**trust the actual SDK headers/AAR over the PDF screenshots**, and never fabricate a symbol.

## Known doc inconsistencies (don't let these trip you up)

| Where | The inconsistency | What to do |
| --- | --- | --- |
| Android env vs gradle | Env says "Android 21+", sample `build.gradle` says `minSdk 19` | Use host app's `minSdk`; warn if below 19. |
| Android pay sample | Declares `JDAIPayParam payParam` but then calls `JDAIPay.pay(context, orderPayParam)` | Typo in the manual; pass the param object you built. |
| iOS init symbol | `JDAIPaySDK_mainModule()` (with parens) vs `[JDAIPaySDK_mainModule pay:...]` (without) | Use whatever `JDPay.h` actually exports. |
| iOS completion | `complete:` in some samples, `completionHandler:` in the method decl | Match the real selector in the header. |
| iOS status enum | §4.2 uses `JDAIPAY_SUCCESS/FAIL/CANCEL`; §4.4–4.5 use `JDAIPAY_PAY_SUCCESS/FAIL/CANCEL` | Match exported constants; handle all 3 outcomes by meaning. |
| Both | SDK download URLs are "待提供" (blank) | The user supplies binaries locally — see jdaipay-integration.md Step 2. |

When the header and the manual disagree, the header wins. Inspect it:
- Android: `unzip -l <sdk>.aar`, then look inside `classes.jar` (`unzip -o classes.jar -d /tmp/jdaipay && javap -classpath /tmp/jdaipay <FQCN>`), or read the bundled demo/javadoc.
- iOS: open `JDPay.xcframework/<slice>/Headers/JDPay.h`, or in Xcode jump-to-definition on the module to see the generated Swift interface.

## Android failures

- **`Unresolved reference: JDAIPay` / wrong package** → the wrapper's `import` doesn't match
  the AAR. Find the real FQCN from the AAR (`unzip -l`) and fix the import. Don't invent one.
- **AAR not packaged** → confirm `implementation fileTree(include: ['*.aar'], dir: 'libs')`
  and that the `.aar` is actually in `app/libs/`. Re-sync Gradle.
- **No callback in `onActivityResult`** → the page stack changed (官方提示). Launch from a
  stable Activity; avoid `singleTask`/`singleInstance`, mid-flow `finish()`, or fragment
  back-stack churn.
- **`ClassNotFoundException` / reflection failures in release** → add ProGuard/R8 `-keep`
  rules for the JD SDK package.
- **声纹支付失败** → missing `RECORD_AUDIO` permission or runtime grant.
- **No `JDAIPaySDKLog` output** → `setDebugMode(true)` not called, or `registeredService`
  not run at startup.

## iOS failures

- **SDK won't start / crashes on launch** → resources not added. Re-check all 3 fonts + 4
  bundles from `Res/` are in **Copy Bundle Resources**.
- **Linker errors (`Undefined symbols` for C++/std)** → `libc++.tbd` not linked. Add it in
  *Link Binary With Libraries* (must coexist with `JDPay.xcframework`).
- **Jump-back from JD app loses the result** → `openURL` not forwarded to
  `handleURL:options:completionHandler:` in `AppDelegate` (or `SceneDelegate` if the app uses
  scenes — wire the equivalent `scene:openURLContexts:`).
- **URL scheme / app jump not working** → the 3 schemes missing from
  `LSApplicationQueriesSchemes`, or pushed past the first 50 entries.
- **声纹支付失败** → `NSMicrophoneUsageDescription` missing.
- **No `[JDAIPaySDK][INIT] registeredService success` log** → `setDebugMode:YES` not
  called, or `registeredService` not run at startup.
- **Swift can't see a method** → check the generated ObjC→Swift interface; the selector maps
  differently than you guessed. Adjust the wrapper.

## SceneDelegate note (modern iOS apps)

If the app uses `UIScene` (has a `SceneDelegate`), URL callbacks may arrive via
`scene(_:openURLContexts:)` instead of `application(_:open:options:)`. Forward those URLs to
the SDK's `handleURL` there too, otherwise return-from-JD-app callbacks are lost.

## Honesty about the smoke test

Real payments need server-issued `aiTokenParam`/`payParam`/`signParam`, so an end-to-end
paid transaction can't be verified at integration time. The smoke test proves **init +
build**, not a real charge. Report exactly what was verified:
- On **iOS**, try the **Simulator first**; if it cannot verify the SDK init/log path, fall
  back to a **real device** before reporting `PARTIAL`.
- ✅ PASS — builds and the platform-appropriate init log observed (`JDAIPaySDKLog` on
  Android; `[JDAIPaySDK][INIT] registeredService success` on iOS).
- ⚠️ PARTIAL — builds, but that required platform-specific log couldn't be captured here
  (e.g. no device/emulator/CI). Say so.
- ❌ FAIL — build or init error; cite the actual error and the fix attempted.

Never report a green result you didn't actually observe.
