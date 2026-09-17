"""AI 付接入配置（环境、凭证、密钥）。

所有可替换的值集中在工程根目录的 `aipay.env` 文件中（KEY=value 格式）。切换环境或更换
凭证（密钥、证书、商户号、应用标识等）时，直接编辑 aipay.env 即可，无需修改任何 Python
代码、也无需重新生成工程。业务参数（订单号、用户、金额等）不在本模块，位于各 demo 中。

本模块导入时读取 aipay.env 并初始化下方常量；文件缺失或缺项会抛出明确异常。
"""

from pathlib import Path


def _load_env_file():
    """从工程根目录查找并解析 aipay.env：当前工作目录 → 本文件逐级向上。"""
    candidates = [Path.cwd() / "aipay.env"]
    here = Path(__file__).resolve()
    candidates.extend(parent / "aipay.env" for parent in here.parents)
    for candidate in candidates:
        if candidate.is_file():
            vals = {}
            for raw in candidate.read_text(encoding="utf-8").splitlines():
                line = raw.strip()
                if not line or line.startswith("#") or "=" not in line:
                    continue
                key, _, value = line.partition("=")
                vals[key.strip()] = value.strip()
            return vals
    raise FileNotFoundError(
        "未找到配置文件 aipay.env（应在工程根目录）。该文件承载全部可替换配置项，请勿删除或改名。"
    )


def _get(key: str) -> str:
    value = _VALUES.get(key)
    if not value:
        raise ValueError(f"aipay.env 缺少配置项: {key}，请补齐后重试")
    return value


_VALUES = _load_env_file()

ENV = _get("AIPAY_ENV")                                # pre | prod | sandbox
ENDPOINT_URL = _get("AIPAY_ENDPOINT_URL")              # 完整 endpoint URL（沙箱环境含沙箱实例 ID）
APP_ID = _get("AIPAY_APP_ID")                          # 应用 ID
AGENT_ID = _get("AIPAY_AGENT_ID")                      # 渠道/Agent 标识
MERCHANT_NO = _get("AIPAY_MERCHANT_NO")                # 商户号
ACQ_MERCHANT_NO = _get("AIPAY_ACQ_MERCHANT_NO")        # 收单商户号
ACCESS_TYPE = _get("AIPAY_ACCESS_TYPE")                # 接入类型：SERVICE_MER 服务商 / COMMON 普通商户
SECRET_KEY = _get("AIPAY_SECRET_KEY")                  # HMAC-SM3 密钥
PFX_PASSWORD = _get("AIPAY_MERCHANT_PFX_PASSWORD")     # 商户 pfx 密码
PFX_BASE64 = _get("AIPAY_MERCHANT_PFX")                # 商户 pfx Base64
SM2_JD_PUB = _get("AIPAY_PUBLIC_KEY")                  # 京东 SM2 公钥证书 Base64（按环境：沙箱/共享）
