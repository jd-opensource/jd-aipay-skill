package com.jdd.demo;

import com.alibaba.fastjson.JSONObject;
import com.jdd.demo.common.Constants;
import com.jdd.demo.utils.EncryptUtils;

import org.bouncycastle.crypto.digests.SM3Digest;
import org.bouncycastle.crypto.macs.HMac;
import org.bouncycastle.crypto.params.KeyParameter;

import javax.net.ssl.HostnameVerifier;
import javax.net.ssl.HttpsURLConnection;
import javax.net.ssl.SSLContext;
import javax.net.ssl.SSLSession;
import javax.net.ssl.TrustManager;
import javax.net.ssl.X509TrustManager;
import java.io.BufferedReader;
import java.io.InputStream;
import java.io.InputStreamReader;
import java.io.OutputStream;
import java.net.HttpURLConnection;
import java.net.URL;
import java.nio.charset.StandardCharsets;
import java.security.SecureRandom;
import java.security.cert.X509Certificate;
import java.text.SimpleDateFormat;
import java.util.Base64;
import java.util.Date;
import java.util.LinkedHashMap;
import java.util.Map;
import java.util.TreeMap;
import java.util.UUID;
import java.util.concurrent.ThreadLocalRandom;

/**
 * AI 付下单接口 Demo（网关版本）：
 * bizContent 使用 SM2 数字信封加密（encType=SM2），外层用 HMAC-SM3 计算 sign，
 * 组装 data.content 结构后通过 HTTPS POST 调用网关接口。
 */
public class RokidCreateOrderGatewayDemo {

    /** 环境标识：pre / prod / sandbox，由 render 脚本替换 */
    private static final String ENV = "__ENV__";

    /** 京东 SM2 公钥证书 Base64 —— 敏感参数，由用户按环境提供（pre/prod 使用同一份，sandbox 使用沙箱证书） */
    private static final String SM2_JD_PUB = "__SM2_JD_PUB__";

    /** 签名密钥（HMAC-SM3）—— 敏感参数，由用户提供 */
    private static final String SECRET_KEY = "__SECRET_KEY__";

    /** 商户私钥 pfx 文件的 Base64 编码 —— 敏感参数，由用户提供 */
    private static final String PFX_BASE64 = "__PFX_BASE64__";

    /** 商户私钥 pfx 密码 —— 敏感参数，由用户提供 */
    private static final String PFX_PASSWORD = "__PFX_PASSWORD__";

    /** 接口全路径 URL（含环境域名）—— skill 生成时替换 */
    private static final String ENDPOINT_URL = "__ENDPOINT_URL__";

    public static void main(String[] args) throws Exception {
        String bizJson = buildBizJson();

        String bizContent = encodeBizContent(bizJson);
        Map<String, String> content = buildContent();
        content.put("bizContent", bizContent);

        String signString = buildSignString(content);
        String sign = computeSign(signString, SECRET_KEY);
        content.put("sign", sign);

        JSONObject data = new JSONObject(true);
        data.put("content", content);
        JSONObject body = new JSONObject(true);
        body.put("data", data);

        Map<String, String> httpHeader = buildHttpHeader(content.get("appId"));

        System.out.println("=================== bizContent 明文 ===================");
        System.out.println(bizJson);
        System.out.println("=================== 签名原文 ===================");
        System.out.println(signString);
        System.out.println("=================== 签名结果 ===================");
        System.out.println(sign);
        System.out.println("=================== HTTP Header ===================");
        for (Map.Entry<String, String> e : httpHeader.entrySet()) {
            System.out.println(e.getKey() + ":" + e.getValue());
        }
        System.out.println("=================== HTTP Body ===================");
        System.out.println(body.toJSONString());

        String responseText = postJson(ENDPOINT_URL, httpHeader, body.toJSONString());
        System.out.println("=================== HTTP Response ===================");
        System.out.println(responseText);

        tryDecryptResponseBizContent(responseText);
    }

