"""
京东 wyaks-security 兼容的 SM2 数字信封与 P7 (SM3WithSM2) 签名实现（Python 版）。

信封结构（非标 CMS，见 SM2EnvelopUtil2）：
  Base64( version(1B=0x01) || DER(SEQ(c11 INTEGER, c12 INTEGER, c3 OCTET STRING, c2 OCTET STRING)) || IV(16B) || SM4-CBC-PKCS7(data, key, iv) )
  其中 key/iv 各 16 字节随机；c11=SM2密文C1点X, c12=Y, c2=对称加密载荷, c3=MAC(SM3)

P7 签名（SM3WithSM2, directSignature=true, attachFlag=true）：
  ContentInfo(signedData) {
    SignedData {
      version = 1,
      digestAlgorithms = { SM3 (1.2.156.10197.1.401) },
      encapContentInfo = { contentType = data(1.2.840.113549.1.7.1), content OCTET STRING = srcData },
      certificates = [商户证书],
      signerInfos = [{
        version = 1,
        sid = IssuerAndSerialNumber,
        digestAlgorithm = SM3,
        signatureAlgorithm = SM3WithSM2 (1.2.156.10197.1.501),
        signature = SM2Sign(srcData, ZA)  # directSignature=true 时无 signed attributes
      }]
    }
  }
"""

from __future__ import annotations

import os
from typing import Tuple

from asn1crypto import cms as _cms, core as _core, x509 as _x509, algos as _algos
from gmssl import sm2 as _sm2

from .pfx import PfxContent

# 京东公钥证书 OID
_OID_SM2_ENC = "1.2.156.10197.1.301"  # sm2(EC) 曲线
_OID_SM3 = "1.2.156.10197.1.401"
_OID_SM3_WITH_SM2 = "1.2.156.10197.1.501"
_OID_DATA = "1.2.840.113549.1.7.1"
_OID_SIGNED_DATA = "1.2.840.113549.1.7.2"


def _sm2_encrypt(pub_uncompressed_65b: bytes, data: bytes) -> Tuple[bytes, bytes, bytes]:
    """用 SM2 公钥加密 data，返回 (c1_xy_64B, c2, c3_32B)。"""

    if len(pub_uncompressed_65b) != 65 or pub_uncompressed_65b[0] != 0x04:
        raise ValueError("expect uncompressed SM2 public key with 0x04 prefix, 65 bytes")
    pub_hex = pub_uncompressed_65b[1:].hex()
    engine = _sm2.CryptSM2(private_key="00" * 32, public_key=pub_hex, mode=0, asn1=False)
    cipher = engine.encrypt(data)
    if cipher is None:
        raise RuntimeError("gmssl sm2.encrypt returned None (KDF failure)")
    # 结构：C1(64B) || C2(len(data)) || C3(32B)
    c1 = cipher[:64]
    c2 = cipher[64:-32]
    c3 = cipher[-32:]
    return c1, c2, c3


def _sm4_cbc_encrypt(key: bytes, iv: bytes, data: bytes) -> bytes:
    from gmssl.sm4 import CryptSM4, SM4_ENCRYPT

    c = CryptSM4()
    c.set_key(key, SM4_ENCRYPT)
    return c.crypt_cbc(iv, data)


def _sm4_cbc_decrypt(key: bytes, iv: bytes, data: bytes) -> bytes:
    from gmssl.sm4 import CryptSM4, SM4_DECRYPT

    c = CryptSM4()
    c.set_key(key, SM4_DECRYPT)
    return c.crypt_cbc(iv, data)


def _get_sm2_pub_from_cert_b64(cert_b64: str) -> bytes:
    """从京东 SM2 公钥证书（Base64 编码 DER）中提取 65 字节未压缩公钥点。"""

    import base64

    cert_der = base64.b64decode(cert_b64)
    cert = _x509.Certificate.load(cert_der)
    pub_key_bit = cert["tbs_certificate"]["subject_public_key_info"]["public_key"]
    # BIT STRING 的原始字节前有 unused-bits 前缀由 asn1crypto 自动处理；.native 返回 (unused_bits, bytes)
    raw = pub_key_bit.native
    if isinstance(raw, tuple):
        raw = raw[1]
    elif isinstance(raw, (bytes, bytearray)):
        raw = bytes(raw)
    else:
        raw = bytes(raw)
    if len(raw) != 65 or raw[0] != 0x04:
        raise ValueError(f"unexpected SM2 public key format from cert (len={len(raw)})")
    return raw


