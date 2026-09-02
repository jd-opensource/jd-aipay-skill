/**
 * PKCS#12 (PFX) 解析：从商户 PFX 文件中提取 SM2 私钥标量 d（32B 大端）与 X.509 证书 DER。
 *
 * Node 内建 crypto / 主流库（node-forge、pkijs）均不支持 SM2 曲线（OID 1.2.156.10197.1.301），
 * 因此本模块手工解 PFX ASN.1 结构：
 *   1. Base64 -> PFX bytes
 *   2. 手工解 Pfx / AuthenticatedSafe / SafeContents / SafeBag（DER 遍历）
 *   3. 对加密的 SafeBag：识别 PBE 算法（3DES-CBC / RC2-40-CBC / PBES2(AES)），派生密钥后解密
 *   4. 从 KeyBag 或 PKCS8ShroudedKeyBag 提取 ECPrivateKey.privateKey（32B 大端 d）
 *   5. 从 CertBag 提取 X.509 证书 DER
 *
 * 与 Python 版 pfx.py 行为一致（对齐 pbeWithSHAAnd3-KeyTripleDES-CBC / pbeWithSHAAnd40BitRC2-CBC / PBES2 三档）。
 */
'use strict';

const crypto = require('crypto');
const { readTLV, readChildren, decodeOid, decodeIntegerUnsigned, TAG } = require('./der');

// PKCS#12 SafeBag ID
const OID_KEY_BAG = '1.2.840.113549.1.12.10.1.1';
const OID_PKCS8_SHROUDED_KEY_BAG = '1.2.840.113549.1.12.10.1.2';
const OID_CERT_BAG = '1.2.840.113549.1.12.10.1.3';

const OID_CT_DATA = '1.2.840.113549.1.7.1';
const OID_CT_ENCRYPTED_DATA = '1.2.840.113549.1.7.6';
const OID_X509_CERTIFICATE = '1.2.840.113549.1.9.22.1';

// PBE 算法
const OID_PBE_3DES = '1.2.840.113549.1.12.1.3'; // pbeWithSHAAnd3-KeyTripleDES-CBC
const OID_PBE_RC2_40 = '1.2.840.113549.1.12.1.6'; // pbeWithSHAAnd40BitRC2-CBC
const OID_PBES2 = '1.2.840.113549.1.5.13';
const OID_PBKDF2 = '1.2.840.113549.1.5.12';

const OID_HMAC_SHA1 = '1.2.840.113549.2.7';
const OID_HMAC_SHA256 = '1.2.840.113549.2.9';
const OID_HMAC_SHA512 = '1.2.840.113549.2.11';

const OID_AES128_CBC = '2.16.840.1.101.3.4.1.2';
const OID_AES192_CBC = '2.16.840.1.101.3.4.1.22';
const OID_AES256_CBC = '2.16.840.1.101.3.4.1.42';

// ---------------------------------------------------------------------------
// RC2 40-bit pure-JS 实现（RFC 2268）
// Node 内建 crypto 不含 RC2；BC 生成 PFX 的 cert bag 常用 pbeWithSHAAnd40BitRC2-CBC。
// ---------------------------------------------------------------------------