    private static String buildBizJson() {
        String createDate = new SimpleDateFormat("yyyyMMddHHmmss").format(new Date());

        // 设备信息（JSON 字符串）
        JSONObject device = new JSONObject(true);
        device.put("vendor", "TEST");
        device.put("deviceId", "SN-TEST-01231231");
        device.put("deviceAccount", "test_device_001");
        device.put("deviceName", "测试设备");
        device.put("deviceType", "SMART_DEVICE");
        device.put("deviceSn", "SN-TEST-01231231");
        device.put("deviceModel", "TestModel");

        JSONObject biz = new JSONObject(true);
        // 交易主题
        biz.put("tradeSubject", "AI付测试订单");
        // 客户端类型：APP
        biz.put("clientType", "APP");
        // 设备信息（JSON 字符串）
        biz.put("deviceInfo", device.toJSONString());
        // 交易金额（分）—— 由用户提供
        biz.put("tradeAmount", "__TRADE_AMOUNT__");
        // 下单时间：yyyyMMddHHmmss
        biz.put("createDate", createDate);
        // 收单商户号 —— 由用户提供
        biz.put("acqMerchantNo", "__ACQ_MERCHANT_NO__");
        // 接入类型：SERVICE_MER 服务商 / COMMON 普通商户 —— 由用户选择
        biz.put("accessType", "__ACCESS_TYPE__");
        // 商户外部订单号 —— 由用户提供
        biz.put("outTradeNo", "__OUT_TRADE_NO__");
        // 交易类型
        biz.put("tradeType", "Aipay");
        // 用户 IP
        biz.put("userIp", "127.0.0.1");
        // 用户 ID —— 由用户提供
        biz.put("userId", "__USER_ID__");
        // 交易备注
        biz.put("tradeRemark", "AI付接口测试");
        // 支付结果异步通知地址
        biz.put("notifyUrl", "https://merchant.example.com/notify/pay");
        // 币种
        biz.put("currency", "CNY");
        // 订单有效期（秒）
        biz.put("expiryTime", "604800");
        return biz.toJSONString();
    }

    private static String encodeBizContent(String bizJson) throws Exception {
        return EncryptUtils.encryptForSm2WithBase64(bizJson, PFX_BASE64, PFX_PASSWORD, SM2_JD_PUB);
    }

    private static Map<String, String> buildContent() {
        Map<String, String> map = new TreeMap<>();
        map.put("appId", "__APP_ID__");
        map.put("merchantNo", "__MERCHANT_NO__");
        map.put("agentId", "__AGENT_ID__");
        map.put("reqNo", UUID.randomUUID().toString().replace("-", "").toUpperCase());
        map.put("timestamp", String.valueOf(System.currentTimeMillis()));
        map.put("nonce", randomHex(16));
        map.put("version", "1.0");
        map.put("signType", "SM3");
        map.put("encType", "SM2");
        return map;
    }

    private static Map<String, String> buildHttpHeader(String appId) {
        Map<String, String> h = new LinkedHashMap<>();
        h.put("app-id", appId);
        h.put("encrypt-type", "NONE");
        h.put("source-type", "H5");
        h.put("login-type", "0");
        h.put("cache-control", "no-cache");
        h.put("content-type", "application/json");
        h.put("stream-type", "false");
        return h;
    }

    private static String buildSignString(Map<String, String> content) {
        // content 层字段按 ASCII 升序排列，排除 sign / signType，空值不参与；
        // encType=SM2 时该字段进入签名串（防降级篡改）。
        TreeMap<String, String> params = new TreeMap<>();
        putIfNotEmpty(params, "agentId", content.get("agentId"));
        putIfNotEmpty(params, "appId", content.get("appId"));
        putIfNotEmpty(params, "bizContent", content.get("bizContent"));
        putIfNotEmpty(params, "encType", content.get("encType"));
        putIfNotEmpty(params, "merchantNo", content.get("merchantNo"));
        putIfNotEmpty(params, "nonce", content.get("nonce"));
        putIfNotEmpty(params, "reqNo", content.get("reqNo"));
        String ts = content.get("timestamp");
        if (ts != null && !ts.isEmpty()) {
            params.put("timestamp", ts);
        }
        putIfNotEmpty(params, "version", content.get("version"));

        StringBuilder sb = new StringBuilder();
        boolean first = true;
        for (Map.Entry<String, String> e : params.entrySet()) {
            if (!first) sb.append("&");
            sb.append(e.getKey()).append("=").append(e.getValue());
            first = false;
        }
        return sb.toString();
    }

    private static void putIfNotEmpty(Map<String, String> map, String key, String value) {
        if (value != null && !value.isEmpty()) {
            map.put(key, value);
        }
    }

