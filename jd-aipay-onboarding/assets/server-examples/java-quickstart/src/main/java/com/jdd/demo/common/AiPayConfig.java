package com.jdd.demo.common;

/**
 * AI 付接入配置（环境、凭证、密钥）。
 *
 * <p>由 render_server_example.sh 按所选环境自动填充。切换环境或更换凭证时，
 * 直接修改本文件中的常量即可，无需重新生成工程。业务参数（订单号、用户、金额等）
 * 不在本类，位于各 Demo 的业务方法中。</p>
 */
public class AiPayConfig {

    /** 环境标识：pre / prod / sandbox */
    public static final String ENV = "__ENV__";

    /** 接口全路径 URL（含环境域名；沙箱环境含沙箱实例 ID） */
    public static final String ENDPOINT_URL = "__ENDPOINT_URL__";

    /** 应用 ID */
    public static final String APP_ID = "__APP_ID__";

    /** 渠道/Agent 标识 */
    public static final String AGENT_ID = "__AGENT_ID__";

    /** 商户号 */
    public static final String MERCHANT_NO = "__MERCHANT_NO__";

    /** 收单商户号 */
    public static final String ACQ_MERCHANT_NO = "__ACQ_MERCHANT_NO__";

    /** 接入类型：SERVICE_MER 服务商 / COMMON 普通商户 */
    public static final String ACCESS_TYPE = "__ACCESS_TYPE__";

    /** HMAC-SM3 签名密钥 */
    public static final String SECRET_KEY = "__SECRET_KEY__";

    /** 商户私钥 pfx 密码 */
    public static final String PFX_PASSWORD = "__PFX_PASSWORD__";

    /** 商户私钥 pfx 文件的 Base64 编码 */
    public static final String PFX_BASE64 = "__PFX_BASE64__";

    /** 京东 SM2 公钥证书 Base64（按环境由渲染脚本注入：沙箱/共享） */
    public static final String SM2_JD_PUB = "__SM2_JD_PUB__";
}