const RC2_PITABLE = Uint8Array.from([
  0xd9, 0x78, 0xf9, 0xc4, 0x19, 0xdd, 0xb5, 0xed, 0x28, 0xe9, 0xfd, 0x79, 0x4a, 0xa0, 0xd8, 0x9d,
  0xc6, 0x7e, 0x37, 0x83, 0x2b, 0x76, 0x53, 0x8e, 0x62, 0x4c, 0x64, 0x88, 0x44, 0x8b, 0xfb, 0xa2,
  0x17, 0x9a, 0x59, 0xf5, 0x87, 0xb3, 0x4f, 0x13, 0x61, 0x45, 0x6d, 0x8d, 0x09, 0x81, 0x7d, 0x32,
  0xbd, 0x8f, 0x40, 0xeb, 0x86, 0xb7, 0x7b, 0x0b, 0xf0, 0x95, 0x21, 0x22, 0x5c, 0x6b, 0x4e, 0x82,
  0x54, 0xd6, 0x65, 0x93, 0xce, 0x60, 0xb2, 0x1c, 0x73, 0x56, 0xc0, 0x14, 0xa7, 0x8c, 0xf1, 0xdc,
  0x12, 0x75, 0xca, 0x1f, 0x3b, 0xbe, 0xe4, 0xd1, 0x42, 0x3d, 0xd4, 0x30, 0xa3, 0x3c, 0xb6, 0x26,
  0x6f, 0xbf, 0x0e, 0xda, 0x46, 0x69, 0x07, 0x57, 0x27, 0xf2, 0x1d, 0x9b, 0xbc, 0x94, 0x43, 0x03,
  0xf8, 0x11, 0xc7, 0xf6, 0x90, 0xef, 0x3e, 0xe7, 0x06, 0xc3, 0xd5, 0x2f, 0xc8, 0x66, 0x1e, 0xd7,
  0x08, 0xe8, 0xea, 0xde, 0x80, 0x52, 0xee, 0xf7, 0x84, 0xaa, 0x72, 0xac, 0x35, 0x4d, 0x6a, 0x2a,
  0x96, 0x1a, 0xd2, 0x71, 0x5a, 0x15, 0x49, 0x74, 0x4b, 0x9f, 0xd0, 0x5e, 0x04, 0x18, 0xa4, 0xec,
  0xc2, 0xe0, 0x41, 0x6e, 0x0f, 0x51, 0xcb, 0xcc, 0x24, 0x91, 0xaf, 0x50, 0xa1, 0xf4, 0x70, 0x39,
  0x99, 0x7c, 0x3a, 0x85, 0x23, 0xb8, 0xb4, 0x7a, 0xfc, 0x02, 0x36, 0x5b, 0x25, 0x55, 0x97, 0x31,
  0x2d, 0x5d, 0xfa, 0x98, 0xe3, 0x8a, 0x92, 0xae, 0x05, 0xdf, 0x29, 0x10, 0x67, 0x6c, 0xba, 0xc9,
  0xd3, 0x00, 0xe6, 0xcf, 0xe1, 0x9e, 0xa8, 0x2c, 0x63, 0x16, 0x01, 0x3f, 0x58, 0xe2, 0x89, 0xa9,
  0x0d, 0x38, 0x34, 0x1b, 0xab, 0x33, 0xff, 0xb0, 0xbb, 0x48, 0x0c, 0x5f, 0xb9, 0xb1, 0xcd, 0x2e,
  0xc5, 0xf3, 0xdb, 0x47, 0xe5, 0xa5, 0x9c, 0x77, 0x0a, 0xa6, 0x20, 0x68, 0xfe, 0x7f, 0xc1, 0xad,
]);

function rc2KeySchedule(key, effectiveBits) {
  const T = key.length;
  const T1 = effectiveBits;
  const T8 = (T1 + 7) >> 3;
  const TM = 0xff >> (8 * T8 - T1);
  const L = new Uint8Array(128);
  for (let i = 0; i < T; i++) L[i] = key[i];
  for (let i = T; i < 128; i++) L[i] = RC2_PITABLE[(L[i - 1] + L[i - T]) & 0xff];
  L[128 - T8] = RC2_PITABLE[L[128 - T8] & TM];
  for (let i = 127 - T8; i >= 0; i--) L[i] = RC2_PITABLE[L[i + 1] ^ L[i + T8]];
  const K = new Uint16Array(64);
  for (let i = 0; i < 64; i++) K[i] = (L[2 * i] | (L[2 * i + 1] << 8)) & 0xffff;
  return K;
}

