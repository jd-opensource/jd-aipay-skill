"""AI 付 queryRefundResult 接口 Demo（Python 版）。"""

from __future__ import annotations

import json
import sys
from collections import OrderedDict
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from aipay_demo import config
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




# 业务参数
REFUND_NO = "__REFUND_NO__"


def build_biz_json() -> str:
    biz = OrderedDict([
        ("acqMerchantNo", config.ACQ_MERCHANT_NO),  # 收单商户号
        ("accessType", config.ACCESS_TYPE),         # 接入类型：SERVICE_MER 服务商 / COMMON 普通商户
        ("refundNo", REFUND_NO),              # 退款单号
    ])
    return json.dumps(biz, ensure_ascii=False, separators=(",", ":"))


def main() -> None:
    pfx = parse_pfx_base64(config.PFX_BASE64, config.PFX_PASSWORD)
    biz_json = build_biz_json()
    biz_content = encode_biz_content(biz_json, pfx, config.SM2_JD_PUB)

    content = build_content(
        biz_content_encrypted=biz_content,
        app_id=config.APP_ID,
        agent_id=config.AGENT_ID,
        merchant_no=config.MERCHANT_NO,
    )
    sign_string = build_sign_string(content)
    sign = hmac_sm3_hex(sign_string, config.SECRET_KEY)
    content["sign"] = sign

    body = json.dumps({"data": {"content": content}}, ensure_ascii=False, separators=(",", ":"))
    headers = build_http_headers(config.APP_ID)

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

    resp = post_json(config.ENDPOINT_URL, headers, body)
    print("=================== HTTP Response ===================")
    print(resp)

    try_decrypt_response_biz_content(resp, pfx)


if __name__ == "__main__":
    main()
