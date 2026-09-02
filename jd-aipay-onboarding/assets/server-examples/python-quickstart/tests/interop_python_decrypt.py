"""Java 加密 → Python 解密。"""
import os, sys, base64

sys.path.insert(0, os.path.join(os.path.dirname(__file__), "..", "src"))

from aipay_demo.utils.pfx import parse_pfx_base64
from aipay_demo.utils.crypto import verify_envelop


def main():
    with open("/tmp/interop/envelope_java.b64") as f:
        env = f.read().strip()
    with open("/tmp/interop/jd.pfx.b64") as f:
        pfx_b64 = f.read().strip()
    with open("/tmp/interop/pfx_password.txt") as f:
        pwd = f.read().strip()
    with open("/tmp/interop/plain.txt", "rb") as f:
        expected = f.read()
    pfx = parse_pfx_base64(pfx_b64, pwd)
    got = verify_envelop(env, pfx)
    print(f"got   = {got!r}")
    print(f"expect= {expected!r}")
    assert got == expected, "cross-decrypt mismatch"
    print("[PASS] Java-encrypted envelope decrypted by Python successfully")


if __name__ == "__main__":
    main()