function rc2DecryptBlock(K, block8) {
  const R = [
    (block8[0] | (block8[1] << 8)) & 0xffff,
    (block8[2] | (block8[3] << 8)) & 0xffff,
    (block8[4] | (block8[5] << 8)) & 0xffff,
    (block8[6] | (block8[7] << 8)) & 0xffff,
  ];
  const s = [1, 2, 3, 5];
  let j = 63;

  function ror16(v, r) {
    return ((v >> r) | (v << (16 - r))) & 0xffff;
  }

  function invMix() {
    for (let i = 3; i >= 0; i--) {
      R[i] = ror16(R[i], s[i]);
      R[i] =
        (R[i] -
          K[j] -
          (R[(i - 1 + 4) % 4] & R[(i - 2 + 4) % 4]) -
          (~R[(i - 1 + 4) % 4] & R[(i - 3 + 4) % 4])) &
        0xffff;
      j--;
    }
  }

  function invMash() {
    for (let i = 3; i >= 0; i--) {
      R[i] = (R[i] - K[R[(i - 1 + 4) % 4] & 63]) & 0xffff;
    }
  }

  for (let r = 0; r < 5; r++) invMix();
  invMash();
  for (let r = 0; r < 6; r++) invMix();
  invMash();
  for (let r = 0; r < 5; r++) invMix();

  const out = Buffer.alloc(8);
  for (let i = 0; i < 4; i++) {
    out[2 * i] = R[i] & 0xff;
    out[2 * i + 1] = (R[i] >> 8) & 0xff;
  }
  return out;
}

function rc2CbcDecryptPkcs7(key, iv, ciphertext, effectiveBits) {
  const K = rc2KeySchedule(key, effectiveBits);
  const out = Buffer.alloc(ciphertext.length);
  let prev = iv;
  for (let i = 0; i < ciphertext.length; i += 8) {
    const block = ciphertext.subarray(i, i + 8);
    const dec = rc2DecryptBlock(K, block);
    for (let k = 0; k < 8; k++) out[i + k] = dec[k] ^ prev[k];
    prev = block;
  }
  const padLen = out[out.length - 1];
  return out.subarray(0, out.length - padLen);
}

// ---------------------------------------------------------------------------
// PKCS#12 PBE 密钥派生（RFC 7292 附录 B，SHA-1 + iteration）
// ---------------------------------------------------------------------------

/**
 * PKCS12 密钥派生：purpose 1=key, 2=IV, 3=MAC；u=20 (SHA-1), v=64 block。
 */
function pkcs12PbeDerive(password, salt, iterations, purpose, keyLen) {
  // UTF-16BE + NUL 结尾
  let pwdBytes;
  if (password && password.length > 0) {
    const enc = Buffer.alloc(password.length * 2);
    for (let i = 0; i < password.length; i++) {
      const code = password.charCodeAt(i);
      enc[2 * i] = (code >> 8) & 0xff;
      enc[2 * i + 1] = code & 0xff;
    }
    pwdBytes = Buffer.concat([enc, Buffer.from([0, 0])]);
  } else {
    pwdBytes = Buffer.from([0, 0]);
  }

  const u = 20;
  const v = 64;
  const D = Buffer.alloc(v, purpose);

  function fill(src, length) {
    if (src.length === 0) return Buffer.alloc(0);
    const n = Math.ceil(length / src.length);
    return Buffer.concat(Array.from({ length: n }, () => src)).subarray(0, length);
  }

  const sLen = Math.ceil(salt.length / v) * v;
  const pLen = Math.ceil(pwdBytes.length / v) * v;
  const S = fill(salt, sLen);
  const P = fill(pwdBytes, pLen);
  let I = Buffer.concat([S, P]);

  const c = Math.ceil(keyLen / u);
  const chunks = [];
  for (let idx = 0; idx < c; idx++) {
    let A = crypto.createHash('sha1').update(Buffer.concat([D, I])).digest();
    for (let it = 1; it < iterations; it++) {
      A = crypto.createHash('sha1').update(A).digest();
    }
    chunks.push(A);
    // 更新 I：把 A 铺满 v，然后每 v 块 = (I_j + B + 1) mod 2^(8v)
    const B = fill(A, v);
    const bBig = BigInt('0x' + B.toString('hex'));
    const mask = (1n << BigInt(8 * v)) - 1n;
    const newI = Buffer.alloc(I.length);
    for (let j = 0; j < I.length; j += v) {
      const chunk = BigInt('0x' + I.subarray(j, j + v).toString('hex'));
      let summed = (chunk + bBig + 1n) & mask;
      // 写回 v 字节大端
      for (let k = v - 1; k >= 0; k--) {
        newI[j + k] = Number(summed & 0xffn);
        summed >>= 8n;
      }
    }
    I = newI;
  }
  return Buffer.concat(chunks).subarray(0, keyLen);
}