def sm2_encrypt_envelop(data: bytes, cert_b64: str) -> str:
    """京东私有 SM2 数字信封（等价 SM2EnvelopUtil2.encryptEnvelop）。

    返回 Base64 字符串。
    """

    import base64

    key = os.urandom(16)
    iv = os.urandom(16)
    pub = _get_sm2_pub_from_cert_b64(cert_b64)

    c1, c2_body, c3 = _sm2_encrypt(pub, key)
    c11 = int.from_bytes(c1[:32], "big")
    c12 = int.from_bytes(c1[32:], "big")

    # DER SEQ(INTEGER c11, INTEGER c12, OCTET STRING c3, OCTET STRING c2)
    seq = _core.Sequence()
    # 用 asn1crypto Any 组装：这里直接构造 SEQUENCE of 4 元素
    from asn1crypto.core import Integer, OctetString

    class _EnvKey(_core.Sequence):
        _fields = [
            ("c11", Integer),
            ("c12", Integer),
            ("c3", OctetString),
            ("c2", OctetString),
        ]

    envelop_key = _EnvKey({"c11": c11, "c12": c12, "c3": c3, "c2": c2_body})
    enkey = envelop_key.dump()

    endata = _sm4_cbc_encrypt(key, iv, data)
    result = bytes([0x01]) + enkey + iv + endata
    return base64.b64encode(result).decode("ascii")


def sm2_decrypt_envelop(envelop_b64: str, private_key_d: bytes) -> bytes:
    """京东私有 SM2 数字信封解密（等价 SM2EnvelopUtil2.decryptEnvelop）。

    private_key_d: 32 字节 SM2 私钥标量（大端）。
    """

    import base64

    data = base64.b64decode(envelop_b64)
    version = data[0]
    if version != 0x01:
        raise ValueError(f"unknown envelop version: {version}")

    # 解 SEQUENCE 头，先读 tag/length 得到 enkey 全长
    body = data[1:]
    if body[0] != 0x30:
        raise ValueError("expect SEQUENCE at offset 1")

    def _read_length(buf: bytes, off: int) -> Tuple[int, int]:
        first = buf[off]
        if first & 0x80 == 0:
            return first, off + 1
        n = first & 0x7F
        length = int.from_bytes(buf[off + 1:off + 1 + n], "big")
        return length, off + 1 + n

    seq_content_len, next_off = _read_length(body, 1)
    total_seq_len = 1 + next_off - 1 + seq_content_len  # tag + len_bytes + content
    # 更严谨：next_off 已经跳过 tag(0)+len，seq_content_len 是 content 长度
    # so total = next_off + seq_content_len
    total_seq_len = next_off + seq_content_len

    enkey = body[:total_seq_len]
    rest = body[total_seq_len:]
    iv = rest[:16]
    endata = rest[16:]

    from asn1crypto.core import Integer, OctetString

    class _EnvKey(_core.Sequence):
        _fields = [
            ("c11", Integer),
            ("c12", Integer),
            ("c3", OctetString),
            ("c2", OctetString),
        ]

    ek = _EnvKey.load(enkey)
    c11b = int(ek["c11"].native).to_bytes(32, "big")
    c12b = int(ek["c12"].native).to_bytes(32, "big")
    c2b = ek["c2"].native
    c3b = ek["c3"].native

    # 拼装成 gmssl 期望的 C1(64) + C2 + C3 结构（无 0x04 前缀）
    sm2_cipher = c11b + c12b + c2b + c3b

    engine = _sm2.CryptSM2(private_key=private_key_d.hex(), public_key="00" * 64, mode=0, asn1=False)
    key = engine.decrypt(sm2_cipher)
    plain = _sm4_cbc_decrypt(key, iv, endata)
    return plain


# ---------------------------------------------------------------------------
# P7 SignedData (SM3WithSM2, directSignature=true, attachFlag=true)
# ---------------------------------------------------------------------------

# SM2 signature 用户身份 ID 默认值
_DEFAULT_SM2_USER_ID = b"1234567812345678"


def _sm3_hash(data: bytes) -> bytes:
    from gmssl import sm3

    return bytes.fromhex(sm3.sm3_hash([b for b in data]))


def _sm2_za(user_id: bytes, pub_uncompressed_65b: bytes) -> bytes:
    """SM2 签名前的 ZA = SM3(ENTLA || ID || a || b || Gx || Gy || Px || Py)。"""

    # SM2 曲线参数
    a = int("FFFFFFFEFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF00000000FFFFFFFFFFFFFFFC", 16).to_bytes(32, "big")
    b = int("28E9FA9E9D9F5E344D5A9E4BCF6509A7F39789F515AB8F92DDBCBD414D940E93", 16).to_bytes(32, "big")
    gx = int("32C4AE2C1F1981195F9904466A39C9948FE30BBFF2660BE1715A4589334C74C7", 16).to_bytes(32, "big")
    gy = int("BC3736A2F4F6779C59BDCEE36B692153D0A9877CC62A474002DF32E52139F0A0", 16).to_bytes(32, "big")
    px = pub_uncompressed_65b[1:33]
    py = pub_uncompressed_65b[33:65]
    entla = (len(user_id) * 8).to_bytes(2, "big")
    return _sm3_hash(entla + user_id + a + b + gx + gy + px + py)


