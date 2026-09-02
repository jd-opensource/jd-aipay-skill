/**
 * 极简 DER (ASN.1) 编解码工具：只覆盖 PFX/CMS/信封所需的 tag（SEQUENCE、SET、OCTET STRING、
 * INTEGER、OID、CONTEXT tag 等），不做通用 X.509 语义化。所有函数返回/接受 Buffer。
 */
'use strict';

const TAG = {
  BOOLEAN: 0x01,
  INTEGER: 0x02,
  BIT_STRING: 0x03,
  OCTET_STRING: 0x04,
  NULL: 0x05,
  OID: 0x06,
  UTF8_STRING: 0x0c,
  PRINTABLE_STRING: 0x13,
  IA5_STRING: 0x16,
  UTC_TIME: 0x17,
  GENERALIZED_TIME: 0x18,
  SEQUENCE: 0x30,
  SET: 0x31,
};

function readLength(buf, off) {
  const first = buf[off];
  if ((first & 0x80) === 0) return { len: first, next: off + 1 };
  const n = first & 0x7f;
  if (n === 0) throw new Error('indefinite DER length not supported');
  let len = 0;
  for (let i = 0; i < n; i++) len = (len << 8) | buf[off + 1 + i];
  return { len, next: off + 1 + n };
}

/**
 * 从 buf[off] 起解析一个 TLV。返回 { tag, headerLen, contentOffset, contentLen, next, content }。
 */
function readTLV(buf, off = 0) {
  const tag = buf[off];
  const { len, next } = readLength(buf, off + 1);
  const contentOffset = next;
  const contentLen = len;
  const end = contentOffset + contentLen;
  return {
    tag,
    headerLen: contentOffset - off,
    contentOffset,
    contentLen,
    next: end,
    content: buf.subarray(contentOffset, end),
  };
}

/**
 * 把 SEQUENCE/SET 的 content 拆分成子 TLV 数组。
 */
function readChildren(content) {
  const out = [];
  let off = 0;
  while (off < content.length) {
    const tlv = readTLV(content, off);
    out.push(tlv);
    off = tlv.next;
  }
  return out;
}

function encodeLength(len) {
  if (len < 0x80) return Buffer.from([len]);
  const bytes = [];
  let v = len;
  while (v > 0) {
    bytes.unshift(v & 0xff);
    v >>>= 8;
  }
  return Buffer.from([0x80 | bytes.length, ...bytes]);
}

function encodeTLV(tag, content) {
  return Buffer.concat([Buffer.from([tag]), encodeLength(content.length), content]);
}

function encodeSequence(children) {
  return encodeTLV(TAG.SEQUENCE, Buffer.concat(children));
}

function encodeSet(children) {
  return encodeTLV(TAG.SET, Buffer.concat(children));
}

function encodeOctetString(buf) {
  return encodeTLV(TAG.OCTET_STRING, buf);
}

/**
 * 编码正整数为 DER INTEGER：确保最高位为 1 时前置 0x00 补齐正号；0 用一字节 0x00。
 */
function encodeInteger(bufOrBigIntOrHex) {
  let bytes;
  if (typeof bufOrBigIntOrHex === 'string') {
    const hex = bufOrBigIntOrHex.length % 2 ? '0' + bufOrBigIntOrHex : bufOrBigIntOrHex;
    bytes = Buffer.from(hex, 'hex');
  } else if (Buffer.isBuffer(bufOrBigIntOrHex)) {
    bytes = bufOrBigIntOrHex;
  } else if (typeof bufOrBigIntOrHex === 'bigint' || typeof bufOrBigIntOrHex === 'number') {
    let v = BigInt(bufOrBigIntOrHex);
    if (v === 0n) return encodeTLV(TAG.INTEGER, Buffer.from([0]));
    const arr = [];
    while (v > 0n) {
      arr.unshift(Number(v & 0xffn));
      v >>= 8n;
    }
    bytes = Buffer.from(arr);
  } else {
    throw new Error('encodeInteger: unsupported input');
  }
  // 去除多余前导 0（保留一个用于符号）
  let i = 0;
  while (i < bytes.length - 1 && bytes[i] === 0x00 && (bytes[i + 1] & 0x80) === 0) i++;
  bytes = bytes.subarray(i);
  if ((bytes[0] & 0x80) !== 0) {
    // 高位置位，补 0x00 表示正数
    bytes = Buffer.concat([Buffer.from([0x00]), bytes]);
  }
  return encodeTLV(TAG.INTEGER, bytes);
}

/**
 * OID 字符串（如 "1.2.156.10197.1.401"）编码为 DER OBJECT IDENTIFIER。
 */
function encodeOid(oid) {
  const parts = oid.split('.').map(Number);
  if (parts.length < 2) throw new Error('invalid OID: ' + oid);
  const first = parts[0] * 40 + parts[1];
  const out = [first];
  for (let i = 2; i < parts.length; i++) {
    const arr = [];
    let v = parts[i];
    arr.unshift(v & 0x7f);
    v >>>= 7;
    while (v > 0) {
      arr.unshift((v & 0x7f) | 0x80);
      v >>>= 7;
    }
    out.push(...arr);
  }
  return encodeTLV(TAG.OID, Buffer.from(out));
}

/**
 * DER OID -> 字符串
 */
function decodeOid(content) {
  const first = content[0];
  const parts = [Math.floor(first / 40), first % 40];
  let cur = 0;
  for (let i = 1; i < content.length; i++) {
    cur = (cur << 7) | (content[i] & 0x7f);
    if ((content[i] & 0x80) === 0) {
      parts.push(cur);
      cur = 0;
    }
  }
  return parts.join('.');
}

/**
 * DER INTEGER content 解为无符号 Buffer（大端），可选左侧补零到 minLen。
 */
function decodeIntegerUnsigned(content, minLen) {
  let bytes = content;
  // DER INTEGER 允许一个 0x00 前缀表示正数，去除
  if (bytes.length > 1 && bytes[0] === 0x00) bytes = bytes.subarray(1);
  if (minLen && bytes.length < minLen) {
    const pad = Buffer.alloc(minLen - bytes.length, 0);
    bytes = Buffer.concat([pad, bytes]);
  }
  return bytes;
}

module.exports = {
  TAG,
  readTLV,
  readChildren,
  readLength,
  encodeTLV,
  encodeSequence,
  encodeSet,
  encodeOctetString,
  encodeInteger,
  encodeOid,
  decodeOid,
  decodeIntegerUnsigned,
};
