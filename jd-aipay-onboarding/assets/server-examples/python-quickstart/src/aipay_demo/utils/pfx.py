"""
PKCS#12 (PFX) 解析工具：从商户 PFX 文件中提取 SM2 私钥标量 d 与 X.509 证书 DER。

背景：Python 的 cryptography 库不支持 SM2 曲线（OID 1.2.156.10197.1.301），无法直接调用
其 pkcs12.load_key_and_certificates。因此本模块使用 asn1crypto 手工解 PKCS12 结构：
  1. 解 PFX_BASE64 -> PFX bytes
  2. 用 cryptography.hazmat.primitives.serialization.pkcs12 的 load_pkcs12 会失败（SM2）
     因此我们直接解析 PFX ASN.1（PFX/AuthenticatedSafe/SafeContents/SafeBag）
  3. 对加密的 SafeBag，用密码派生 PBE 密钥并 3DES-CBC 解密（PKCS12 常见默认加密算法）
  4. 从 KeyBag 或 PKCS8ShroudedKeyBag 中提取 PrivateKeyInfo，再解出 ECPrivateKey.privateKey（32B 大端标量 d）
  5. 从 CertBag 中提取 X.509 证书 DER
"""

from __future__ import annotations

import base64
import hashlib
from dataclasses import dataclass
from typing import Tuple

from asn1crypto import pkcs12 as _p12, cms as _cms, keys as _keys, x509 as _x509, core as _core, algos as _algos


# ---------------------------------------------------------------------------
# RC2 40-bit pure-python 实现（PBES1 pbeWithSHAAnd40BitRC2-CBC 场景专用）
# 新版 cryptography 库已移除 ARC2；BC 生成的 PFX 证书 bag 默认用此算法加密。
# ---------------------------------------------------------------------------

_RC2_PITABLE = (
    0xd9,0x78,0xf9,0xc4,0x19,0xdd,0xb5,0xed,0x28,0xe9,0xfd,0x79,0x4a,0xa0,0xd8,0x9d,
    0xc6,0x7e,0x37,0x83,0x2b,0x76,0x53,0x8e,0x62,0x4c,0x64,0x88,0x44,0x8b,0xfb,0xa2,
    0x17,0x9a,0x59,0xf5,0x87,0xb3,0x4f,0x13,0x61,0x45,0x6d,0x8d,0x09,0x81,0x7d,0x32,
    0xbd,0x8f,0x40,0xeb,0x86,0xb7,0x7b,0x0b,0xf0,0x95,0x21,0x22,0x5c,0x6b,0x4e,0x82,
    0x54,0xd6,0x65,0x93,0xce,0x60,0xb2,0x1c,0x73,0x56,0xc0,0x14,0xa7,0x8c,0xf1,0xdc,
    0x12,0x75,0xca,0x1f,0x3b,0xbe,0xe4,0xd1,0x42,0x3d,0xd4,0x30,0xa3,0x3c,0xb6,0x26,
    0x6f,0xbf,0x0e,0xda,0x46,0x69,0x07,0x57,0x27,0xf2,0x1d,0x9b,0xbc,0x94,0x43,0x03,
    0xf8,0x11,0xc7,0xf6,0x90,0xef,0x3e,0xe7,0x06,0xc3,0xd5,0x2f,0xc8,0x66,0x1e,0xd7,
    0x08,0xe8,0xea,0xde,0x80,0x52,0xee,0xf7,0x84,0xaa,0x72,0xac,0x35,0x4d,0x6a,0x2a,
    0x96,0x1a,0xd2,0x71,0x5a,0x15,0x49,0x74,0x4b,0x9f,0xd0,0x5e,0x04,0x18,0xa4,0xec,
    0xc2,0xe0,0x41,0x6e,0x0f,0x51,0xcb,0xcc,0x24,0x91,0xaf,0x50,0xa1,0xf4,0x70,0x39,
    0x99,0x7c,0x3a,0x85,0x23,0xb8,0xb4,0x7a,0xfc,0x02,0x36,0x5b,0x25,0x55,0x97,0x31,
    0x2d,0x5d,0xfa,0x98,0xe3,0x8a,0x92,0xae,0x05,0xdf,0x29,0x10,0x67,0x6c,0xba,0xc9,
    0xd3,0x00,0xe6,0xcf,0xe1,0x9e,0xa8,0x2c,0x63,0x16,0x01,0x3f,0x58,0xe2,0x89,0xa9,
    0x0d,0x38,0x34,0x1b,0xab,0x33,0xff,0xb0,0xbb,0x48,0x0c,0x5f,0xb9,0xb1,0xcd,0x2e,
    0xc5,0xf3,0xdb,0x47,0xe5,0xa5,0x9c,0x77,0x0a,0xa6,0x20,0x68,0xfe,0x7f,0xc1,0xad,
)


