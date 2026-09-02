/**
 * AI 付网关公共辅助：HMAC-SM3 签名、content 组装、HTTP 调用、响应 bizContent 解密。
 */
'use strict';

const crypto = require('crypto');
const http = require('http');
const https = require('https');
const url = require('url');

const { signEnvelop, verifyEnvelop, smCrypto } = require('./crypto');
const { parsePfxBase64 } = require('./pfx');

const { sm3 } = smCrypto;

// 参与签名的字段（ASCII 升序）——与 Java / Python demo 保持一致
const SIGN_KEYS = [
  'agentId',
  'appId',
  'bizContent',
  'encType',
  'merchantNo',
  'nonce',
  'reqNo',
  'timestamp',
  'version',
];

function randomHex(length) {
  const chars = '0123456789ABCDEF';
  let out = '';
  for (let i = 0; i < length; i++) {
    out += chars[Math.floor(Math.random() * 16)];
  }
  return out;
}

function genReqNo() {
  // 32 位大写 hex（等价 Java UUID.randomUUID().toString().replace('-','').toUpperCase()）
  return crypto.randomBytes(16).toString('hex').toUpperCase();
}

function nowMs() {
  return String(Date.now());
}

/**
 * HMAC-SM3：按 RFC 2104 手写外壳（block_size = 64），底层哈希用 sm-crypto 的 sm3。
 * 返回小写 hex 字符串。
 */
function hmacSm3Hex(stringToSign, secretKey) {
  const BLOCK = 64;
  let key = Buffer.from(secretKey, 'utf8');
  const data = Buffer.from(stringToSign, 'utf8');
  if (key.length > BLOCK) {
    key = Buffer.from(sm3(Array.from(key)), 'hex');
  }
  if (key.length < BLOCK) {
    key = Buffer.concat([key, Buffer.alloc(BLOCK - key.length, 0)]);
  }
  const oPad = Buffer.alloc(BLOCK);
  const iPad = Buffer.alloc(BLOCK);
  for (let i = 0; i < BLOCK; i++) {
    oPad[i] = key[i] ^ 0x5c;
    iPad[i] = key[i] ^ 0x36;
  }
  const inner = Buffer.from(sm3(Array.from(Buffer.concat([iPad, data]))), 'hex');
  const outer = sm3(Array.from(Buffer.concat([oPad, inner])));
  return outer.toLowerCase();
}

/**
 * 按 ASCII 升序把 content 层字段拼成 k1=v1&k2=v2；空值不参与；sign/signType 不参与（signType 本身不在 SIGN_KEYS 内）。
 */
function buildSignString(content) {
  const parts = [];
  for (const key of SIGN_KEYS) {
    const val = content[key];
    if (val === undefined || val === null || val === '') continue;
    parts.push(`${key}=${val}`);
  }
  return parts.join('&');
}

/**
 * SM2 数字信封加密 bizContent，返回 Base64 字符串。
 */
function encodeBizContent(bizJson, pfx, jdCertB64) {
  return signEnvelop(Buffer.from(bizJson, 'utf8'), pfx, jdCertB64);
}

/**
 * SM2 数字信封解密 bizContent，返回明文字符串。
 */
function decodeBizContent(bizContentB64, pfx) {
  return verifyEnvelop(bizContentB64, pfx).toString('utf8');
}

/**
 * 组装 content 层字段。
 */
function buildContent({ bizContentEncrypted, appId, agentId, merchantNo }) {
  return {
    agentId,
    appId,
    bizContent: bizContentEncrypted,
    encType: 'SM2',
    merchantNo,
    nonce: randomHex(16),
    reqNo: genReqNo(),
    signType: 'SM3',
    timestamp: nowMs(),
    version: '1.0',
  };
}

/**
 * 构造 HTTP header。所有 key 都是全小写——网关对 header 名大小写敏感。
 */
function buildHttpHeaders(appId) {
  return {
    'app-id': appId,
    'encrypt-type': 'NONE',
    'source-type': 'H5',
    'login-type': '0',
    'cache-control': 'no-cache',
    'content-type': 'application/json',
    'stream-type': 'false',
  };
}

