"""AI 付网关公共辅助：HMAC-SM3 签名、content 组装、HTTP 调用、响应 bizContent 解密。"""

from __future__ import annotations

import http.client
import json
import random
import ssl
import time
import urllib.parse
import uuid
from typing import Any, Dict

from gmssl import sm3

from .crypto import sign_envelop, verify_envelop
from .pfx import PfxContent, parse_pfx_base64


# 参与签名的字段（ASCII 升序）——与 Java demo 保持一致
_SIGN_KEYS = (
    "agentId",
    "appId",
    "bizContent",
    "encType",
    "merchantNo",
    "nonce",
    "reqNo",
    "timestamp",
    "version",
)


def random_hex(length: int) -> str:
    return "".join(random.choice("0123456789ABCDEF") for _ in range(length))


def gen_req_no() -> str:
    return uuid.uuid4().hex.upper()


def now_ms() -> str:
    return str(int(time.time() * 1000))


def _hmac_sm3(secret_key: bytes, data: bytes) -> str:
    """HMAC-SM3，返回小写 hex 字符串。"""

    block_size = 64  # SM3 分组
    if len(secret_key) > block_size:
        # 大于 block_size 先做一次 SM3
        secret_key = bytes.fromhex(sm3.sm3_hash([b for b in secret_key]))
    if len(secret_key) < block_size:
        secret_key = secret_key + b"\x00" * (block_size - len(secret_key))
    o_key_pad = bytes(b ^ 0x5C for b in secret_key)
    i_key_pad = bytes(b ^ 0x36 for b in secret_key)
    inner = sm3.sm3_hash([b for b in (i_key_pad + data)])
    outer = sm3.sm3_hash([b for b in (o_key_pad + bytes.fromhex(inner))])
    return outer.lower()


def hmac_sm3_hex(string_to_sign: str, secret_key: str) -> str:
    return _hmac_sm3(secret_key.encode("utf-8"), string_to_sign.encode("utf-8"))


def build_sign_string(content: Dict[str, str]) -> str:
    """按 ASCII 升序把 content 层字段拼成 k1=v1&k2=v2；空值不参与；signType 不参与。"""

    parts = []
    for key in _SIGN_KEYS:
        val = content.get(key)
        if val is None or val == "":
            continue
        parts.append(f"{key}={val}")
    return "&".join(parts)


def encode_biz_content(biz_json: str, pfx: PfxContent, jd_cert_b64: str) -> str:
    """SM2 数字信封加密 bizContent，返回 Base64 字符串。"""

    return sign_envelop(biz_json.encode("utf-8"), pfx, jd_cert_b64)


def decode_biz_content(biz_content_b64: str, pfx: PfxContent) -> str:
    """SM2 数字信封解密 bizContent，返回明文字符串。"""

    return verify_envelop(biz_content_b64, pfx).decode("utf-8")


def build_content(
    *,
    biz_content_encrypted: str,
    app_id: str,
    agent_id: str,
    merchant_no: str,
) -> Dict[str, str]:
    """组装 content 层字段（TreeMap 顺序 = ASCII 排序，Python dict 3.7+ 保序输出无所谓）。"""

    return {
        "agentId": agent_id,
        "appId": app_id,
        "bizContent": biz_content_encrypted,
        "encType": "SM2",
        "merchantNo": merchant_no,
        "nonce": random_hex(16),
        "reqNo": gen_req_no(),
        "signType": "SM3",
        "timestamp": now_ms(),
        "version": "1.0",
    }


def build_http_headers(app_id: str) -> Dict[str, str]:
    return {
        "app-id": app_id,
        "encrypt-type": "NONE",
        "source-type": "H5",
        "login-type": "0",
        "cache-control": "no-cache",
        "content-type": "application/json",
        "stream-type": "false",
    }


def post_json(url: str, headers: Dict[str, str], body_json: str, *, timeout: int = 30) -> str:
    """使用 http.client 直发，保留 header 名的原始大小写。

    ⚠️ 关键陷阱：AI 付网关对 HTTP header 名**大小写敏感**（必须是全小写
    ``stream-type`` / ``app-id`` / ``encrypt-type`` / ``source-type`` /
    ``login-type`` 等）。若用 ``urllib.request.Request.add_header()``，
    Python 会把 header 名规范化为首字母大写（``stream-type`` → ``Stream-Type``），
    网关识别不到 ``stream-type=false``，会当成 SSE 流式请求处理，走另一条鉴权
    链路，返回 ``10000403 资源受权信息缺失``。

    因此这里改用 ``http.client.HTTPSConnection.putheader``，它按传入字节
    原样发出，不做首字母大写化。
    """

    parsed = urllib.parse.urlparse(url)
    is_https = parsed.scheme == "https"
    port = parsed.port or (443 if is_https else 80)
    host = parsed.hostname
    path = parsed.path or "/"
    if parsed.query:
        path = f"{path}?{parsed.query}"

    body_bytes = body_json.encode("utf-8")
    if is_https:
        ctx = ssl.create_default_context()
        ctx.check_hostname = False
        ctx.verify_mode = ssl.CERT_NONE
        conn = http.client.HTTPSConnection(host, port, timeout=timeout, context=ctx)
    else:
        conn = http.client.HTTPConnection(host, port, timeout=timeout)
    try:
        conn.putrequest("POST", path, skip_host=False, skip_accept_encoding=True)
        conn.putheader("Content-Length", str(len(body_bytes)))
        for k, v in headers.items():
            conn.putheader(k, v)
        conn.endheaders(message_body=body_bytes)
        resp = conn.getresponse()
        code = resp.status
        body = resp.read().decode("utf-8", errors="replace")
    finally:
        conn.close()
    return f"HTTP {code} | {body}"


def try_decrypt_response_biz_content(response_text: str, pfx: PfxContent) -> None:
    """尝试解密响应 bizContent 并打印。"""

    idx = response_text.find("{")
    if idx < 0:
        return
    try:
        root = json.loads(response_text[idx:])
    except Exception as ex:  # 响应不是 JSON
        print(f"[响应 bizContent 解密失败] 非 JSON 响应: {ex}")
        return
    content_obj = ((root or {}).get("data") or {}).get("content") or {}
    # 网关成功响应时 data.content 是 JSON 字符串（内层含 bizContent 等字段），
    # 失败响应时 data.content 直接是 dict 或空串，需分别兼容
    if isinstance(content_obj, str):
        try:
            content_obj = json.loads(content_obj)
        except Exception:
            return
    if not isinstance(content_obj, dict):
        return
    biz_content = content_obj.get("bizContent")
    enc_type = content_obj.get("encType") or content_obj.get("encryptType")
    if not biz_content:
        return
    print("=================== 响应 encType ===================")
    print(enc_type)
    print("=================== 响应 bizContent 明文 ===================")
    try:
        if (enc_type or "").upper() == "SM2":
            print(decode_biz_content(biz_content, pfx))
        else:
            import base64

            print(base64.b64decode(biz_content).decode("utf-8", errors="replace"))
    except Exception as ex:
        print(f"[响应 bizContent 解密失败] {ex}")


__all__ = [
    "PfxContent",
    "parse_pfx_base64",
    "random_hex",
    "gen_req_no",
    "now_ms",
    "hmac_sm3_hex",
    "build_sign_string",
    "encode_biz_content",
    "decode_biz_content",
    "build_content",
    "build_http_headers",
    "post_json",
    "try_decrypt_response_biz_content",
]