    private static String computeSign(String stringToSign, String secretKey) {
        HMac mac = new HMac(new SM3Digest());
        mac.init(new KeyParameter(secretKey.getBytes(StandardCharsets.UTF_8)));
        byte[] data = stringToSign.getBytes(StandardCharsets.UTF_8);
        mac.update(data, 0, data.length);
        byte[] out = new byte[mac.getMacSize()];
        mac.doFinal(out, 0);
        return bytesToHex(out);
    }

    private static String bytesToHex(byte[] bytes) {
        StringBuilder sb = new StringBuilder(bytes.length * 2);
        for (byte b : bytes) {
            sb.append(String.format("%02x", b & 0xFF));
        }
        return sb.toString();
    }

    private static String randomHex(int len) {
        StringBuilder sb = new StringBuilder(len);
        for (int i = 0; i < len; i++) {
            sb.append(Integer.toHexString(ThreadLocalRandom.current().nextInt(16)));
        }
        return sb.toString().toUpperCase();
    }

    private static String postJson(String urlStr, Map<String, String> headers, String bodyJson) throws Exception {
        URL url = new URL(urlStr);
        HttpURLConnection conn = (HttpURLConnection) url.openConnection();
        if (conn instanceof HttpsURLConnection) {
            trustAllHttps((HttpsURLConnection) conn);
        }
        conn.setRequestMethod("POST");
        for (Map.Entry<String, String> e : headers.entrySet()) {
            conn.setRequestProperty(e.getKey(), e.getValue());
        }
        conn.setDoOutput(true);
        conn.setConnectTimeout(10_000);
        conn.setReadTimeout(30_000);
        try (OutputStream os = conn.getOutputStream()) {
            os.write(bodyJson.getBytes(StandardCharsets.UTF_8));
        }
        int code = conn.getResponseCode();
        InputStream is = code >= 400 ? conn.getErrorStream() : conn.getInputStream();
        StringBuilder resp = new StringBuilder();
        if (is != null) {
            try (BufferedReader br = new BufferedReader(new InputStreamReader(is, StandardCharsets.UTF_8))) {
                String line;
                while ((line = br.readLine()) != null) {
                    resp.append(line);
                }
            }
        }
        return "HTTP " + code + " | " + resp.toString();
    }

    private static void trustAllHttps(HttpsURLConnection conn) throws Exception {
        TrustManager[] trustAll = new TrustManager[]{new X509TrustManager() {
            public X509Certificate[] getAcceptedIssuers() { return new X509Certificate[0]; }
            public void checkClientTrusted(X509Certificate[] chain, String authType) { }
            public void checkServerTrusted(X509Certificate[] chain, String authType) { }
        }};
        SSLContext ctx = SSLContext.getInstance("TLS");
        ctx.init(null, trustAll, new SecureRandom());
        conn.setSSLSocketFactory(ctx.getSocketFactory());
        conn.setHostnameVerifier(new HostnameVerifier() {
            public boolean verify(String hostname, SSLSession session) { return true; }
        });
    }

    /**
     * 尝试解密响应 bizContent：encType=SM2 时用商户私钥解签数字信封，否则 Base64 解码打印。
     */
    private static void tryDecryptResponseBizContent(String responseText) {
        try {
            int idx = responseText.indexOf('{');
            if (idx < 0) return;
            JSONObject root = JSONObject.parseObject(responseText.substring(idx));
            JSONObject dataObj = root.getJSONObject("data");
            if (dataObj == null) return;
            JSONObject contentObj = dataObj.getJSONObject("content");
            if (contentObj == null) return;
            String encType = contentObj.getString("encType");
            String bizContent = contentObj.getString("bizContent");
            if (bizContent == null || bizContent.isEmpty()) return;
            System.out.println("=================== 响应 encType ===================");
            System.out.println(encType);
            System.out.println("=================== 响应 bizContent 明文 ===================");
            if ("SM2".equalsIgnoreCase(encType)) {
                String plain = EncryptUtils.decryptForSm2WithBase64(bizContent, PFX_BASE64, PFX_PASSWORD);
                System.out.println(plain);
            } else {
                byte[] plain = Base64.getDecoder().decode(bizContent);
                System.out.println(new String(plain, StandardCharsets.UTF_8));
            }
        } catch (Exception ex) {
            System.out.println("[响应 bizContent 解密失败] " + ex.getMessage());
        }
    }
}
