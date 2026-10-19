// CMS (PKCS7) SignedData：SM3WithSM2 签名（directSignature=true、attachFlag=true）
// 的构造与明文提取，等价京东 wyaks-security 的对应能力。
//
// 结构（与 Java/Python 版逐字段对齐）：
//
//	ContentInfo(signedData) {
//	  SignedData {
//	    version = 1,
//	    digestAlgorithms = { SM3 (1.2.156.10197.1.401) },
//	    encapContentInfo = { contentType = data(1.2.840.113549.1.7.1), eContent = srcData },
//	    certificates = [商户证书],
//	    signerInfos = [{ version=1, sid=IssuerAndSerialNumber, digestAlg=SM3,
//	                     signatureAlgorithm=SM3WithSM2(1.2.156.10197.1.501),
//	                     signature = SM2Sign(SM3(ZA||srcData)) }]
//	  }
//	}
package aipay

const (
	oidData       = "1.2.840.113549.1.7.1"
	oidSignedData = "1.2.840.113549.1.7.2"
	oidSM3        = "1.2.156.10197.1.401"
	oidSM3WithSM2 = "1.2.156.10197.1.501"
)

// Sm2P7SignAttached 生成附带证书的 CMS SignedData，返回 DER。
func Sm2P7SignAttached(srcData []byte, pfx *PfxContent) ([]byte, error) {
	// e = SM3(ZA || M)：Sm2Sign 内部按 GB/T 32918 完成 ZA 预处理
	r2, s2, err := Sm2Sign(pfx.PrivateKeyD, pfx.Cert.Pub65, srcData, DefaultSm2UserID)
	if err != nil {
		return nil, err
	}
	// ⚠️ 网关（wyaks/BC）要求签名为 DER SEQUENCE{r INTEGER, s INTEGER}（与 Java/Python 版一致），
	// 不能放裸 64 字节 r||s，否则网关解签异常（报"aks解密过程异常"）。
	sig := DerSeq(DerRawInteger(r2), DerRawInteger(s2))

	digestAlg := DerAlgorithmIdentifier(oidSM3)
	sigAlg := DerAlgorithmIdentifier(oidSM3WithSM2)
	sid := DerSeq(pfx.Cert.IssuerDER, DerInteger(pfx.Cert.SerialNumber))
	signerInfo := DerSeq(
		DerInt(1), // version
		sid,
		digestAlg,
		sigAlg,
		DerOctetString(sig),
	)
	encap := DerSeq(
		DerOID(oidData),
		DerTLV(0xA0, DerOctetString(srcData)), // eContent [0] EXPLICIT OCTET STRING
	)
	// certificates [0] IMPLICIT SET OF CertificateChoices —— 证书裸 DER 直接入 context-0
	certificates := DerTLV(0xA0, pfx.Cert.RawDER)

	signedData := DerSeq(
		DerInt(1),          // version
		DerSet(digestAlg),  // digestAlgorithms
		encap,              // encapContentInfo
		certificates,       // certificates [0] IMPLICIT
		DerSet(signerInfo), // signerInfos
	)
	return DerSeq(DerOID(oidSignedData), DerTLV(0xA0, signedData)), nil
}

// CmsExtractContent 从 CMS SignedData DER 中提取 encapContentInfo.eContent 明文。
func CmsExtractContent(cmsDER []byte) ([]byte, error) {
	root, err := expectSeq(cmsDER)
	if err != nil {
		return nil, err
	}
	kids, err := parseTLVs(root.Content)
	if err != nil || len(kids) < 2 {
		return nil, ErrInvalidInput("cms: 结构非法")
	}
	oid, err := oidToString(kids[0])
	if err != nil {
		return nil, err
	}
	if oid != oidSignedData {
		return nil, ErrInvalidInput("cms: 期望 signedData")
	}
	inner, err := unwrapContext(kids[1])
	if err != nil {
		return nil, err
	}
	sd, err := expectSeq(inner)
	if err != nil {
		return nil, err
	}
	sdKids, err := parseTLVs(sd.Content)
	if err != nil || len(sdKids) < 3 {
		return nil, ErrInvalidInput("cms: SignedData 字段缺失")
	}
	// sdKids[0]=version, sdKids[1]=digestAlgorithms, sdKids[2]=encapContentInfo
	encapKids, err := parseTLVs(sdKids[2].Content)
	if err != nil || len(encapKids) < 2 {
		return nil, ErrInvalidInput("cms: encapContentInfo 非法")
	}
	// eContent = [0] EXPLICIT OCTET STRING
	ec, err := unwrapContext(encapKids[1])
	if err != nil {
		return nil, ErrInvalidInput("cms: eContent 缺失")
	}
	os, _, err := parseTLV(ec, 0)
	if err != nil || os.Tag != 0x04 {
		return nil, ErrInvalidInput("cms: eContent 应为 OCTET STRING")
	}
	return os.Content, nil
}
