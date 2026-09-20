package com.jdd.demo.common;

import java.io.BufferedReader;
import java.io.IOException;
import java.io.InputStream;
import java.io.InputStreamReader;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.util.LinkedHashMap;
import java.util.Map;

/**
 * AI 付接入配置（环境、凭证、密钥）。
 *
 * <p>所有可替换的值集中在本工程根目录的 {@code aipay.env} 文件中（KEY=value 格式）。
 * 切换环境或更换凭证（密钥、证书、商户号、应用标识等）时，直接编辑 aipay.env 即可，
 * 无需修改任何 Java 代码、也无需重新生成工程。</p>
 *
 * <p>本类在类加载时读取 aipay.env 并初始化下方常量；文件缺失或缺项会抛出明确异常。</p>
 */
public class AiPayConfig {

    /** 环境标识：pre / prod / sandbox */
    public static final String ENV;

    /** 接口全路径 URL（含环境域名；沙箱环境含沙箱实例 ID） */
    public static final String ENDPOINT_URL;

    /** 应用 ID */
    public static final String APP_ID;

    /** 渠道/Agent 标识 */
    public static final String AGENT_ID;

    /** 商户号 */
    public static final String MERCHANT_NO;

    /** 收单商户号 */
    public static final String ACQ_MERCHANT_NO;

    /** 接入类型：SERVICE_MER 服务商 / COMMON 普通商户 */
    public static final String ACCESS_TYPE;

    /** HMAC-SM3 签名密钥 */
    public static final String SECRET_KEY;

    /** 商户私钥 pfx 密码 */
    public static final String PFX_PASSWORD;

    /** 商户私钥 pfx 文件的 Base64 编码 */
    public static final String PFX_BASE64;

    /** 京东 SM2 公钥证书 Base64（按环境：沙箱/共享） */
    public static final String SM2_JD_PUB;

    static {
        Map<String, String> vals = loadEnvFile();
        ENV = require(vals, "AIPAY_ENV");
        ENDPOINT_URL = require(vals, "AIPAY_ENDPOINT_URL");
        APP_ID = require(vals, "AIPAY_APP_ID");
        AGENT_ID = require(vals, "AIPAY_AGENT_ID");
        MERCHANT_NO = require(vals, "AIPAY_MERCHANT_NO");
        ACQ_MERCHANT_NO = require(vals, "AIPAY_ACQ_MERCHANT_NO");
        ACCESS_TYPE = require(vals, "AIPAY_ACCESS_TYPE");
        SECRET_KEY = require(vals, "AIPAY_SECRET_KEY");
        PFX_PASSWORD = require(vals, "AIPAY_MERCHANT_PFX_PASSWORD");
        PFX_BASE64 = require(vals, "AIPAY_MERCHANT_PFX");
        SM2_JD_PUB = require(vals, "AIPAY_PUBLIC_KEY");
    }

    private AiPayConfig() { }

    /**
     * 从工程根目录查找并解析 aipay.env。
     * 查找顺序：当前工作目录 → 逐级向上（覆盖 mvn 在子目录执行的场景）。
     */
    private static Map<String, String> loadEnvFile() {
        Path dir = Paths.get("").toAbsolutePath();
        for (int i = 0; i < 6; i++) {
            Path candidate = dir.resolve("aipay.env");
            if (Files.isRegularFile(candidate)) {
                return parse(candidate);
            }
            Path parent = dir.getParent();
            if (parent == null) {
                break;
            }
            dir = parent;
        }
        throw new IllegalStateException(
                "未找到配置文件 aipay.env（应在工程根目录）。该文件承载全部可替换配置项，请勿删除或改名。");
    }

    private static Map<String, String> parse(Path file) {
        Map<String, String> vals = new LinkedHashMap<>();
        try (InputStream in = Files.newInputStream(file);
             BufferedReader reader = new BufferedReader(new InputStreamReader(in, StandardCharsets.UTF_8))) {
            String line;
            while ((line = reader.readLine()) != null) {
                String trimmed = line.trim();
                if (trimmed.isEmpty() || trimmed.startsWith("#") || !trimmed.contains("=")) {
                    continue;
                }
                int idx = trimmed.indexOf('=');
                vals.put(trimmed.substring(0, idx).trim(), trimmed.substring(idx + 1).trim());
            }
        } catch (IOException e) {
            throw new IllegalStateException("读取 aipay.env 失败: " + file, e);
        }
        return vals;
    }

    private static String require(Map<String, String> vals, String key) {
        String value = vals.get(key);
        if (value == null || value.isEmpty()) {
            throw new IllegalStateException("aipay.env 缺少配置项: " + key + "，请补齐后重试");
        }
        return value;
    }
}
