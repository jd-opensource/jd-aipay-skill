/**
 * AI 付 createOrder 接口 Demo（Node.js 版）：
 * bizContent 使用 SM2 数字信封加密（encType=SM2），外层用 HMAC-SM3 计算 sign，
 * 组装 data.content 结构后通过 HTTPS POST 调用网关接口。
 */
'use strict';

const {
  buildContent,
  buildHttpHeaders,
  buildSignString,
  encodeBizContent,
  hmacSm3Hex,
  parsePfxBase64,
  postJson,
  tryDecryptResponseBizContent,
} = require('./utils/common');

const ENV = '__ENV__'; // pre | prod | sandbox
const SM2_JD_PUB = '__SM2_JD_PUB__'; // 京东 SM2 公钥证书 Base64 —— 敏感参数，由用户按环境提供

// 敏感参数
const SECRET_KEY = '__SECRET_KEY__'; // HMAC-SM3 密钥
const PFX_BASE64 = '__PFX_BASE64__'; // 商户 pfx Base64
const PFX_PASSWORD = '__PFX_PASSWORD__'; // 商户 pfx 密码

// 接口 URL 与身份
const ENDPOINT_URL = '__ENDPOINT_URL__'; // 完整 endpoint URL
const APP_ID = '__APP_ID__';
const AGENT_ID = '__AGENT_ID__';
const MERCHANT_NO = '__MERCHANT_NO__'; // 服务商商户号
const ACQ_MERCHANT_NO = '__ACQ_MERCHANT_NO__'; // 收单商户号
const ACCESS_TYPE = '__ACCESS_TYPE__'; // 接入类型：SERVICE_MER 服务商 / COMMON 普通商户

// 业务参数
const OUT_TRADE_NO = '__OUT_TRADE_NO__';
const USER_ID = '__USER_ID__';
const TRADE_AMOUNT = '__TRADE_AMOUNT__'; // 分

function pad2(n) {
  return n < 10 ? '0' + n : String(n);
}

function nowYyyyMmDdHhMmSs() {
  const d = new Date();
  return (
    d.getFullYear().toString() +
    pad2(d.getMonth() + 1) +
    pad2(d.getDate()) +
    pad2(d.getHours()) +
    pad2(d.getMinutes()) +
    pad2(d.getSeconds())
  );
}

function buildBizJson() {
  // 保持字段顺序与 Python/Java demo 一致（LinkedHashMap / OrderedDict）
  const device = {
    vendor: 'TEST',
    deviceId: 'SN-TEST-01231231',
    deviceAccount: 'test_device_001',
    deviceName: '测试设备',
    deviceType: 'SMART_DEVICE',
    deviceSn: 'SN-TEST-01231231',
    deviceModel: 'TestModel',
  };
  const biz = {
    tradeSubject: 'AI付测试订单', // 交易主题
    clientType: 'APP', // 客户端类型
    deviceInfo: JSON.stringify(device),
    tradeAmount: TRADE_AMOUNT, // 交易金额（分）
    createDate: nowYyyyMmDdHhMmSs(),
    acqMerchantNo: ACQ_MERCHANT_NO, // 收单商户号
    accessType: ACCESS_TYPE, // 接入类型：SERVICE_MER 服务商 / COMMON 普通商户
    outTradeNo: OUT_TRADE_NO, // 商户外部订单号
    tradeType: 'Aipay', // 交易类型
    userIp: '127.0.0.1', // 用户 IP
    userId: USER_ID, // 用户 ID
    tradeRemark: 'AI付接口测试',
    notifyUrl: 'https://merchant.example.com/notify/pay',
    currency: 'CNY', // 币种
    expiryTime: '604800', // 订单有效期（秒）
  };
  return JSON.stringify(biz);
}

async function main() {
  const pfx = parsePfxBase64(PFX_BASE64, PFX_PASSWORD);

  const bizJson = buildBizJson();
  const bizContent = encodeBizContent(bizJson, pfx, SM2_JD_PUB);

  const content = buildContent({
    bizContentEncrypted: bizContent,
    appId: APP_ID,
    agentId: AGENT_ID,
    merchantNo: MERCHANT_NO,
  });
  const signString = buildSignString(content);
  const sign = hmacSm3Hex(signString, SECRET_KEY);
  content.sign = sign;

  const body = JSON.stringify({ data: { content } });
  const headers = buildHttpHeaders(APP_ID);

  console.log('=================== bizContent 明文 ===================');
  console.log(bizJson);
  console.log('=================== 签名原文 ===================');
  console.log(signString);
  console.log('=================== 签名结果 ===================');
  console.log(sign);
  console.log('=================== HTTP Header ===================');
  for (const [k, v] of Object.entries(headers)) {
    console.log(`${k}:${v}`);
  }
  console.log('=================== HTTP Body ===================');
  console.log(body);

  const resp = await postJson(ENDPOINT_URL, headers, body);
  console.log('=================== HTTP Response ===================');
  console.log(resp);

  tryDecryptResponseBizContent(resp, pfx);
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
