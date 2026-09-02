"""AI 付 createOrder 接口 Demo（Python 版）：
bizContent 使用 SM2 数字信封加密（encType=SM2），外层用 HMAC-SM3 计算 sign，
组装 data.content 结构后通过 HTTPS POST 调用网关接口。
"""

from __future__ import annotations

import json
import sys
import time
from collections import OrderedDict
from datetime import datetime
from pathlib import Path

# 让 `python src/aipay_demo/xxx.py` 或 `PYTHONPATH=src python -m aipay_demo.xxx` 都能 import
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

# ============================================================================
# 环境与业务参数（由 render_server_example.sh 替换占位符）
# ============================================================================
ENV = "__ENV__"                       # pre | prod | sandbox
SM2_JD_PUB = "__SM2_JD_PUB__"         # 京东 SM2 公钥证书 Base64 —— 敏感参数，由用户按环境提供

# 敏感参数
SECRET_KEY = "__SECRET_KEY__"         # HMAC-SM3 密钥
PFX_BASE64 = "__PFX_BASE64__"         # 商户 pfx Base64
PFX_PASSWORD = "__PFX_PASSWORD__"     # 商户 pfx 密码

# 接口 URL 与身份
ENDPOINT_URL = "__ENDPOINT_URL__"     # 完整 endpoint URL
APP_ID = "__APP_ID__"
AGENT_ID = "__AGENT_ID__"
MERCHANT_NO = "__MERCHANT_NO__"       # 服务商商户号
ACQ_MERCHANT_NO = "__ACQ_MERCHANT_NO__"  # 收单商户号
ACCESS_TYPE = "__ACCESS_TYPE__"       # 接入类型：SERVICE_MER 服务商 / COMMON 普通商户

# 业务参数
OUT_TRADE_NO = "__OUT_TRADE_NO__"
USER_ID = "__USER_ID__"
TRADE_AMOUNT = "__TRADE_AMOUNT__"     # 分


def build_biz_json() -> str:
    device = OrderedDict([
        ("vendor", "TEST"),
        ("deviceId", "SN-TEST-01231231"),
        ("deviceAccount", "test_device_001"),
        ("deviceName", "测试设备"),
        ("deviceType", "SMART_DEVICE"),
        ("deviceSn", "SN-TEST-01231231"),
        ("deviceModel", "TestModel"),
    ])
    biz = OrderedDict([
        ("tradeSubject", "AI付测试订单"),           # 交易主题
        ("clientType", "APP"),                       # 客户端类型
        ("deviceInfo", json.dumps(device, ensure_ascii=False, separators=(",", ":"))),
        ("tradeAmount", TRADE_AMOUNT),               # 交易金额（分）
        ("createDate", datetime.now().strftime("%Y%m%d%H%M%S")),
        ("acqMerchantNo", ACQ_MERCHANT_NO),          # 收单商户号
        ("accessType", ACCESS_TYPE),                 # 接入类型：SERVICE_MER 服务商 / COMMON 普通商户
        ("outTradeNo", OUT_TRADE_NO),                # 商户外部订单号
        ("tradeType", "Aipay"),                      # 交易类型
        ("userIp", "127.0.0.1"),                     # 用户 IP
        ("userId", USER_ID),                         # 用户 ID
        ("tradeRemark", "AI付接口测试"),
        ("notifyUrl", "https://merchant.example.com/notify/pay"),
        ("currency", "CNY"),                         # 币种
        ("expiryTime", "604800"),                    # 订单有效期（秒）
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
