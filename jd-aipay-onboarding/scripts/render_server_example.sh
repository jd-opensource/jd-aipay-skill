#!/usr/bin/env bash
#
# render_server_example.sh — 一键渲染 AI付服务端示例工程（Java / Python / Node.js）
#
# 用法：
#   render_server_example.sh --config <kv_file> --target <output_dir>
#
# kv_file 必填字段：
#   language          语言: java（默认） | python | nodejs
#   interface         接口: createOrder | queryPayResult | refund | queryRefundResult
#   env               环境: pre | prod | sandbox
#   base_url          环境域名（不含 /api 与尾斜杠）, 如 https://ridepassfront-pre.jd.com；env=sandbox 时可省略（内置 https://fpitest.jd.com）
#   sandbox_id        沙箱实例 ID —— 仅 env=sandbox 时必填，商户在平台创建沙箱后获得，脚本将其追加在接口路径后
#   secret_key        HMAC-SM3 密钥（env=sandbox 时可省略，固定为 "test"）
#   pfx_base64        商户 pfx 文件 base64（与 pfx_path 二选一；env=sandbox 时可省略，缺省用内置沙箱测试私钥）
#   pfx_path          商户 pfx 文件路径（与 pfx_base64 二选一，脚本自动 base64）
#   pfx_password      pfx 密码（env=sandbox 时可省略，缺省为内置沙箱测试私钥密码）
#   sm2_jd_pub        京东 SM2 公钥证书 Base64（可选；缺省按环境自动选择——sandbox 用 assets/certs/jd-sm2-pub-sandbox.b64，pre/prod 用 assets/certs/jd-sm2-pub.b64，用户提供时覆盖默认）
#   agent_id
#   app_id
#   merchant_no
#   acq_merchant_no
#   access_type       接入类型: SERVICE_MER（服务商） | COMMON（普通商户），由用户选择
#
# 按接口追加：
#   createOrder       out_trade_no  user_id  trade_amount
#   queryPayResult    out_trade_no
#   refund            original_out_trade_no  refund_amount
#   queryRefundResult refund_no
#
set -euo pipefail

CONFIG=""
TARGET=""
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SKILL_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
EXAMPLES_DIR="$SKILL_DIR/assets/server-examples"

usage() {
  sed -n '3,32p' "$0"
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --config) CONFIG="$2"; shift 2 ;;
    --target) TARGET="$2"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) echo "未知参数: $1" >&2; usage; exit 1 ;;
  esac
done

if [[ -z "$CONFIG" || -z "$TARGET" ]]; then
  usage; exit 1
fi
[[ -f "$CONFIG" ]] || { echo "配置文件不存在: $CONFIG" >&2; exit 1; }
[[ -d "$EXAMPLES_DIR" ]] || { echo "示例工程目录缺失: $EXAMPLES_DIR" >&2; exit 1; }

# 目标目录已存在则拒绝覆盖，防止误删用户数据
if [[ -e "$TARGET" ]]; then
  echo "目标目录已存在: $TARGET （请先删除或换一个目录）" >&2
  exit 1
fi

python3 - "$CONFIG" "$TARGET" "$EXAMPLES_DIR" "$SKILL_DIR" <<'PYEOF'
import os, sys, shutil, base64, re

config_path, target, examples_dir, skill_dir = sys.argv[1], sys.argv[2], sys.argv[3], sys.argv[4]

def parse_kv(p):
    d = {}
    with open(p, 'r', encoding='utf-8') as f:
        for raw in f:
            line = raw.strip()
            if not line or line.startswith('#'):
                continue
            if '=' not in line:
                continue
            k, v = line.split('=', 1)
            d[k.strip()] = v.strip()
    return d

kv = parse_kv(config_path)

def need(k):
    v = kv.get(k, '').strip()
    if not v:
        print(f"[ERR] 配置缺少必填字段: {k}", file=sys.stderr)
        sys.exit(2)
    return v

language = kv.get('language', 'java').strip().lower() or 'java'
if language not in ('java', 'python', 'nodejs'):
    print(f"[ERR] language 非法: {language}, 应为 java|python|nodejs", file=sys.stderr); sys.exit(2)

iface = need('interface')
env = need('env').lower()

# 沙箱环境使用内置域名；pre/prod 必须提供 base_url
if env == 'sandbox':
    base_url = (kv.get('base_url', '').strip() or 'https://fpitest.jd.com').rstrip('/')
else:
    base_url = need('base_url').rstrip('/')

