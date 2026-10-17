package aipay

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// TestPfxInteropWithLocalEnv 用真实 aipay.env（含商户 PFX 与京东公钥）做联调互验：
//
//	AIPAY_INTEROP_ENV=/path/to/aipay.env go test -v -run TestPfxInterop ./aipay/
//
// 未设置环境变量时跳过（不向外部暴露任何路径）。输出私钥标量/证书字段，
// 可与 Python/Java 版解析结果人工比对，亦可用于跨语言互操作排障。
func TestPfxInteropWithLocalEnv(t *testing.T) {
	envPath := os.Getenv("AIPAY_INTEROP_ENV")
	if envPath == "" {
		t.Skip("未设置 AIPAY_INTEROP_ENV，跳过本地联调互验")
	}
	raw, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("读取 %s 失败: %v", envPath, err)
	}
	values := map[string]string{}
	for _, line := range strings.Split(string(raw), "\n") {
		line = trimSpace(line)
		if line == "" || line[0] == '#' {
			continue
		}
		if idx := strings.Index(line, "="); idx > 0 {
			values[trimSpace(line[:idx])] = trimSpace(line[idx+1:])
		}
	}
	pfxB64 := values["AIPAY_MERCHANT_PFX"]
	password := values["AIPAY_MERCHANT_PFX_PASSWORD"]
	jdPub := values["AIPAY_PUBLIC_KEY"]
	if pfxB64 == "" || password == "" || jdPub == "" {
		t.Fatalf("aipay.env 缺少 AIPAY_MERCHANT_PFX / PASSWORD / AIPAY_PUBLIC_KEY")
	}

	pfx, err := ParsePfxFromBase64(pfxB64, password)
	if err != nil {
		t.Fatalf("PFX 解析失败: %v", err)
	}
	fmt.Printf("PrivateKeyD = %s\n", hexEncode(pfx.PrivateKeyD))
	fmt.Printf("CertSerial  = %s\n", pfx.Cert.SerialNumber.Text(16))
	fmt.Printf("CertPub65   = %s\n", hexEncode(pfx.Cert.Pub65))
	fmt.Printf("IssuerDER   = %x\n", pfx.Cert.IssuerDER)
	fmt.Printf("IssuerDERLen = %d\n", len(pfx.Cert.IssuerDER))

	cert, err := ParseCertFromBase64DER(jdPub)
	if err != nil {
		t.Fatalf("京东公钥证书解析失败: %v", err)
	}
	fmt.Printf("JdPub65     = %s\n", hexEncode(cert.Pub65))

	// 全链路互验（模拟响应方向）：网关用「商户公钥」做信封加密，商户用私钥解密剥层。
	// （请求方向是加密给京东公钥，只有京东能解，本地无法自解密还原。）
	biz := []byte(`{"interop":"go-pfx-test","ok":true}`)
	p7, err := Sm2P7SignAttached(biz, pfx)
	if err != nil {
		t.Fatalf("CMS 签名失败: %v", err)
	}
	envelop, err := Sm2EncryptEnvelopWithCert(p7, pfx.Cert)
	if err != nil {
		t.Fatalf("信封加密失败: %v", err)
	}
	got, err := DecodeBizContent(envelop, pfx)
	if err != nil {
		t.Fatalf("信封解密/CMS 剥层失败: %v", err)
	}
	if got != string(biz) {
		t.Fatalf("互验往返不一致: %s", got)
	}
	fmt.Println("InteropRoundtrip = OK")
}
