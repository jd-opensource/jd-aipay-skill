/**
 * 京东 wyaks-security 兼容的 SM2 数字信封与 CMS SM3WithSM2 签名（Node.js 版）。
 *
 * 信封结构（非标 CMS，见 SM2EnvelopUtil2）：
 *   Base64( version(1B=0x01) || DER(SEQ(c11 INTEGER, c12 INTEGER, c3 OCTET STRING, c2 OCTET STRING)) || IV(16B) || SM4-CBC-PKCS7(P7Sign, key, iv) )
 *   其中 key/iv 各 16 字节随机；c11=C1.X, c12=C1.Y, c2=对称加密载荷, c3=SM3 MAC
 *
 * P7 签名（SM3WithSM2, directSignature=true, attachFlag=true）：
 *   ContentInfo(signedData) {
 *     SignedData {
 *       version = 1,
 *       digestAlgorithms = { SM3 (1.2.156.10197.1.401) },
 *       encapContentInfo = { contentType = data(1.2.840.113549.1.7.1), content OCTET STRING = srcData },
 *       certificates = [商户证书],
 *       signerInfos = [{
 *         version = 1,
 *         sid = IssuerAndSerialNumber,
 *         digestAlgorithm = SM3,
 *         signatureAlgorithm = SM3WithSM2 (1.2.156.10197.1.501),
 *         signature = SM2Sign(srcData, ZA)  // directSignature=true 时无 signed attributes
 *       }]
 *     }
 *   }
 *
 * 依赖 sm-crypto 提供 SM2/SM3/SM4 基元；ASN.1 DER 由本模块内 der.js 手写。
 */
'use strict';

const crypto = require('crypto');
const smCrypto = require('sm-crypto');
const { sm2, sm3, sm4 } = smCrypto;

const {
  TAG,
  readTLV,
  readChildren,
  encodeTLV,
  encodeSequence,
  encodeSet,
  encodeOctetString,
  encodeInteger,
  encodeOid,
  decodeIntegerUnsigned,
} = require('./der');

const OID_SM3 = '1.2.156.10197.1.401';
const OID_SM3_WITH_SM2 = '1.2.156.10197.1.501';
const OID_DATA = '1.2.840.113549.1.7.1';
const OID_SIGNED_DATA = '1.2.840.113549.1.7.2';

const SM2_USER_ID = '1234567812345678';

// ---------------------------------------------------------------------------
// 辅助：从证书 DER 中提取 65 字节未压缩 SM2 公钥点
// ---------------------------------------------------------------------------

function extractSm2PubFromCertDer(certDer) {
  // Certificate ::= SEQUENCE { tbsCertificate, signatureAlgorithm, signatureValue }
  const cert = readTLV(certDer);
  const tbsAndRest = readChildren(cert.content);
  const tbs = tbsAndRest[0];
  const tbsChildren = readChildren(tbs.content);
  // tbsCertificate SEQUENCE：[版本(可选[0])] serialNumber signature issuer validity subject subjectPublicKeyInfo ...
  // 找到 SubjectPublicKeyInfo：它是最后一个 SEQUENCE 前的一个，格式最稳的做法是找 BIT STRING 所在 SEQUENCE
  let spki;
  for (const c of tbsChildren) {
    if (c.tag === TAG.SEQUENCE) {
      const inner = readChildren(c.content);
      if (inner.length === 2 && inner[1].tag === TAG.BIT_STRING) {
        spki = c;
      }
    }
  }
  if (!spki) throw new Error('SubjectPublicKeyInfo not found');
  const spkiChildren = readChildren(spki.content);
  const bitString = spkiChildren[1].content;
  // BIT STRING 首字节是 unused bits count；SM2 公钥无 unused bits
  if (bitString[0] !== 0x00) throw new Error('unexpected BIT STRING unused bits: ' + bitString[0]);
  const pub = bitString.subarray(1);
  if (pub.length !== 65 || pub[0] !== 0x04) {
    throw new Error('unexpected SM2 pubkey format, len=' + pub.length);
  }
  return pub;
}