# accessType：接入类型，SERVICE_MER 服务商 / COMMON 普通商户，由用户选择
access_type = need('access_type').strip().upper()
if access_type not in ('SERVICE_MER', 'COMMON'):
    print(f"[ERR] access_type 非法: {access_type}, 应为 SERVICE_MER|COMMON", file=sys.stderr); sys.exit(2)

# 沙箱环境密钥与商户测试证书全部内置：secret_key 固定 "test"，私钥/密码用内置沙箱物料，商户无需提供
secret_key = kv.get('secret_key', '').strip()
pfx_b64 = kv.get('pfx_base64', '').strip()
pfx_path = kv.get('pfx_path', '').strip()
pfx_password = kv.get('pfx_password', '').strip()
app_id = kv.get('app_id', '').strip()
agent_id = kv.get('agent_id', '').strip()
merchant_no = kv.get('merchant_no', '').strip()
acq_merchant_no = kv.get('acq_merchant_no', '').strip()
if env == 'sandbox':
    if not secret_key:
        secret_key = 'test'
    if not pfx_b64 and not pfx_path:
        sandbox_pfx = os.path.join(skill_dir, 'assets', 'certs', 'jd-merchant-pfx-sandbox.b64')
        if not os.path.isfile(sandbox_pfx):
            print(f"[ERR] 沙箱内置商户测试私钥缺失: {sandbox_pfx}", file=sys.stderr); sys.exit(2)
        with open(sandbox_pfx, 'r', encoding='ascii') as f:
            pfx_b64 = ''.join(f.read().split())
    if not pfx_password:
        pfx_password = 'gnvy4aQDs15VIU8H'
else:
    # pre/prod 无内置物料，密钥与证书仍为必填
    if not secret_key:
        print("[ERR] 配置缺少必填字段: secret_key", file=sys.stderr); sys.exit(2)
    if not pfx_password:
        print("[ERR] 配置缺少必填字段: pfx_password", file=sys.stderr); sys.exit(2)

# 接口映射：不同语言使用不同的主入口文件名 / 模块
JAVA_IFACE_MAP = {
    'createOrder':       ('RokidCreateOrderGatewayDemo',   'pay-ai-agent/createOrder'),
    'queryPayResult':    ('RokidQueryPayResultGatewayDemo','pay-ai-agent/queryPayResult'),
    'refund':            ('RokidRefundGatewayDemo',        'pay-ai-agent/refund'),
    'queryRefundResult': ('RokidQueryRefundGatewayDemo',   'pay-ai-agent/queryRefundResult'),
}
PY_IFACE_MAP = {
    'createOrder':       ('create_order_gateway_demo',        'pay-ai-agent/createOrder'),
    'queryPayResult':    ('query_pay_result_gateway_demo',    'pay-ai-agent/queryPayResult'),
    'refund':            ('refund_gateway_demo',              'pay-ai-agent/refund'),
    'queryRefundResult': ('query_refund_result_gateway_demo', 'pay-ai-agent/queryRefundResult'),
}
NODE_IFACE_MAP = {
    'createOrder':       ('create_order_gateway_demo',        'pay-ai-agent/createOrder'),
    'queryPayResult':    ('query_pay_result_gateway_demo',    'pay-ai-agent/queryPayResult'),
    'refund':            ('refund_gateway_demo',              'pay-ai-agent/refund'),
    'queryRefundResult': ('query_refund_result_gateway_demo', 'pay-ai-agent/queryRefundResult'),
}
if language == 'java':
    iface_map = JAVA_IFACE_MAP
elif language == 'python':
    iface_map = PY_IFACE_MAP
else:
    iface_map = NODE_IFACE_MAP
if iface not in iface_map:
    print(f"[ERR] interface 非法: {iface}, 应为 {list(iface_map)}", file=sys.stderr); sys.exit(2)
if env not in ('pre', 'prod', 'sandbox'):
    print(f"[ERR] env 非法: {env}, 应为 pre|prod|sandbox", file=sys.stderr); sys.exit(2)

main_entry, path_suffix = iface_map[iface]
# base_url 是环境域名（不含 /api），/api 由脚本统一拼接到 endpoint；
# 沙箱环境将商户自填的沙箱实例 ID 追加在接口路径后
if env == 'sandbox':
    sandbox_id = need('sandbox_id')
    endpoint_url = f"{base_url}/api/{path_suffix}/{sandbox_id}"
else:
    endpoint_url = f"{base_url}/api/{path_suffix}"

