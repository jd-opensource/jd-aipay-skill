/**
 * AI 付接入配置（环境、凭证、密钥）。
 *
 * 所有可替换的值集中在工程根目录的 `aipay.env` 文件中（KEY=value 格式）。切换环境或更换
 * 凭证（密钥、证书、商户号、应用标识等）时，直接编辑 aipay.env 即可，无需修改任何 JS 代码、
 * 也无需重新生成工程。业务参数（订单号、用户、金额等）不在本文件，位于各 demo 中。
 *
 * 本模块导入时读取 aipay.env 并初始化导出对象；文件缺失或缺项会抛出明确异常。
 */
const fs = require('fs');
const path = require('path');

function parseEnvFile(file) {
  const vals = {};
  for (const raw of fs.readFileSync(file, 'utf-8').split('\n')) {
    const line = raw.trim();
    if (!line || line.startsWith('#')) continue;
    const idx = line.indexOf('=');
    if (idx < 0) continue;
    vals[line.slice(0, idx).trim()] = line.slice(idx + 1).trim();
  }
  return vals;
}

function loadEnvFile() {
  const candidates = [
    path.resolve(process.cwd(), 'aipay.env'),
    path.resolve(__dirname, '../../aipay.env'),
    path.resolve(__dirname, '../../../aipay.env'),
  ];
  for (const candidate of candidates) {
    try {
      if (fs.existsSync(candidate)) {
        return { vals: parseEnvFile(candidate), file: candidate };
      }
    } catch (e) {
      // 忽略单个候选路径的读取异常，继续尝试下一个
    }
  }
  throw new Error('未找到配置文件 aipay.env（应在工程根目录）。该文件承载全部可替换配置项，请勿删除或改名。');
}

const loaded = loadEnvFile();

function get(key) {
  const value = loaded.vals[key];
  if (value === undefined || value === '') {
    throw new Error(`aipay.env 缺少配置项: ${key}，请补齐后重试`);
  }
  return value;
}

module.exports = {
  ENV_FILE: loaded.file,
  ENV: get('AIPAY_ENV'), // pre | prod | sandbox
  ENDPOINT_URL: get('AIPAY_ENDPOINT_URL'), // 完整 endpoint URL（沙箱环境含沙箱实例 ID）
  APP_ID: get('AIPAY_APP_ID'), // 应用 ID
  AGENT_ID: get('AIPAY_AGENT_ID'), // 渠道/Agent 标识
  MERCHANT_NO: get('AIPAY_MERCHANT_NO'), // 商户号
  ACQ_MERCHANT_NO: get('AIPAY_ACQ_MERCHANT_NO'), // 收单商户号
  ACCESS_TYPE: get('AIPAY_ACCESS_TYPE'), // 接入类型：SERVICE_MER 服务商 / COMMON 普通商户
  SECRET_KEY: get('AIPAY_SECRET_KEY'), // HMAC-SM3 密钥
  PFX_PASSWORD: get('AIPAY_MERCHANT_PFX_PASSWORD'), // 商户 pfx 密码
  PFX_BASE64: get('AIPAY_MERCHANT_PFX'), // 商户 pfx Base64
  SM2_JD_PUB: get('AIPAY_PUBLIC_KEY'), // 京东 SM2 公钥证书 Base64（按环境：沙箱/共享）
};