def _sm2_sign(data_with_za: bytes, private_key_d: bytes, pub_uncompressed_65b: bytes) -> bytes:
    """SM2 签名，返回 DER 编码的 SEQUENCE(r,s)。"""

    priv_hex = private_key_d.hex()
    pub_hex = pub_uncompressed_65b[1:].hex()
    engine = _sm2.CryptSM2(private_key=priv_hex, public_key=pub_hex, mode=0, asn1=False)
    # gmssl.sm2.sign 返回 r||s hex（128 位十六进制字符串）
    from gmssl import func

    k = func.random_hex(engine.para_len)
    sign_raw = engine.sign(data_with_za, k)  # 输入是 M~=ZA||M 已经处理，还是自己 hash 后签？
    # gmssl.sm2.sign 内部：直接把 data 当消息哈希后签，需要我们外部先算 e = SM3(ZA||M)
    # 但 sign() 内部再 hash 一次会不对；查看源码：
    # gmssl.sm2.sign(data, K) 内部：e = int(data.hex(), 16) —— 把 data 当"已 hash 的 e"直接签
    # 所以我们传入的应是 SM3(ZA||M) 结果的 32 字节
    r_s_hex = sign_raw
    r_bytes = bytes.fromhex(r_s_hex[:64])
    s_bytes = bytes.fromhex(r_s_hex[64:128])
    r = int.from_bytes(r_bytes, "big")
    s = int.from_bytes(s_bytes, "big")
    from asn1crypto.core import Integer, Sequence

    class _RS(Sequence):
        _fields = [("r", Integer), ("s", Integer)]

    return _RS({"r": r, "s": s}).dump()


def sm2_p7_sign_attached(src_data: bytes, pfx: PfxContent) -> bytes:
    """SM3WithSM2 CMS 签名（directSignature=true, attachFlag=true），返回 CMS ContentInfo DER。"""

    cert = _x509.Certificate.load(pfx.cert_der)
    # 商户 SM2 公钥（65B）
    pub_bit = cert["tbs_certificate"]["subject_public_key_info"]["public_key"].native
    if isinstance(pub_bit, tuple):
        pub_uncompressed = pub_bit[1]
    else:
        pub_uncompressed = bytes(pub_bit)
    if len(pub_uncompressed) != 65 or pub_uncompressed[0] != 0x04:
        raise ValueError(f"unexpected merchant SM2 public key length: {len(pub_uncompressed)}")

    # 计算 e = SM3(ZA || M)
    za = _sm2_za(_DEFAULT_SM2_USER_ID, pub_uncompressed)
    e = _sm3_hash(za + src_data)
    signature = _sm2_sign(e, pfx.private_key_d, pub_uncompressed)

    # 构造 SignedData
    issuer = cert.issuer
    serial = cert.serial_number

    digest_algo = _algos.DigestAlgorithm({"algorithm": _OID_SM3})
    signature_algo = _algos.SignedDigestAlgorithm({"algorithm": _OID_SM3_WITH_SM2})

    sid = _cms.SignerIdentifier({
        "issuer_and_serial_number": _cms.IssuerAndSerialNumber({
            "issuer": issuer,
            "serial_number": serial,
        })
    })

    signer_info = _cms.SignerInfo({
        "version": "v1",
        "sid": sid,
        "digest_algorithm": digest_algo,
        "signature_algorithm": signature_algo,
        "signature": signature,
    })

    encap_content_info = _cms.ContentInfo({
        "content_type": _OID_DATA,
        "content": _core.OctetString(src_data),
    })

    signed_data = _cms.SignedData({
        "version": "v1",
        "digest_algorithms": _cms.DigestAlgorithms([digest_algo]),
        "encap_content_info": encap_content_info,
        "certificates": _cms.CertificateSet([_cms.CertificateChoices({"certificate": cert})]),
        "signer_infos": _cms.SignerInfos([signer_info]),
    })

    content_info = _cms.ContentInfo({
        "content_type": _OID_SIGNED_DATA,
        "content": signed_data,
    })

    return content_info.dump()


def cms_extract_content(cms_der: bytes) -> bytes:
    """从 CMS SignedData DER 中提取 encapContentInfo.eContent 明文（用于解签数字信封后取原文）。"""

    ci = _cms.ContentInfo.load(cms_der)
    sd = ci["content"]
    econtent = sd["encap_content_info"]["content"]
    # content 可能是 OCTET STRING（ParsableOctetString / OctetString）
    if econtent is None or (hasattr(econtent, "contents") and econtent.contents is None):
        return b""
    raw = econtent.native if not isinstance(econtent, bytes) else econtent
    if isinstance(raw, (bytes, bytearray)):
        return bytes(raw)
    return bytes(raw)


def sign_envelop(src_data: bytes, pfx: PfxContent, jd_cert_b64: str) -> str:
    """入口：SM2 数字信封 + P7 签名（等价 Java signEnvelop）。"""

    p7 = sm2_p7_sign_attached(src_data, pfx)
    return sm2_encrypt_envelop(p7, jd_cert_b64)


def verify_envelop(envelop_b64: str, pfx: PfxContent) -> bytes:
    """入口：解京东 SM2 信封，剥出 CMS 明文（等价 Java verifyEnvelop 拿第 0 项）。"""

    p7_bytes = sm2_decrypt_envelop(envelop_b64, pfx.private_key_d)
    return cms_extract_content(p7_bytes)
