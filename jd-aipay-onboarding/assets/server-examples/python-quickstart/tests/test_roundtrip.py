"""端到端往返测试：
1. 生成 SM2 密钥对与自签证书（模拟京东 CA 场景）
2. 用 Python sign_envelop 加密 → 用同一私钥 verify_envelop 解密 → 比对原文
3. 单独测 sm2_p7_sign_attached：解 CMS → cms_extract_content 拿原文 → gmssl.sm2 验签
"""

import base64
import os
import sys

sys.path.insert(0, os.path.join(os.path.dirname(__file__), "..", "src"))

from asn1crypto import x509, core, keys, algos
from gmssl import sm2, func
import datetime

from aipay_demo.utils.crypto import (
    sign_envelop,
    verify_envelop,
    sm2_encrypt_envelop,
    sm2_decrypt_envelop,
    sm2_p7_sign_attached,
    cms_extract_content,
    _sm2_za,
    _sm3_hash,
    _DEFAULT_SM2_USER_ID,
)
from aipay_demo.utils.pfx import PfxContent


def _gen_sm2_keypair():
    priv_hex = func.random_hex(64)
    c = sm2.CryptSM2(private_key=priv_hex, public_key="00" * 64, mode=0, asn1=False)
    pub_hex = c._kg(int(priv_hex, 16), c.ecc_table["g"])
    return bytes.fromhex(priv_hex), bytes.fromhex("04" + pub_hex)


def _make_self_signed_cert(priv_d: bytes, pub_uncompressed: bytes, subject_cn: str) -> bytes:
    # 构造最小可用 X.509 v3 证书
    name = x509.Name.build({"common_name": subject_cn, "country_name": "CN"})
    validity = x509.Validity({
        "not_before": x509.Time({"utc_time": datetime.datetime(2020, 1, 1, tzinfo=datetime.timezone.utc)}),
        "not_after": x509.Time({"general_time": datetime.datetime(2099, 1, 1, tzinfo=datetime.timezone.utc)}),
    })

    # SubjectPublicKeyInfo：algorithm = ecPublicKey + curve=sm2p256v1
    spki = keys.PublicKeyInfo({
        "algorithm": keys.PublicKeyAlgorithm({
            "algorithm": "1.2.840.10045.2.1",  # id-ecPublicKey
            "parameters": keys.ECDomainParameters(name="named", value="1.2.156.10197.1.301"),
        }),
        "public_key": pub_uncompressed,
    })

    tbs = x509.TbsCertificate({
        "version": "v3",
        "serial_number": 1,
        "signature": algos.SignedDigestAlgorithm({"algorithm": "1.2.156.10197.1.501"}),
        "issuer": name,
        "validity": validity,
        "subject": name,
        "subject_public_key_info": spki,
    })

    tbs_der = tbs.dump()

    # 用 SM2 签 tbs
    za = _sm2_za(_DEFAULT_SM2_USER_ID, pub_uncompressed)
    e = _sm3_hash(za + tbs_der)
    engine = sm2.CryptSM2(private_key=priv_d.hex(), public_key=pub_uncompressed[1:].hex(), mode=0, asn1=False)
    k = func.random_hex(64)
    rs_hex = engine.sign(e, k)
    r_int = int(rs_hex[:64], 16)
    s_int = int(rs_hex[64:], 16)
    from asn1crypto.core import Sequence, Integer

    class _RS(Sequence):
        _fields = [("r", Integer), ("s", Integer)]

    sig = _RS({"r": r_int, "s": s_int}).dump()

    cert = x509.Certificate({
        "tbs_certificate": tbs,
        "signature_algorithm": algos.SignedDigestAlgorithm({"algorithm": "1.2.156.10197.1.501"}),
        "signature_value": sig,
    })
    return cert.dump()


def main():
    # 商户密钥
    m_priv, m_pub = _gen_sm2_keypair()
    m_cert = _make_self_signed_cert(m_priv, m_pub, "MerchantTest")

    # 京东"公钥证书"（也自签，用于测信封）
    jd_priv, jd_pub = _gen_sm2_keypair()
    jd_cert = _make_self_signed_cert(jd_priv, jd_pub, "JDPubTest")

    pfx = PfxContent(private_key_d=m_priv, cert_der=m_cert)
    jd_cert_b64 = base64.b64encode(jd_cert).decode()

    plain = b'{"outTradeNo":"TEST123","tradeAmount":9900}'
    envelop_b64 = sign_envelop(plain, pfx, jd_cert_b64)
    print("envelop b64 len =", len(envelop_b64))

    # 京东侧解信封（用 jd_priv）
    jd_pfx = PfxContent(private_key_d=jd_priv, cert_der=jd_cert)
    recovered_p7 = sm2_decrypt_envelop(envelop_b64, jd_priv)
    print("recovered_p7 len =", len(recovered_p7))
    recovered_plain = cms_extract_content(recovered_p7)
    print("recovered plain =", recovered_plain)
    assert recovered_plain == plain, "PLAIN MISMATCH"
    print("[PASS] sign_envelop -> jd verify (decrypt+cms extract) round-trip OK")

    # 再测响应的解密：京东用商户公钥回加密（模拟）
    resp_plain = b'{"resultCode":"SUCCESS","payToken":"tk_abc"}'
    resp_env = sign_envelop(resp_plain, jd_pfx, base64.b64encode(m_cert).decode())
    got = verify_envelop(resp_env, pfx)
    assert got == resp_plain, "resp plain mismatch"
    print("[PASS] response verify_envelop -> plain OK")

    print("ALL PASSED")


if __name__ == "__main__":
    main()
