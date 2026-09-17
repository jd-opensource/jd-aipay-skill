/**
 * AI 付 refund 接口 Demo（Node.js 版）。
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

const config = require('./config');




// 业务参数
const ORIGINAL_OUT_TRADE_NO = '__ORIGINAL_OUT_TRADE_NO__';
const REFUND_AMOUNT = __REFUND_AMOUNT__; // int 字面量（不加引号）

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

/**
 * REFUND + yyyyMMddHHmmss + 3 位随机数字，保证幂等。
 */
function genRefundNo() {
  const rand = Math.floor(Math.random() * 1000).toString().padStart(3, '0');
  return 'REFUND' + nowYyyyMmDdHhMmSs() + rand;
}

function buildBizJson() {
  const biz = {
    acqMerchantNo: config.ACQ_MERCHANT_NO, // 收单商户号
    accessType: config.ACCESS_TYPE, // 接入类型：SERVICE_MER 服务商 / COMMON 普通商户
    originalOutTradeNo: ORIGINAL_OUT_TRADE_NO, // 原下单商户订单号
    refundNo: genRefundNo(), // 退款单号（幂等键）
    refundAmount: REFUND_AMOUNT, // 退款金额（分）
    currency: 'CNY', // 币种
    refundReason: 'AI付退款测试', // 退款原因
  };
  return JSON.stringify(biz);
}

async function main() {
  const pfx = parsePfxBase64(config.PFX_BASE64, config.PFX_PASSWORD);
  const bizJson = buildBizJson();
  const bizContent = encodeBizContent(bizJson, pfx, config.SM2_JD_PUB);

  const content = buildContent({
    bizContentEncrypted: bizContent,
    appId: config.APP_ID,
    agentId: config.AGENT_ID,
    merchantNo: config.MERCHANT_NO,
  });
  const signString = buildSignString(content);
  const sign = hmacSm3Hex(signString, config.SECRET_KEY);
  content.sign = sign;

  const body = JSON.stringify({ data: { content } });
  const headers = buildHttpHeaders(config.APP_ID);

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

  const resp = await postJson(config.ENDPOINT_URL, headers, body);
  console.log('=================== HTTP Response ===================');
  console.log(resp);

  tryDecryptResponseBizContent(resp, pfx);
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
