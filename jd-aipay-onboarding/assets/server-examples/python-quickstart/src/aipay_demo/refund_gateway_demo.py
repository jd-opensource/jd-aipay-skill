"""AI 付 refund 接口 Demo（Python 版）。"""

from __future__ import annotations

import json
import random
import sys
from collections import OrderedDict
from datetime import datetime
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
ORIGINAL_OUT_TRADE_NO = "__ORIGINAL_OUT_TRADE_NO__"
REFUND_AMOUNT = __REFUND_AMOUNT__  # int 字面量（不加引号）


def _gen_refund_no() -> str:
    """REFUND + yyyyMMddHHmmss + 3位随机数字，保证幂等。"""

    return "REFUND" + datetime.now().strftime("%Y%m%d%H%M%S") + f"{random.randint(0, 999):03d}"


def build_biz_json() -> str:
    biz = OrderedDict([
        ("acqMerchantNo", ACQ_MERCHANT_NO),           # 收单商户号
        ("accessType", ACCESS_TYPE),                  # 接入类型：SERVICE_MER 服务商 / COMMON 普通商户
        ("originalOutTradeNo", ORIGINAL_OUT_TRADE_NO),  # 原下单商户订单号
        ("refundNo", _gen_refund_no()),               # 退款单号（幂等键）
        ("refundAmount", REFUND_AMOUNT),              # 退款金额（分）
        ("currency", "CNY"),                          # 币种
        ("refundReason", "AI付退款测试"),               # 退款原因
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