/**
 * 从 tbsCertificate 中提取 issuer（DER SEQUENCE）与 serialNumber（Buffer 无符号）。
 * 用于构造 CMS SignerInfo 的 IssuerAndSerialNumber。
 */
function extractIssuerAndSerial(certDer) {
  const cert = readTLV(certDer);
  const tbs = readTLV(cert.content);
  const off0 = tbs.contentOffset;
  const tbsChildren = readChildren(tbs.content);
  // 结构（v3）：[0]{version}, serialNumber INTEGER, signature AlgId, issuer Name, validity, subject Name, subjectPublicKeyInfo, ...
  // v1 无 [0] version 标签
  let idx = 0;
  // 有的话 tbsChildren[0] 是 [0] EXPLICIT version
  if (tbsChildren[0].tag === 0xa0) idx = 1;
  const serialTlv = tbsChildren[idx];
  const issuerTlv = tbsChildren[idx + 2];
  // 序列化时保留原始 DER
  const serial = decodeIntegerUnsigned(serialTlv.content);
  // issuer 需要完整 TLV（含 tag/length）
  const issuerFullOffset = tbs.content === cert.content.subarray(0, tbs.contentOffset - tbs.headerLen)
    ? issuerTlv.contentOffset
    : issuerTlv.contentOffset;
  // 简化：直接用 subarray 从 tbs.content 内拿 issuer 的完整 TLV
  const issuerStart = issuerTlv.contentOffset - issuerTlv.headerLen;
  const issuerEnd = issuerTlv.next;
  const issuerDer = tbs.content.subarray(issuerStart, issuerEnd);
  return { issuerDer: Buffer.from(issuerDer), serial };
}

// ---------------------------------------------------------------------------
// 基元封装
// ---------------------------------------------------------------------------

function sm3Hash(data) {
  const arr = sm3(Array.from(data));
  return Buffer.from(arr, 'hex');
}

function sm4CbcEncryptPkcs7(key, iv, data) {
  const hex = sm4.encrypt(Array.from(data), Array.from(key), {
    padding: 'pkcs#7',
    mode: 'cbc',
    iv: Array.from(iv),
    output: 'string',
  });
  return Buffer.from(hex, 'hex');
}

function sm4CbcDecryptPkcs7(key, iv, data) {
  // output=array 返回字节数组；不用 'string'（内部会走 UTF-8 解码，二进制/密钥数据会抛错）
  const arr = sm4.decrypt(Array.from(data), Array.from(key), {
    padding: 'pkcs#7',
    mode: 'cbc',
    iv: Array.from(iv),
    output: 'array',
  });
  return Buffer.from(arr);
}

/**
 * SM2 公钥加密。pub 是 65B 未压缩点（含 0x04 前缀）。
 * 使用 cipherMode=1 (C1C3C2)，方便直接取 C1(64B)、C3(32B)、C2(rest)。
 * 返回 { c1: Buffer(64), c3: Buffer(32), c2: Buffer }
 */
function sm2Encrypt(pub, data) {
  // sm-crypto 期望 130 hex（带 04 前缀）
  const pubHex = pub.toString('hex');
  const cipherHex = sm2.doEncrypt(Array.from(data), pubHex, 1 /* C1C3C2 */);
  const c1 = Buffer.from(cipherHex.slice(0, 128), 'hex');
  const c3 = Buffer.from(cipherHex.slice(128, 192), 'hex');
  const c2 = Buffer.from(cipherHex.slice(192), 'hex');
  return { c1, c3, c2 };
}

/**
 * SM2 私钥解密。dHex 是 32B 私钥标量的 hex。cipherC1C3C2 是 C1(64) || C3(32) || C2 结构（无 04 前缀）。
 * 返回原始字节 Buffer（不做 UTF-8 转换，因为解密对象可能是任意二进制，如 16B 会话密钥）。
 */
function sm2Decrypt(dHex, cipherC1C3C2) {
  const plainArr = sm2.doDecrypt(cipherC1C3C2.toString('hex'), dHex, 1 /* C1C3C2 */, {
    output: 'array',
  });
  return Buffer.from(plainArr);
}

