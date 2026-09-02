"""跨语言互通测试：Python 加密 → Java wyaks-security 解密。"""

import base64
import os
import sys

sys.path.insert(0, os.path.join(os.path.dirname(__file__), "..", "src"))

from aipay_demo.utils.pfx import parse_pfx_base64
from aipay_demo.utils.crypto import sign_envelop, sm2_decrypt_envelop, cms_extract_content


def main():
    with open("/tmp/interop/merchant.pfx.b64") as f:
        m_pfx_b64 = f.read().strip()
    with open("/tmp/interop/jd_pub.b64") as f:
        jd_cert_b64 = f.read().strip()
    with open("/tmp/interop/pfx_password.txt") as f:
        pfx_pwd = f.read().strip()
    with open("/tmp/interop/plain.txt", "rb") as f:
        plain = f.read()

    # 用 Python 解 PFX
    m_pfx = parse_pfx_base64(m_pfx_b64, pfx_pwd)
    print(f"[python] parsed merchant pfx, cert_der len={len(m_pfx.cert_der)}, d len={len(m_pfx.private_key_d)}")

    # Python 加密 → 写入 envelope.b64 供 Java 侧解
    envelop = sign_envelop(plain, m_pfx, jd_cert_b64)
    with open("/tmp/interop/envelope.b64", "w") as f:
        f.write(envelop)
    print(f"[python] wrote /tmp/interop/envelope.b64, len={len(envelop)}")

    # 顺便让 Python 自己也能解，走 jd 的 pfx（跨语言不依赖，但验证 gmssl 解码正确性）
    with open("/tmp/interop/jd.pfx.b64") as f:
        jd_pfx_b64 = f.read().strip()
    jd_pfx = parse_pfx_base64(jd_pfx_b64, pfx_pwd)
    p7 = sm2_decrypt_envelop(envelop, jd_pfx.private_key_d)
    recovered = cms_extract_content(p7)
    print(f"[python-self-check] recovered plain = {recovered!r}")
    assert recovered == plain, "python self decrypt mismatch"
    print("[python-self-check] OK")


if __name__ == "__main__":
    main()