def _rc2_key_schedule(key: bytes, effective_bits: int) -> list:
    T = len(key)
    T1 = effective_bits
    T8 = (T1 + 7) // 8
    TM = 0xff >> (8 * T8 - T1)
    L = list(key) + [0] * (128 - T)
    for i in range(T, 128):
        L[i] = _RC2_PITABLE[(L[i - 1] + L[i - T]) & 0xff]
    L[128 - T8] = _RC2_PITABLE[L[128 - T8] & TM]
    for i in range(127 - T8, -1, -1):
        L[i] = _RC2_PITABLE[L[i + 1] ^ L[i + T8]]
    K = [(L[2 * i] | (L[2 * i + 1] << 8)) & 0xffff for i in range(64)]
    return K


def _rc2_decrypt_block_ref(K: list, block8: bytes) -> bytes:
    """兼容旧调用点。"""
    return _tiny_rc2_decrypt(K, block8)


def _tiny_rc2_decrypt(K: list, block8: bytes) -> bytes:
    """RC2 单块 8B 解密（RFC 2268）。"""
    R = [int.from_bytes(block8[i:i + 2], "little") for i in range(0, 8, 2)]
    s = (1, 2, 3, 5)

    def rol16(v, r):
        return ((v << r) | (v >> (16 - r))) & 0xffff

    def ror16(v, r):
        return ((v >> r) | (v << (16 - r))) & 0xffff

    # 反向执行：加密顺序为 mix*5, mash, mix*6, mash, mix*5
    # 反向：反 mix*5 → 反 mash → 反 mix*6 → 反 mash → 反 mix*5
    j = 63

    def inv_mix_round():
        nonlocal j
        for i in (3, 2, 1, 0):
            R[i] = ror16(R[i], s[i])
            R[i] = (R[i] - K[j] - (R[(i - 1) % 4] & R[(i - 2) % 4]) - ((~R[(i - 1) % 4]) & R[(i - 3) % 4])) & 0xffff
            j -= 1

    def inv_mash_round():
        for i in (3, 2, 1, 0):
            R[i] = (R[i] - K[R[(i - 1) % 4] & 63]) & 0xffff

    for _ in range(5):
        inv_mix_round()
    inv_mash_round()
    for _ in range(6):
        inv_mix_round()
    inv_mash_round()
    for _ in range(5):
        inv_mix_round()

    out = b""
    for v in R:
        out += v.to_bytes(2, "little")
    return out


def _rc2_cbc_decrypt_pkcs7(key: bytes, iv: bytes, ciphertext: bytes, effective_bits: int) -> bytes:
    K = _rc2_key_schedule(key, effective_bits)
    plain = b""
    prev = iv
    for i in range(0, len(ciphertext), 8):
        block = ciphertext[i:i + 8]
        dec = _tiny_rc2_decrypt(K, block)
        plain += bytes(a ^ b for a, b in zip(dec, prev))
        prev = block
    pad_len = plain[-1]
    return plain[:-pad_len]


@dataclass
class PfxContent:
    """PFX 解析结果。"""

    # SM2 私钥标量 d（32 字节大端整数）
    private_key_d: bytes
    # X.509 证书 DER 编码
    cert_der: bytes


