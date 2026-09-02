# local-repo — 服务端示例工程本地 Maven 仓库

本目录以标准 Maven 仓库布局，内置了服务端示例工程所需的 `com.wangyin.plat-arch:wyaks-security:1.1.1`，用于演示 AI付服务端接口的加签、加密与验签流程。

示例工程根目录下的 `pom.xml` 通过 `<repositories>` 声明 `file://${project.basedir}/local-repo` 优先查找本地目录。

## 依赖分析

`wyaks-security:1.1.1` 的 pom 中：

- **compile scope**（会传递到示例工程 classpath）——全部 Maven Central 可拉：
  - `org.bouncycastle:bcpkix-jdk15on:1.60`（示例工程 `pom.xml` 已显式锁死 1.60）
  - `org.bouncycastle:bcprov-jdk15on:1.60`（同上）
  - `com.itextpdf:itextpdf:5.5.8`、`com.itextpdf:itext-asian:5.2.0`
  - `com.google.code.gson:gson:2.8.6`
  - `commons-codec:commons-codec:1.9`
- **test scope**（Maven 传递依赖时自动过滤，示例工程不受影响）：
  - `com.wangyin.plat-arch:wyaks-server-api:2.0.5`（京东内网）
  - `com.jd:jsf:1.6.0`（京东内网）
  - `apache-logging:commons-logging:1.1.0.jboss`
  - 其他 testng / junit / spring-test

结论：示例工程只需要内置 `wyaks-security` 这一个 artifact，即可在没有京东内网 Maven 私服的环境下完成编译演示。

## 目录结构

```
local-repo/
└── com/wangyin/plat-arch/wyaks-security/1.1.1/
    ├── wyaks-security-1.1.1.jar   (~1.7 MB)
    └── wyaks-security-1.1.1.pom
```

## 更新版本

京东内网机器拉到新版本后：

```bash
VERSION=<新版本号>
DEST=assets/server-examples/java-quickstart/local-repo/com/wangyin/plat-arch/wyaks-security/${VERSION}
mkdir -p "$DEST"
cp ~/.m2/repository/com/wangyin/plat-arch/wyaks-security/${VERSION}/wyaks-security-${VERSION}.{jar,pom} "$DEST/"
```

同步更新示例工程 `pom.xml` 中 `wyaks-security` 的 `<version>` 即可。若新版本引入新的 compile scope 传递依赖且非 Maven Central 来源，需要将该依赖也一并放入本目录。
