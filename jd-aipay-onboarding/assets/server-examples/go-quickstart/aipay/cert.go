// 极简 X.509 证书解析：只提取信封加密与 CMS 签名所需的字段
// （SM2 公钥裸点、签发方 Name 原始 DER、序列号），
// 不解析也不校验签名算法——国密证书的 OID 不被 Go 标准库支持，故手工解析。
package aipay

import (
	"crypto/x509/pkix"
	"encoding/asn1"
	"math/big"
)

type x509Certificate struct {
	TBS tbsCertificate
	// 后两个字段不关心，但需要声明才能定位 TBS
	SignatureAlgorithmRaw asn1.RawValue
	SignatureValueRaw     asn1.BitString
}

type tbsCertificate struct {
	Raw                  asn1.RawContent
	Version              int `asn1:"optional,explicit,default:0,tag:0"`
	SerialNumber         *big.Int
	SignatureAlgorithm   pkix.AlgorithmIdentifier
	Issuer               asn1.RawValue
	Validity             asn1.RawValue
	Subject              asn1.RawValue
	SubjectPublicKeyInfo subjectPublicKeyInfo
}

type subjectPublicKeyInfo struct {
	Algorithm pkix.AlgorithmIdentifier
	PublicKey asn1.BitString
}

// CertInfo 从 X.509 DER 中解析出的最小字段集。
type CertInfo struct {
	// Pub65 未压缩公钥（0x04 || X || Y，65 字节）
	Pub65 []byte
	// IssuerDER 签发方 Name 的完整 DER（用于 CMS IssuerAndSerialNumber）
	IssuerDER []byte
	// SerialNumber 证书序列号
	SerialNumber *big.Int
	// RawDER 证书完整 DER
	RawDER []byte
}

// ParseCertFromDER 解析 X.509 证书（支持国密证书）。
func ParseCertFromDER(der []byte) (*CertInfo, error) {
	var cert x509Certificate
	if rest, err := asn1.Unmarshal(der, &cert); err != nil {
		return nil, ErrInvalidInput("x509: 解析失败: " + err.Error())
	} else if len(rest) != 0 {
		return nil, ErrInvalidInput("x509: 证书后存在多余数据")
	}
	pub := cert.TBS.SubjectPublicKeyInfo.PublicKey
	// BIT STRING 内容首字节为 unused-bits 计数（DER 中为 0）
	raw := pub.RightAlign()
	if len(raw) != 65 || raw[0] != 0x04 {
		return nil, ErrInvalidInput("x509: 期望 65 字节未压缩 SM2 公钥")
	}
	return &CertInfo{
		Pub65:        append([]byte(nil), raw...),
		IssuerDER:    append([]byte(nil), cert.TBS.Issuer.FullBytes...),
		SerialNumber: cert.TBS.SerialNumber,
		RawDER:       append([]byte(nil), der...),
	}, nil
}

// ParseCertFromBase64DER 解析 Base64(DER) 形式的证书（aipay.env 中 AIPAY_PUBLIC_KEY 的格式）。
func ParseCertFromBase64DER(b64 string) (*CertInfo, error) {
	der, err := Base64Decode(b64)
	if err != nil {
		return nil, ErrInvalidInput("x509: 证书 Base64 非法: " + err.Error())
	}
	return ParseCertFromDER(der)
}