# pfx 来源：优先 pfx_base64；否则 pfx_path（沙箱环境缺省时已在上方注入内置测试私钥）
if not pfx_b64:
    if not pfx_path:
        print("[ERR] 必须提供 pfx_base64 或 pfx_path 之一", file=sys.stderr); sys.exit(2)
    if not os.path.isfile(pfx_path):
        print(f"[ERR] pfx_path 文件不存在: {pfx_path}", file=sys.stderr); sys.exit(2)
    with open(pfx_path, 'rb') as f:
        pfx_b64 = base64.b64encode(f.read()).decode('ascii')

# sm2_jd_pub 可选：缺省按环境取内置公钥——沙箱用沙箱公钥，pre/prod 用共享公钥
sm2_jd_pub = kv.get('sm2_jd_pub', '').strip()
if not sm2_jd_pub:
    cert_name = 'jd-sm2-pub-sandbox.b64' if env == 'sandbox' else 'jd-sm2-pub.b64'
    cert_path = os.path.join(skill_dir, 'assets', 'certs', cert_name)
    if not os.path.isfile(cert_path):
        print(f"[ERR] 未提供 sm2_jd_pub，且内置公钥缺失: {cert_path}", file=sys.stderr); sys.exit(2)
    with open(cert_path, 'r', encoding='ascii') as f:
        sm2_jd_pub = ''.join(f.read().split())

placeholders = {
    '__ENV__':             env,
    '__SECRET_KEY__':      secret_key,
    '__PFX_BASE64__':      pfx_b64,
    '__PFX_PASSWORD__':    pfx_password,
    '__SM2_JD_PUB__':      sm2_jd_pub,
    '__ENDPOINT_URL__':    endpoint_url,
    '__APP_ID__':          need('app_id'),
    '__AGENT_ID__':        need('agent_id'),
    '__MERCHANT_NO__':     need('merchant_no'),
    '__ACQ_MERCHANT_NO__': need('acq_merchant_no'),
    '__ACCESS_TYPE__':     access_type,   # 接入类型：SERVICE_MER 服务商 / COMMON 普通商户
    '__MAIN_CLASS__':      main_entry,   # Java 侧使用；Python 无引用
}

if iface == 'createOrder':
    placeholders['__OUT_TRADE_NO__'] = need('out_trade_no')
    placeholders['__USER_ID__']      = need('user_id')
    placeholders['__TRADE_AMOUNT__'] = need('trade_amount')
elif iface == 'queryPayResult':
    placeholders['__OUT_TRADE_NO__'] = need('out_trade_no')
elif iface == 'refund':
    placeholders['__ORIGINAL_OUT_TRADE_NO__'] = need('original_out_trade_no')
    placeholders['__REFUND_AMOUNT__']         = need('refund_amount')
elif iface == 'queryRefundResult':
    placeholders['__REFUND_NO__'] = need('refund_no')


def replace_all(text, mapping):
    for k, v in mapping.items():
        text = text.replace(k, v)
    return text


def collect_leftover(root, exts):
    leftover = set()
    for base, _, files in os.walk(root):
        for f in files:
            if not any(f.endswith(ext) or f == ext for ext in exts):
                continue
            path = os.path.join(base, f)
            with open(path, 'r', encoding='utf-8') as h:
                src = h.read()
            for m in re.findall(r'__[A-Z][A-Z0-9_]+__', src):
                leftover.add((path, m))
    return leftover


def render_files(root, exts):
    for base, _, files in os.walk(root):
        for f in files:
            if not any(f.endswith(ext) or f == ext for ext in exts):
                continue
            path = os.path.join(base, f)
            with open(path, 'r', encoding='utf-8') as h:
                src = h.read()
            new = replace_all(src, placeholders)
            with open(path, 'w', encoding='utf-8') as h:
                h.write(new)


# ---------------------------------------------------------------------------
# Java / Python / Node.js 分支：从 assets/server-examples/ 拷贝对应示例，
# 仅保留目标接口入口文件。示例工程不等同于商户业务系统已完成接入。
# ---------------------------------------------------------------------------

if language == 'java':
    java_src = os.path.join(examples_dir, 'java-quickstart')
    shutil.copytree(java_src, target)
    demo_dir = os.path.join(target, 'src', 'main', 'java', 'com', 'jdd', 'demo')
    for name in os.listdir(demo_dir):
        p = os.path.join(demo_dir, name)
        if os.path.isfile(p) and name.endswith('.java') and name != f"{main_entry}.java":
            os.remove(p)
    render_files(target, ('.java', 'pom.xml'))
    leftover = collect_leftover(target, ('.java', 'pom.xml'))
