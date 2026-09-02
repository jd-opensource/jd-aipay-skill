/**
 * AI 付 queryRefundResult 接口 Demo（Node.js 版）。
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

const SECRET_KEY = '__SECRET_KEY__';
const PFX_BASE64 = '__PFX_BASE64__';
const PFX_PASSWORD = '__PFX_PASSWORD__';

const ENDPOINT_URL = '__ENDPOINT_URL__';
const APP_ID = '__APP_ID__';
const AGENT_ID = '__AGENT_ID__';
const MERCHANT_NO = '__MERCHANT_NO__';
const ACQ_MERCHANT_NO = '__ACQ_MERCHANT_NO__';
const ACCESS_TYPE = '__ACCESS_TYPE__'; // 接入类型：SERVICE_MER 服务商 / COMMON 普通商户

// 业务参数
const REFUND_NO = '__REFUND_NO__';

function buildBizJson() {
  const biz = {
    acqMerchantNo: ACQ_MERCHANT_NO, // 收单商户号
    accessType: ACCESS_TYPE, // 接入类型：SERVICE_MER 服务商 / COMMON 普通商户
    refundNo: REFUND_NO, // 退款单号
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
