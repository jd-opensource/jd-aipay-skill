/**
 * AI 付 queryPayResult 接口 Demo（Node.js 版）。
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
const OUT_TRADE_NO = '__OUT_TRADE_NO__';

function buildBizJson() {
  const biz = {
    acqMerchantNo: config.ACQ_MERCHANT_NO, // 收单商户号
    accessType: config.ACCESS_TYPE, // 接入类型：SERVICE_MER 服务商 / COMMON 普通商户
    outTradeNo: OUT_TRADE_NO, // 商户外部订单号
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