elif language == 'python':
    py_src = os.path.join(examples_dir, 'python-quickstart')
    shutil.copytree(py_src, target)
    demo_dir = os.path.join(target, 'src', 'aipay_demo')
    keep = {f"{main_entry}.py", '__init__.py', 'config.py', 'utils'}
    for name in os.listdir(demo_dir):
        if name in keep:
            continue
        full = os.path.join(demo_dir, name)
        if os.path.isfile(full) and name.endswith('.py'):
            os.remove(full)
    # 删除 tests 目录避免向用户目录发布互通脚本
    shutil.rmtree(os.path.join(target, 'tests'), ignore_errors=True)
    render_files(target, ('.py',))
    leftover = collect_leftover(target, ('.py',))
else:
    # nodejs
    node_src = os.path.join(examples_dir, 'nodejs-quickstart')
    shutil.copytree(node_src, target)
    demo_dir = os.path.join(target, 'src')
    keep = {f"{main_entry}.js", 'config.js', 'utils'}
    for name in os.listdir(demo_dir):
        if name in keep:
            continue
        full = os.path.join(demo_dir, name)
        if os.path.isfile(full) and name.endswith('.js'):
            os.remove(full)
    render_files(target, ('.js',))
    leftover = collect_leftover(target, ('.js',))


if leftover:
    print("[WARN] 存在未替换的占位符：", file=sys.stderr)
    for p, m in sorted(leftover):
        print(f"  {m}  @ {p}", file=sys.stderr)
    sys.exit(3)

# 生成工程根目录的 aipay.env：全部可替换配置项的唯一入口（支付宝式交互）
if env == 'sandbox':
    env_label = '沙箱环境'
    endpoint_note = '# 接口完整地址。【替换】末尾的沙箱实例 ID 为你自己的'
else:
    env_label = '预发环境' if env == 'pre' else '生产环境'
    endpoint_note = '# 接口完整地址'
env_text = f"""# =====================================================
# 京东 AI 付接入配置（{env_label}）
# 本文件是本工程唯一需要编辑的配置：切换环境、更换密钥/证书/商户信息，改这里即可。
# 标注【替换】的值请改成你自己的；其余为当前环境已生效的值。
# =====================================================

# 环境：pre（预发）/ prod（生产）/ sandbox（沙箱）
AIPAY_ENV={env}

{endpoint_note}
AIPAY_ENDPOINT_URL={endpoint_url}

# 应用与渠道标识 【替换：平台「应用信息」页分配的值】
AIPAY_APP_ID={app_id}
AIPAY_AGENT_ID={agent_id}

# 商户号与接入类型 【替换：平台分配的商户号；COMMON 普通商户 / SERVICE_MER 服务商】
AIPAY_MERCHANT_NO={merchant_no}
AIPAY_ACQ_MERCHANT_NO={acq_merchant_no}
AIPAY_ACCESS_TYPE={access_type}

# HMAC-SM3 签名密钥 【替换：平台「密钥配置」页获取（沙箱固定为 test）】
AIPAY_SECRET_KEY={secret_key}

# 京东 SM2 公钥证书 Base64（信封加密用；已按当前环境内置，无需替换）
AIPAY_PUBLIC_KEY={sm2_jd_pub}

# 商户私钥证书（PFX）Base64 与密码 【替换：你自己的商户证书；沙箱环境已内置公共测试私钥，无需替换】
AIPAY_MERCHANT_PFX={pfx_b64}
AIPAY_MERCHANT_PFX_PASSWORD={pfx_password}
"""
with open(os.path.join(target, 'aipay.env'), 'w', encoding='utf-8') as f:
    f.write(env_text)

print(f"[OK] 已生成 {language} 工程: {target}")
print(f"     接口: {iface}")
print(f"     环境: {env}")
print(f"     Endpoint: {endpoint_url}")
print(f"     配置文件: {os.path.join(target, 'aipay.env')}（全部可替换值集中在此，改完即生效）")
if language == 'java':
    print(f"     mainClass: com.jdd.demo.{main_entry}")
    print(f"     运行: cd {target} && mvn -q compile exec:java")
elif language == 'python':
    print(f"     入口模块: aipay_demo.{main_entry}")
    print(f"     运行:")
    print(f"       cd {target}")
    print(f"       python3 -m venv .venv && source .venv/bin/activate")
    print(f"       pip install -r requirements.txt")
    print(f"       PYTHONPATH=src python -m aipay_demo.{main_entry}")
    print(f"     或直接运行文件:")
    print(f"       python3 src/aipay_demo/{main_entry}.py")
else:
    print(f"     入口文件: src/{main_entry}.js")
    print(f"     运行:")
    print(f"       cd {target}")
    print(f"       npm install")
    print(f"       node src/{main_entry}.js")
PYEOF
