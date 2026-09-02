"""AI 付 queryPayResult 接口 Demo（Python 版）。"""

from __future__ import annotations

import json
import sys
from collections import OrderedDict
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from aipay_demo.utils.common import (
    build_content,
    build_http_headers,
    build_sign_string,
    encode_biz_content,
    hmac_sm3_hex,
    parse_pfx_base64,
    post_json,
    try_decrypt_response_biz_content,
)

ENV = "__ENV__"                       # pre | prod | sandbox
SM2_JD_PUB = "__SM2_JD_PUB__"         # 京东 SM2 公钥证书 Base64 —— 敏感参数，由用户按环境提供

SECRET_KEY = "__SECRET_KEY__"
PFX_BASE64 = "__PFX_BASE64__"
PFX_PASSWORD = "__PFX_PASSWORD__"

ENDPOINT_URL = "__ENDPOINT_URL__"
APP_ID = "__APP_ID__"
AGENT_ID = "__AGENT_ID__"
MERCHANT_NO = "__MERCHANT_NO__"
ACQ_MERCHANT_NO = "__ACQ_MERCHANT_NO__"
ACCESS_TYPE = "__ACCESS_TYPE__"       # 接入类型：SERVICE_MER 服务商 / COMMON 普通商户

# 业务参数
OUT_TRADE_NO = "__OUT_TRADE_NO__"


def build_biz_json() -> str:
    biz = OrderedDict([
        ("acqMerchantNo", ACQ_MERCHANT_NO),  # 收单商户号
        ("accessType", ACCESS_TYPE),         # 接入类型：SERVICE_MER 服务商 / COMMON 普通商户
        ("outTradeNo", OUT_TRADE_NO),        # 商户外部订单号
    ])
    return json.dumps(biz, ensure_ascii=False, separators=(",", ":"))


def main() -> None:
    pfx = parse_pfx_base64(PFX_BASE64, PFX_PASSWORD)
    biz_json = build_biz_json()
    biz_content = encode_biz_content(biz_json, pfx, SM2_JD_PUB)

    content = build_content(
        biz_content_encrypted=biz_content,
        app_id=APP_ID,
        agent_id=AGENT_ID,
        merchant_no=MERCHANT_NO,
    )
    sign_string = build_sign_string(content)
    sign = hmac_sm3_hex(sign_string, SECRET_KEY)
    content["sign"] = sign

    body = json.dumps({"data": {"content": content}}, ensure_ascii=False, separators=(",", ":"))
    headers = build_http_headers(APP_ID)

    print("=================== bizContent 明文 ===================")
    print(biz_json)
    print("=================== 签名原文 ===================")
    print(sign_string)
    print("=================== 签名结果 ===================")
    print(sign)
    print("=================== HTTP Header ===================")
    for k, v in headers.items():
        print(f"{k}:{v}")
    print("=================== HTTP Body ===================")
    print(body)

    resp = post_json(ENDPOINT_URL, headers, body)
    print("=================== HTTP Response ===================")
    print(resp)

    try_decrypt_response_biz_content(resp, pfx)


if __name__ == "__main__":
    main()