function tripleDesCbcDecryptPkcs7(key, iv, ct) {
  // Node 内建 des-ede3-cbc 天然支持 24B key + 8B IV，自动去 PKCS#7 填充
  const d = crypto.createDecipheriv('des-ede3-cbc', key, iv);
  d.setAutoPadding(true);
  return Buffer.concat([d.update(ct), d.final()]);
}

function aesCbcDecryptPkcs7(key, iv, ct) {
  const algo = key.length === 32 ? 'aes-256-cbc' : key.length === 24 ? 'aes-192-cbc' : 'aes-128-cbc';
  const d = crypto.createDecipheriv(algo, key, iv);
  d.setAutoPadding(true);
  return Buffer.concat([d.update(ct), d.final()]);
}

/**
 * 解密 SafeBag 加密内容。encryptionAlgorithm TLV 为 AlgorithmIdentifier SEQUENCE。
 */
function decryptPBE(encryptionAlgorithmTlv, encryptedData, password) {
  const children = readChildren(encryptionAlgorithmTlv.content);
  const oid = decodeOid(children[0].content);
  const paramsTlv = children[1];

  if (oid === OID_PBE_3DES || oid === OID_PBE_RC2_40) {
    // pkcs-12PbeParams ::= SEQUENCE { salt OCTET STRING, iterations INTEGER }
    const paramsChildren = readChildren(paramsTlv.content);
    const salt = paramsChildren[0].content;
    const iterations = Number(BigInt('0x' + paramsChildren[1].content.toString('hex')));

    if (oid === OID_PBE_3DES) {
      const key = pkcs12PbeDerive(password, salt, iterations, 1, 24);
      const iv = pkcs12PbeDerive(password, salt, iterations, 2, 8);
      return tripleDesCbcDecryptPkcs7(key, iv, encryptedData);
    }
    // RC2-40
    const key = pkcs12PbeDerive(password, salt, iterations, 1, 5);
    const iv = pkcs12PbeDerive(password, salt, iterations, 2, 8);
    return rc2CbcDecryptPkcs7(key, iv, encryptedData, 40);
  }

  if (oid === OID_PBES2) {
    // parameters = SEQUENCE { keyDerivationFunc AlgId, encryptionScheme AlgId }
    const p = readChildren(paramsTlv.content);
    const kdfAlg = p[0];
    const encAlg = p[1];
    const kdfChildren = readChildren(kdfAlg.content);
    const kdfOid = decodeOid(kdfChildren[0].content);
    if (kdfOid !== OID_PBKDF2) throw new Error('unsupported KDF: ' + kdfOid);
    const kdfParams = readChildren(kdfChildren[1].content);
    // PBKDF2-params ::= SEQUENCE { salt OCTET STRING, iterationCount INTEGER,
    //                              keyLength INTEGER OPTIONAL, prf AlgorithmIdentifier OPTIONAL }
    const salt = kdfParams[0].content;
    const iterations = Number(BigInt('0x' + kdfParams[1].content.toString('hex')));
    let prfOid = OID_HMAC_SHA1;
    for (let i = 2; i < kdfParams.length; i++) {
      if (kdfParams[i].tag === TAG.SEQUENCE) {
        const prfChildren = readChildren(kdfParams[i].content);
        prfOid = decodeOid(prfChildren[0].content);
      }
    }
    const prfMap = {
      [OID_HMAC_SHA1]: 'sha1',
      [OID_HMAC_SHA256]: 'sha256',
      [OID_HMAC_SHA512]: 'sha512',
    };
    const prfName = prfMap[prfOid] || 'sha1';

    const encChildren = readChildren(encAlg.content);
    const encOid = decodeOid(encChildren[0].content);
    const iv = encChildren[1].content;
    let keyLen;
    if (encOid === OID_AES256_CBC) keyLen = 32;
    else if (encOid === OID_AES192_CBC) keyLen = 24;
    else if (encOid === OID_AES128_CBC) keyLen = 16;
    else throw new Error('unsupported PBES2 encryption: ' + encOid);

    const key = crypto.pbkdf2Sync(password, salt, iterations, keyLen, prfName);
    return aesCbcDecryptPkcs7(key, iv, encryptedData);
  }

  throw new Error('unsupported PBE algorithm: ' + oid);
}