def _pbe_derive(password: str, salt: bytes, iterations: int, purpose: int, key_len: int) -> bytes:
    """PKCS#12 密钥派生算法（RFC 7292 附录 B）。

    purpose = 1: encryption key
    purpose = 2: initialization vector
    purpose = 3: MAC key
    """

    # Unicode 密码 UTF-16BE + 2 字节 NUL 结尾
    pwd_bytes = password.encode("utf-16-be") + b"\x00\x00" if password else b"\x00\x00"

    u = 20  # SHA-1 输出长度
    v = 64  # SHA-1 block size

    D = bytes([purpose] * v)

    def _fill(buf: bytes, length: int) -> bytes:
        if not buf:
            return b""
        n = ((length + len(buf) - 1) // len(buf))
        return (buf * n)[:length]

    s_len = ((len(salt) + v - 1) // v) * v
    p_len = ((len(pwd_bytes) + v - 1) // v) * v
    S = _fill(salt, s_len)
    P = _fill(pwd_bytes, p_len)
    I = S + P

    c = ((key_len + u - 1) // u)
    out = b""
    for _i in range(c):
        A = hashlib.sha1(D + I).digest()
        for _ in range(iterations - 1):
            A = hashlib.sha1(A).digest()
        out += A

        # 更新 I
        B = _fill(A, v)
        B_int = int.from_bytes(B, "big")
        new_I = b""
        for j in range(0, len(I), v):
            chunk = int.from_bytes(I[j:j + v], "big")
            summed = (chunk + B_int + 1) & ((1 << (8 * v)) - 1)
            new_I += summed.to_bytes(v, "big")
        I = new_I

    return out[:key_len]


def _decrypt_pbe(encryption_algo: _algos.EncryptionAlgorithm, encrypted_data: bytes, password: str) -> bytes:
    """支持 PKCS12 常见加密算法（pbeWithSHAAnd3-KeyTripleDES-CBC / pbeWithSHAAnd40BitRC2-CBC / PBES2）。"""

    oid = encryption_algo["algorithm"].dotted

    # pbeWithSHAAnd3-KeyTripleDES-CBC = 1.2.840.113549.1.12.1.3
    # pbeWithSHAAnd40BitRC2-CBC = 1.2.840.113549.1.12.1.6
    if oid == "1.2.840.113549.1.12.1.3":
        params = encryption_algo["parameters"]
        salt = params["salt"].native
        iterations = params["iterations"].native
        key = _pbe_derive(password, salt, iterations, purpose=1, key_len=24)
        iv = _pbe_derive(password, salt, iterations, purpose=2, key_len=8)

        from cryptography.hazmat.primitives.ciphers import Cipher, algorithms, modes  # type: ignore

        cipher = Cipher(algorithms.TripleDES(key), modes.CBC(iv)).decryptor()
        plain_padded = cipher.update(encrypted_data) + cipher.finalize()
        # 去除 PKCS7 填充
        pad_len = plain_padded[-1]
        return plain_padded[:-pad_len]

    if oid == "1.2.840.113549.1.12.1.6":
        # 40-bit RC2；BC 生成 PFX 的证书 bag 常用
        params = encryption_algo["parameters"]
        salt = params["salt"].native
        iterations = params["iterations"].native
        key = _pbe_derive(password, salt, iterations, purpose=1, key_len=5)  # 40-bit
        iv = _pbe_derive(password, salt, iterations, purpose=2, key_len=8)
        try:
            from cryptography.hazmat.primitives.ciphers import Cipher, algorithms, modes  # type: ignore

            cipher = Cipher(algorithms.ARC2(key), modes.CBC(iv)).decryptor()  # type: ignore
        except Exception:
            # 新版 cryptography 移除了 ARC2；退回纯 Python 实现
            return _rc2_cbc_decrypt_pkcs7(key, iv, encrypted_data, effective_bits=40)
        plain_padded = cipher.update(encrypted_data) + cipher.finalize()
        pad_len = plain_padded[-1]
        return plain_padded[:-pad_len]

    if oid == "1.2.840.113549.1.5.13":  # PBES2
        params = encryption_algo["parameters"]
        kdf = params["key_derivation_func"]
        enc = params["encryption_scheme"]
        if kdf["algorithm"].dotted != "1.2.840.113549.1.5.12":
            raise NotImplementedError(f"unsupported KDF: {kdf['algorithm'].dotted}")
        kdf_params = kdf["parameters"]
        salt = kdf_params["salt"].native
        iterations = kdf_params["iteration_count"].native
        prf_oid = kdf_params["prf"]["algorithm"].dotted if kdf_params["prf"] else "1.2.840.113549.2.7"
        prf_map = {
            "1.2.840.113549.2.7": hashlib.sha1,
            "1.2.840.113549.2.9": hashlib.sha256,
            "1.2.840.113549.2.11": hashlib.sha512,
        }
        hash_func = prf_map.get(prf_oid, hashlib.sha1)

        from cryptography.hazmat.primitives.kdf.pbkdf2 import PBKDF2HMAC  # type: ignore
        from cryptography.hazmat.primitives import hashes as _hashes  # type: ignore

        hash_algo_map = {
            hashlib.sha1: _hashes.SHA1(),
            hashlib.sha256: _hashes.SHA256(),
            hashlib.sha512: _hashes.SHA512(),
        }
        # 依据 enc 判断 key 长度
        enc_oid = enc["algorithm"].dotted
        if enc_oid == "2.16.840.1.101.3.4.1.42":  # aes256-CBC
            key_len = 32
        elif enc_oid == "2.16.840.1.101.3.4.1.22":  # aes192-CBC
            key_len = 24
        elif enc_oid == "2.16.840.1.101.3.4.1.2":  # aes128-CBC
            key_len = 16
        else:
            raise NotImplementedError(f"unsupported PBES2 encryption: {enc_oid}")
        kdf_obj = PBKDF2HMAC(algorithm=hash_algo_map[hash_func], length=key_len, salt=salt, iterations=iterations)
        key = kdf_obj.derive(password.encode("utf-8"))
        iv = enc["parameters"].native
        from cryptography.hazmat.primitives.ciphers import Cipher, algorithms, modes  # type: ignore
        cipher = Cipher(algorithms.AES(key), modes.CBC(iv)).decryptor()
        plain_padded = cipher.update(encrypted_data) + cipher.finalize()
        pad_len = plain_padded[-1]
        return plain_padded[:-pad_len]

    raise NotImplementedError(f"unsupported PBE encryption algorithm: {oid}")


def parse_pfx(pfx_bytes: bytes, password: str) -> PfxContent:
    """解析 PFX 字节，返回 SM2 私钥标量 d（32B）与 X.509 证书 DER。"""

    pfx = _p12.Pfx.load(pfx_bytes)
    auth_safe = pfx["auth_safe"]
    if auth_safe["content_type"].native != "data":
        raise ValueError("unsupported PFX outer content type (expect 'data')")
    # PFX 的 auth_safe.content 是 OctetString，内含 AuthenticatedSafe DER
    auth_safe_data = auth_safe["content"].native
    authenticated_safe = _p12.AuthenticatedSafe.load(auth_safe_data)

    private_key_d: bytes | None = None
    cert_der: bytes | None = None

    for content_info in authenticated_safe:
        ct = content_info["content_type"].native
        if ct == "data":
            # 未加密的 SafeContents，包在 OCTET STRING 里
            inner = content_info["content"].native
            safe_contents = _p12.SafeContents.load(inner)
        elif ct == "encrypted_data":
            encrypted_content_info = content_info["content"]["encrypted_content_info"]
            enc_algo = encrypted_content_info["content_encryption_algorithm"]
            enc_content = encrypted_content_info["encrypted_content"].native
            decrypted = _decrypt_pbe(enc_algo, enc_content, password)
            safe_contents = _p12.SafeContents.load(decrypted)
        else:
            continue

        for safe_bag in safe_contents:
            bag_id = safe_bag["bag_id"].native
            if bag_id == "cert_bag":
                cert_bag = safe_bag["bag_value"]
                # cert_bag.cert_value 是 ParsableOctetString(x509.Certificate)
                cert_value = cert_bag["cert_value"]
                try:
                    cert_der = cert_value.parsed.dump()
                except Exception:
                    cert_der = cert_value.native
            elif bag_id == "pkcs8_shrouded_key_bag":
                enc_pki = safe_bag["bag_value"]  # EncryptedPrivateKeyInfo
                enc_algo = enc_pki["encryption_algorithm"]
                enc_data = enc_pki["encrypted_data"].native
                pki_bytes = _decrypt_pbe(enc_algo, enc_data, password)
                pki = _keys.PrivateKeyInfo.load(pki_bytes)
                private_key_d = _extract_d_from_pki(pki)
            elif bag_id == "key_bag":
                pki = safe_bag["bag_value"]
                private_key_d = _extract_d_from_pki(pki)

    if private_key_d is None or cert_der is None:
        raise ValueError("PFX missing private key or certificate")
    return PfxContent(private_key_d=private_key_d, cert_der=cert_der)


def _extract_d_from_pki(pki: _keys.PrivateKeyInfo) -> bytes:
    """从 PKCS8 PrivateKeyInfo 中提取 EC 私钥 d（32B 大端）。"""

    algo_oid = pki["private_key_algorithm"]["algorithm"].dotted
    # id-ecPublicKey = 1.2.840.10045.2.1；SM2 私钥常见承载 OID
    inner = pki["private_key"].parsed
    # inner 是 ECPrivateKey SEQUENCE { version, privateKey OCTET STRING(32), ..., publicKey [1] IMPLICIT BIT STRING OPTIONAL }
    ec_priv = _keys.ECPrivateKey.load(inner.dump()) if not isinstance(inner, _keys.ECPrivateKey) else inner
    d_raw = ec_priv["private_key"].native
    # asn1crypto 对 ECPrivateKey.private_key 返回 int（大端整数）
    if isinstance(d_raw, int):
        d_bytes = d_raw.to_bytes(32, "big")
    else:
        d_bytes = bytes(d_raw)
        if len(d_bytes) < 32:
            d_bytes = b"\x00" * (32 - len(d_bytes)) + d_bytes
    return d_bytes


def parse_pfx_base64(pfx_base64: str, password: str) -> PfxContent:
    """便利入口：直接接收 pfx 的 base64 字符串。"""

    return parse_pfx(base64.b64decode(pfx_base64), password)