/**
 * SM2 SM3WithSM2 签名（内部计算 ZA + e），返回 DER SEQUENCE(r,s) Buffer。
 * pfx.privateKeyD: 32B Buffer；pfx.certDer: 证书 DER Buffer。
 */
function sm2SignSm3(srcData, pfx) {
  const pub = extractSm2PubFromCertDer(pfx.certDer);
  const dHex = pfx.privateKeyD.toString('hex');
  const pubHex = pub.toString('hex'); // 带 04 前缀，130 hex
  // hash=true → 内部计算 e = SM3(ZA || M)；der=true → 输出 DER SEQ(r,s) hex
  const sigHex = sm2.doSignature(Array.from(srcData), dHex, {
    hash: true,
    der: true,
    userId: SM2_USER_ID,
    publicKey: pubHex,
  });
  return Buffer.from(sigHex, 'hex');
}

// ---------------------------------------------------------------------------
// SM2 数字信封（京东私有格式）
// ---------------------------------------------------------------------------

function sm2EncryptEnvelop(data, jdCertB64) {
  const key = crypto.randomBytes(16);
  const iv = crypto.randomBytes(16);
  const pub = extractSm2PubFromCertDer(Buffer.from(jdCertB64, 'base64'));

  const { c1, c3, c2 } = sm2Encrypt(pub, key);
  const c11 = c1.subarray(0, 32);
  const c12 = c1.subarray(32, 64);

  // DER SEQ(INTEGER c11, INTEGER c12, OCTET STRING c3, OCTET STRING c2)
  const envelopKey = encodeSequence([
    encodeInteger(c11.toString('hex')),
    encodeInteger(c12.toString('hex')),
    encodeOctetString(c3),
    encodeOctetString(c2),
  ]);

  const endata = sm4CbcEncryptPkcs7(key, iv, data);
  const result = Buffer.concat([Buffer.from([0x01]), envelopKey, iv, endata]);
  return result.toString('base64');
}

function sm2DecryptEnvelop(envelopB64, dHex) {
  const buf = Buffer.from(envelopB64, 'base64');
  if (buf[0] !== 0x01) throw new Error('unknown envelop version: ' + buf[0]);
  // 从 offset=1 起是 SEQUENCE
  const seqTlv = readTLV(buf, 1);
  const enkeyEnd = seqTlv.next;
  const seqChildren = readChildren(seqTlv.content);
  const c11 = decodeIntegerUnsigned(seqChildren[0].content, 32);
  const c12 = decodeIntegerUnsigned(seqChildren[1].content, 32);
  const c3 = seqChildren[2].content;
  const c2 = seqChildren[3].content;

  const iv = buf.subarray(enkeyEnd, enkeyEnd + 16);
  const endata = buf.subarray(enkeyEnd + 16);

  const cipherC1C3C2 = Buffer.concat([c11, c12, c3, c2]);
  const key = sm2Decrypt(dHex, cipherC1C3C2);
  const plain = sm4CbcDecryptPkcs7(key, iv, endata);
  return plain;
}

// ---------------------------------------------------------------------------
// CMS SignedData (SM3WithSM2, directSignature=true, attachFlag=true)
// ---------------------------------------------------------------------------

function encodeAlgorithmIdentifier(oid, params) {
  // SEQUENCE { algorithm OID, parameters ANY DEFINED BY algorithm OPTIONAL }
  const children = [encodeOid(oid)];
  if (params !== undefined) children.push(params);
  return encodeSequence(children);
}