// ---------------------------------------------------------------------------
// SafeBag 遍历
// ---------------------------------------------------------------------------

function extractDFromPKI(pkiBuf) {
  // PrivateKeyInfo ::= SEQUENCE { version, privateKeyAlgorithm AlgId, privateKey OCTET STRING }
  const pki = readTLV(pkiBuf);
  const pkiChildren = readChildren(pki.content);
  // 期待 pkiChildren = [version, algId, privateKey OCTET STRING]
  const pkOctet = pkiChildren.find((c) => c.tag === TAG.OCTET_STRING);
  // ECPrivateKey ::= SEQUENCE { version, privateKey OCTET STRING, parameters [0], publicKey [1] }
  const ecTlv = readTLV(pkOctet.content);
  const ecChildren = readChildren(ecTlv.content);
  // 找到私钥 OCTET STRING（32B）
  let d;
  for (const c of ecChildren) {
    if (c.tag === TAG.OCTET_STRING) {
      d = c.content;
      break;
    }
  }
  if (!d) throw new Error('ECPrivateKey privateKey not found');
  if (d.length < 32) d = Buffer.concat([Buffer.alloc(32 - d.length, 0), d]);
  if (d.length > 32) d = d.subarray(d.length - 32);
  return d;
}

function walkSafeBags(safeContentsBuf, password, out) {
  const scTlv = readTLV(safeContentsBuf);
  const bags = readChildren(scTlv.content);
  for (const bag of bags) {
    if (bag.tag !== TAG.SEQUENCE) continue;
    const bagChildren = readChildren(bag.content);
    const bagId = decodeOid(bagChildren[0].content);
    // bagValue 在 [0] EXPLICIT，tag = 0xa0
    const bagValueTlv = bagChildren[1];
    // 剥 [0] 外层，拿真正内容
    const inner = readTLV(bagValueTlv.content);

    if (bagId === OID_CERT_BAG) {
      // CertBag ::= SEQUENCE { certId OID, certValue [0] EXPLICIT ANY }
      const certChildren = readChildren(inner.content);
      const certType = decodeOid(certChildren[0].content);
      if (certType !== OID_X509_CERTIFICATE) continue;
      const certOctet = readTLV(certChildren[1].content);
      // certOctet 是 OCTET STRING，内含 X.509 DER
      const x509Der = certOctet.content;
      out.certDer = Buffer.from(x509Der);
    } else if (bagId === OID_KEY_BAG) {
      // bagValue 直接是 PrivateKeyInfo（未加密）
      // inner 就是 PrivateKeyInfo 的 SEQUENCE TLV，重新序列化其原始字节
      const wholeBuf = bagValueTlv.content; // 已经剥了外层 [0]
      out.privateKeyD = extractDFromPKI(wholeBuf);
    } else if (bagId === OID_PKCS8_SHROUDED_KEY_BAG) {
      // EncryptedPrivateKeyInfo ::= SEQUENCE { encryptionAlgorithm AlgId, encryptedData OCTET STRING }
      const encChildren = readChildren(inner.content);
      const encAlgTlv = encChildren[0];
      const encDataOctet = encChildren[1];
      const pkiBytes = decryptPBE(encAlgTlv, encDataOctet.content, password);
      out.privateKeyD = extractDFromPKI(pkiBytes);
    }
  }
}

