"""AI 付接入配置（环境、凭证、密钥）。

由 render_server_example.sh 按所选环境自动填充。切换环境或更换凭证时，直接修改本文件
即可，无需重新生成工程。业务参数（订单号、用户、金额等）不在本模块，位于各 demo 中。
"""

ENV = "__ENV__"                       # pre | prod | sandbox
ENDPOINT_URL = "__ENDPOINT_URL__"     # 完整 endpoint URL（沙箱环境含沙箱实例 ID）
APP_ID = "__APP_ID__"                 # 应用 ID
AGENT_ID = "__AGENT_ID__"             # 渠道/Agent 标识
MERCHANT_NO = "__MERCHANT_NO__"       # 商户号
ACQ_MERCHANT_NO = "__ACQ_MERCHANT_NO__"  # 收单商户号
ACCESS_TYPE = "__ACCESS_TYPE__"       # 接入类型：SERVICE_MER 服务商 / COMMON 普通商户
SECRET_KEY = "__SECRET_KEY__"         # HMAC-SM3 密钥
PFX_PASSWORD = "__PFX_PASSWORD__"     # 商户 pfx 密码
PFX_BASE64 = "__PFX_BASE64__"         # 商户 pfx Base64
SM2_JD_PUB = "__SM2_JD_PUB__"         # 京东 SM2 公钥证书 Base64（按环境注入：沙箱/共享）