function sm2P7SignAttached(srcData, pfx) {
  const signature = sm2SignSm3(srcData, pfx);
  const { issuerDer, serial } = extractIssuerAndSerial(pfx.certDer);

  // digestAlgorithm SEQ { OID sm3 }（不带 NULL 参数，与 BC directSignature 输出对齐）
  const digestAlgo = encodeAlgorithmIdentifier(OID_SM3);
  const signatureAlgo = encodeAlgorithmIdentifier(OID_SM3_WITH_SM2);

  // IssuerAndSerialNumber ::= SEQUENCE { issuer Name, serialNumber INTEGER }
  const issuerAndSerial = encodeSequence([issuerDer, encodeInteger(serial)]);

  // SignerInfo ::= SEQUENCE {
  //   version 1,
  //   sid IssuerAndSerialNumber,
  //   digestAlgorithm AlgId,
  //   digestEncryptionAlgorithm(=signatureAlgorithm) AlgId,
  //   encryptedDigest(=signature) OCTET STRING
  // }
  const signerInfo = encodeSequence([
    encodeInteger(1),
    issuerAndSerial,
    digestAlgo,
    signatureAlgo,
    encodeOctetString(signature),
  ]);

  // EncapsulatedContentInfo ::= SEQUENCE {
  //   contentType OID = data,
  //   content [0] EXPLICIT OCTET STRING srcData OPTIONAL
  // }
  const eContent = encodeTLV(0xa0 /* [0] EXPLICIT */, encodeOctetString(srcData));
  const encapContentInfo = encodeSequence([encodeOid(OID_DATA), eContent]);

  // certificates [0] IMPLICIT CertificateSet —— 直接内嵌 X.509 DER
  const certificatesField = encodeTLV(0xa0 /* [0] IMPLICIT SET */, pfx.certDer);

  // SignedData ::= SEQUENCE {
  //   version 1,
  //   digestAlgorithms SET OF AlgId,
  //   encapContentInfo,
  //   certificates [0] IMPLICIT CertificateSet OPTIONAL,
  //   crls [1] IMPLICIT ... OPTIONAL,   -- 省略
  //   signerInfos SET OF SignerInfo
  // }
  const signedData = encodeSequence([
    encodeInteger(1),
    encodeSet([digestAlgo]),
    encapContentInfo,
    certificatesField,
    encodeSet([signerInfo]),
  ]);

  // ContentInfo ::= SEQUENCE {
  //   contentType OID = signedData,
  //   content [0] EXPLICIT SignedData
  // }
  const contentInfo = encodeSequence([
    encodeOid(OID_SIGNED_DATA),
    encodeTLV(0xa0, signedData),
  ]);
  return contentInfo;
}

/**
 * 从 CMS SignedData DER 中剥出 encapContentInfo.eContent 明文。
 */
function cmsExtractContent(cmsDer) {
  const contentInfo = readTLV(cmsDer);
  const ciChildren = readChildren(contentInfo.content);
  // ciChildren[1] = [0] EXPLICIT SignedData
  const signedData = readTLV(ciChildren[1].content);
  const sdChildren = readChildren(signedData.content);
  // sdChildren: [version, digestAlgorithms SET, encapContentInfo SEQ, ...]
  const encap = sdChildren[2];
  const encapChildren = readChildren(encap.content);
  // encapChildren[0] = contentType OID, encapChildren[1] = [0] EXPLICIT OCTET STRING (optional)
  if (!encapChildren[1]) return Buffer.alloc(0);
  const eContentExplicit = readTLV(encapChildren[1].content);
  // eContentExplicit 是 OCTET STRING
  if (eContentExplicit.tag === TAG.OCTET_STRING) {
    return Buffer.from(eContentExplicit.content);
  }
  return Buffer.from(encapChildren[1].content);
}

/**
 * 入口：SM2 数字信封（内含 CMS P7 签名），等价 Java signEnvelop。
 */
function signEnvelop(srcData, pfx, jdCertB64) {
  const p7 = sm2P7SignAttached(srcData, pfx);
  return sm2EncryptEnvelop(p7, jdCertB64);
}

/**
 * 入口：解京东 SM2 信封，剥出 CMS srcData（等价 Java verifyEnvelop 拿第 0 项）。
 */
function verifyEnvelop(envelopB64, pfx) {
  const p7 = sm2DecryptEnvelop(envelopB64, pfx.privateKeyD.toString('hex'));
  return cmsExtractContent(p7);
}

module.exports = {
  sm2EncryptEnvelop,
  sm2DecryptEnvelop,
  sm2P7SignAttached,
  cmsExtractContent,
  signEnvelop,
  verifyEnvelop,
  // 供 common.js 复用
  sm3Hash,
  smCrypto,
};