/**
 * 发起 HTTPS/HTTP POST。
 *
 * ⚠️ 关键陷阱：AI 付网关对 HTTP header 名**大小写敏感**（必须是全小写 `stream-type`
 *   / `app-id` / `encrypt-type` / `source-type` / `login-type` 等）。
 *   Node.js 的 http/https 模块通过 options.headers 传对象时会**保留 key 原始大小写**，
 *   但某些代理/框架封装（如 axios、undici 默认代理）会把 header 名规范化为首字母大写，
 *   导致网关识别不到 `stream-type=false`，把请求当成 SSE 流式请求处理，走另一条
 *   鉴权链路，返回 `10000403 资源受权信息缺失`。
 *
 * 因此这里直接用底层 http/https 模块，避免任何 header 规范化。
 * 同时关闭 TLS 校验（`rejectUnauthorized: false`），与 Java / Python demo 行为对齐。
 */
function postJson(endpoint, headers, bodyJson, { timeoutMs = 30000 } = {}) {
  return new Promise((resolve, reject) => {
    const parsed = url.parse(endpoint);
    const isHttps = parsed.protocol === 'https:';
    const bodyBuf = Buffer.from(bodyJson, 'utf8');
    const options = {
      method: 'POST',
      hostname: parsed.hostname,
      port: parsed.port || (isHttps ? 443 : 80),
      path: (parsed.pathname || '/') + (parsed.search || ''),
      headers: {
        ...headers,
        'content-length': String(bodyBuf.length),
      },
      timeout: timeoutMs,
    };
    if (isHttps) options.rejectUnauthorized = false;

    const lib = isHttps ? https : http;
    const req = lib.request(options, (res) => {
      const chunks = [];
      res.on('data', (c) => chunks.push(c));
      res.on('end', () => {
        const body = Buffer.concat(chunks).toString('utf8');
        resolve(`HTTP ${res.statusCode} | ${body}`);
      });
    });
    req.on('error', (err) => reject(err));
    req.on('timeout', () => {
      req.destroy(new Error(`request timeout after ${timeoutMs}ms`));
    });
    req.write(bodyBuf);
    req.end();
  });
}

/**
 * 尝试解密响应 bizContent 并打印。
 *
 * 网关成功响应（`code=00000`）时 `data.content` 是 **JSON 字符串**（需再 JSON.parse 一次拿到内层 bizContent/sign）；
 * 失败响应（如 10000403）时 `data.content` 直接是 object 或空串。需兼容两种形态。
 */
function tryDecryptResponseBizContent(responseText, pfx) {
  const idx = responseText.indexOf('{');
  if (idx < 0) return;
  let root;
  try {
    root = JSON.parse(responseText.substring(idx));
  } catch (ex) {
    console.log(`[响应 bizContent 解密失败] 非 JSON 响应: ${ex.message}`);
    return;
  }
  let contentObj = root && root.data ? root.data.content : null;
  if (contentObj === null || contentObj === undefined) return;
  if (typeof contentObj === 'string') {
    try {
      contentObj = JSON.parse(contentObj);
    } catch (_ex) {
      return;
    }
  }
  if (typeof contentObj !== 'object') return;
  const bizContent = contentObj.bizContent;
  const encType = contentObj.encType || contentObj.encryptType;
  if (!bizContent) return;
  console.log('=================== 响应 encType ===================');
  console.log(encType);
  console.log('=================== 响应 bizContent 明文 ===================');
  try {
    if ((encType || '').toUpperCase() === 'SM2') {
      console.log(decodeBizContent(bizContent, pfx));
    } else {
      console.log(Buffer.from(bizContent, 'base64').toString('utf8'));
    }
  } catch (ex) {
    console.log(`[响应 bizContent 解密失败] ${ex.message}`);
  }
}

module.exports = {
  parsePfxBase64,
  randomHex,
  genReqNo,
  nowMs,
  hmacSm3Hex,
  buildSignString,
  encodeBizContent,
  decodeBizContent,
  buildContent,
  buildHttpHeaders,
  postJson,
  tryDecryptResponseBizContent,
};