/**
 * 解析 PFX bytes，返回 { privateKeyD: Buffer(32), certDer: Buffer }。
 */
function parsePfx(pfxBytes, password) {
  // Pfx ::= SEQUENCE { version INTEGER, authSafe ContentInfo, macData MacData OPTIONAL }
  const pfxTlv = readTLV(pfxBytes);
  const pfxChildren = readChildren(pfxTlv.content);
  // pfxChildren[1] = ContentInfo(authSafe)
  const authSafeCi = pfxChildren[1];
  const authSafeChildren = readChildren(authSafeCi.content);
  const authSafeContentType = decodeOid(authSafeChildren[0].content);
  if (authSafeContentType !== OID_CT_DATA) {
    throw new Error('unsupported outer contentType: ' + authSafeContentType);
  }
  // authSafe.content 是 [0] EXPLICIT OCTET STRING，剥一层
  const asExplicit = readTLV(authSafeChildren[1].content);
  const asOctet = asExplicit; // 已经拿到 OCTET STRING 本体
  // 内部再是 AuthenticatedSafe (SEQUENCE OF ContentInfo) DER 编码
  const authSafeDer = asOctet.content;
  const asTlv = readTLV(authSafeDer);
  const contentInfos = readChildren(asTlv.content);

  const out = { privateKeyD: null, certDer: null };
  for (const ci of contentInfos) {
    const ciChildren = readChildren(ci.content);
    const ct = decodeOid(ciChildren[0].content);
    // ciChildren[1] = [0] EXPLICIT content
    const contentExplicit = readTLV(ciChildren[1].content);
    if (ct === OID_CT_DATA) {
      // content 是 OCTET STRING，内含 SafeContents DER
      const inner = contentExplicit.content;
      walkSafeBags(inner, password, out);
    } else if (ct === OID_CT_ENCRYPTED_DATA) {
      // EncryptedData ::= SEQUENCE { version INTEGER, encryptedContentInfo }
      // encryptedContentInfo ::= SEQUENCE { contentType OID, contentEncryptionAlgorithm AlgId,
      //                                     encryptedContent [0] IMPLICIT OCTET STRING OPTIONAL }
      const edChildren = readChildren(contentExplicit.content);
      const eci = edChildren[1];
      const eciChildren = readChildren(eci.content);
      // eciChildren[0] = contentType, [1] = enc algorithm, [2] = [0] IMPLICIT OCTET STRING
      const encAlgTlv = eciChildren[1];
      const encContentTlv = eciChildren[2];
      // encContentTlv 是 [0] IMPLICIT，其 content 就是密文（IMPLICIT 意味着原本是 OCTET STRING）
      const encData = encContentTlv.content;
      const decrypted = decryptPBE(encAlgTlv, encData, password);
      walkSafeBags(decrypted, password, out);
    }
  }

  if (!out.privateKeyD || !out.certDer) {
    throw new Error('PFX missing private key or certificate');
  }
  return out;
}

function parsePfxBase64(pfxBase64, password) {
  return parsePfx(Buffer.from(pfxBase64, 'base64'), password);
}

module.exports = {
  parsePfx,
  parsePfxBase64,
  // 导出用于测试/复用
  pkcs12PbeDerive,
  rc2CbcDecryptPkcs7,
};
