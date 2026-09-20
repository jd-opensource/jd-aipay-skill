#!/usr/bin/env bash
#
# cert_to_base64.sh — 证书文件转 Base64（企业站下载的证书 → aipay.env 可用的值）
#
# 支持类型：
#   .pfx / .p12                 商户私钥证书（PKCS#12）→ Base64（用于 AIPAY_MERCHANT_PFX）
#   .pem / .cer / .crt          X.509 证书（PEM 或 DER 编码）→ Base64(DER)（用于 AIPAY_PUBLIC_KEY 等）
#
# 用法：
#   cert_to_base64.sh <证书文件> [--password <pfx密码>] [--out <输出文件>] [--env-file <aipay.env路径> --env-key <键名>]
#
# 行为：
#   - Base64 结果默认输出到 stdout（单行，可直接管道/重定向）；诊断信息走 stderr
#   - --out 将结果写入指定文件
#   - --env-file + --env-key 将结果直接替换/追加到 aipay.env 的对应键（等号后整值替换）
#   - --password 用于校验 pfx 完整性（可选；不提供时仅做转换不校验）
#
set -euo pipefail

FILE=""
PASSWORD=""
OUT=""
ENV_FILE=""
ENV_KEY=""

usage() {
  sed -n '3,17p' "$0"
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --password) PASSWORD="$2"; shift 2 ;;
    --out) OUT="$2"; shift 2 ;;
    --env-file) ENV_FILE="$2"; shift 2 ;;
    --env-key) ENV_KEY="$2"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) FILE="$1"; shift ;;
  esac
done

[[ -n "$FILE" ]] || { echo "[ERR] 缺少证书文件参数" >&2; usage; exit 1; }
[[ -f "$FILE" ]] || { echo "[ERR] 文件不存在: $FILE" >&2; exit 1; }
if [[ -n "$ENV_FILE" && -z "$ENV_KEY" ]]; then
  echo "[ERR] 指定 --env-file 时必须同时指定 --env-key" >&2; exit 1
fi

TMP_B64="$(mktemp /tmp/certb64.XXXXXX)"
trap 'rm -f "$TMP_B64"' EXIT

detect_and_convert() {
  # PEM 证书
  if grep -q "BEGIN CERTIFICATE" "$FILE" 2>/dev/null; then
    echo "[TYPE] X.509 证书（PEM）" >&2
    openssl x509 -in "$FILE" -outform DER 2>/dev/null | openssl base64 -A > "$TMP_B64"
    openssl x509 -in "$FILE" -noout -subject -serial -dates 2>&1 | sed 's/^/[META] /' >&2
    return 0
  fi
  # 按扩展名的私钥证书（兼容 bash3.2：用 tr 做小写化）
  FILE_LOWER=$(printf '%s' "$FILE" | tr '[:upper:]' '[:lower:]')
  case "$FILE_LOWER" in
    *.pfx|*.p12)
      echo "[TYPE] 商户私钥证书（PKCS#12/PFX）" >&2
      if [[ -n "$PASSWORD" ]]; then
        openssl pkcs12 -info -in "$FILE" -passin "pass:$PASSWORD" -nokeys -noout 2>&1 | grep -q "MAC verified OK" \
          || { echo "[ERR] PFX 密码校验失败（密码错误或文件损坏）" >&2; exit 1; }
        echo "[META] PFX 密码校验通过" >&2
      fi
      openssl base64 -A -in "$FILE" > "$TMP_B64"
      return 0
      ;;
  esac
  # DER 证书
  if openssl x509 -inform DER -in "$FILE" -noout >/dev/null 2>&1; then
    echo "[TYPE] X.509 证书（DER）" >&2
    openssl x509 -inform DER -in "$FILE" -outform DER 2>/dev/null | openssl base64 -A > "$TMP_B64"
    openssl x509 -inform DER -in "$FILE" -noout -subject -serial -dates 2>&1 | sed 's/^/[META] /' >&2
    return 0
  fi
  echo "[ERR] 无法识别的证书类型。支持：.pfx/.p12（商户私钥证书）、.pem/.cer/.crt（X.509 证书，PEM/DER）" >&2
  exit 1
}

detect_and_convert

B64=$(cat "$TMP_B64")
if [[ -z "$B64" ]]; then
  echo "[ERR] 转换结果为空" >&2
  exit 1
fi
echo "[META] Base64 长度: $(printf '%s' "$B64" | wc -c | tr -d ' ') 字符" >&2

# 写 aipay.env（存在该键则整值替换，否则追加）
if [[ -n "$ENV_FILE" ]]; then
  if [[ ! -f "$ENV_FILE" ]]; then
    echo "[ERR] aipay.env 不存在: $ENV_FILE" >&2
    exit 1
  fi
  if grep -q "^${ENV_KEY}=" "$ENV_FILE"; then
    sed -i '' "s|^${ENV_KEY}=.*|${ENV_KEY}=${B64}|" "$ENV_FILE" 2>/dev/null \
      || sed -i "s|^${ENV_KEY}=.*|${ENV_KEY}=${B64}|" "$ENV_FILE"
    echo "[OK] 已更新 $ENV_FILE 中的 ${ENV_KEY}" >&2
  else
    printf '\n# 由 cert_to_base64.sh 写入\n%s=%s\n' "$ENV_KEY" "$B64" >> "$ENV_FILE"
    echo "[OK] 已在 $ENV_FILE 追加 ${ENV_KEY}" >&2
  fi
fi

if [[ -n "$OUT" ]]; then
  cp "$TMP_B64" "$OUT"
  echo "[OK] 已写入文件: $OUT" >&2
fi

# Base64 结果输出到 stdout（单行）
cat "$TMP_B64"
