// PKCS#12 (PFX) 解析：从商户 PFX 中提取 SM2 私钥标量 d 与 X.509 证书 DER。
// 与 Java/Python 版行为对齐；SM2 证书不被 Go 标准库支持，故手工解析 ASN.1。
// 支持 PBE 算法：pbeWithSHAAnd3-KeyTripleDES-CBC / pbeWithSHAAnd40BitRC2-CBC /
// PBES2(PBKDF2 + AES-128/192/256-CBC)。
package aipay

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/des"
	"crypto/sha1"
	"crypto/sha256"
	"hash"
	"math/big"
)

// PfxContent PFX 解析结果。
type PfxContent struct {
	// PrivateKeyD SM2 私钥标量 d（32 字节大端）
	PrivateKeyD []byte
	// CertDER X.509 证书 DER
	CertDER []byte
	// Cert 解析后的证书字段（公钥、签发方、序列号）
	Cert *CertInfo
}

// ParsePfxFromBase64 解析 Base64 形式的 PFX（aipay.env 中 AIPAY_MERCHANT_PFX 的格式）。
func ParsePfxFromBase64(pfxBase64, password string) (*PfxContent, error) {
	raw, err := Base64Decode(pfxBase64)
	if err != nil {
		return nil, ErrInvalidInput("pfx: Base64 非法: " + err.Error())
	}
	return ParsePfx(raw, password)
}

// ParsePfx 解析 PFX 字节。
func ParsePfx(pfxBytes []byte, password string) (*PfxContent, error) {
	pfxSeq, err := expectSeq(pfxBytes)
	if err != nil {
		return nil, ErrInvalidInput("pfx: 顶层结构非法: " + err.Error())
	}
	kids, err := parseTLVs(pfxSeq.Content)
	if err != nil || len(kids) < 2 {
		return nil, ErrInvalidInput("pfx: 顶层字段缺失")
	}
	// kids[0]=version(3), kids[1]=authSafe ContentInfo, kids[2]=macData(忽略)
	authSafe, err := parseContentInfo(kids[1])
	if err != nil {
		return nil, ErrInvalidInput("pfx: authSafe 解析失败: " + err.Error())
	}
	if !authSafe.oidEqualTo("1.2.840.113549.1.7.1") {
		return nil, ErrInvalidInput("pfx: 不支持的外层 content type（期望 data）")
	}
	// data 的内容是 OCTET STRING，内含 AuthenticatedSafe（SEQUENCE OF ContentInfo）
	inner, _, err := parseTLV(authSafe.content, 0)
	if err != nil || inner.Tag != 0x04 {
		return nil, ErrInvalidInput("pfx: authSafe 内容应为 OCTET STRING")
	}
	authSafeSeq, err := expectSeq(inner.Content)
	if err != nil {
		return nil, ErrInvalidInput("pfx: AuthenticatedSafe 应为 SEQUENCE")
	}
	safeInfos, err := parseTLVs(authSafeSeq.Content)
	if err != nil {
		return nil, ErrInvalidInput("pfx: AuthenticatedSafe 解析失败")
	}

	out := &PfxContent{}
	for _, ci := range safeInfos {
		info, err := parseContentInfo(ci)
		if err != nil {
			continue
		}
		var safeContents []byte
		switch {
		case info.oidEqualTo("1.2.840.113549.1.7.1"): // data
			node, _, err := parseTLV(info.content, 0)
			if err != nil || node.Tag != 0x04 {
				return nil, ErrInvalidInput("pfx: data 内容应为 OCTET STRING")
			}
			safeContents = node.Content
		case info.oidEqualTo("1.2.840.113549.1.7.6"): // encryptedData
			plain, err := decryptEncryptedData(info.content, password)
			if err != nil {
				return nil, err
			}
			safeContents = plain
		default:
			continue
		}
		if err := collectSafeBags(safeContents, password, out); err != nil {
			return nil, err
		}
	}
	if out.PrivateKeyD == nil || out.CertDER == nil {
		return nil, ErrInvalidInput("pfx: 缺少私钥或证书")
	}
	cert, err := ParseCertFromDER(out.CertDER)
	if err != nil {
		return nil, err
	}
	out.Cert = cert
	return out, nil
}

type contentInfoNode struct {
	oid     string
	content []byte // [0] EXPLICIT 包装内的值 TLV 的完整字节
}

func (c *contentInfoNode) oidEqualTo(s string) bool { return c.oid == s }

// parseContentInfo 解析 ContentInfo ::= SEQ{ OID, [0] EXPLICIT ANY }
func parseContentInfo(node tlvNode) (*contentInfoNode, error) {
	if node.Tag != 0x30 {
		return nil, ErrInvalidInput("contentInfo: 期望 SEQUENCE")
	}
	kids, err := parseTLVs(node.Content)
	if err != nil || len(kids) < 2 {
		return nil, ErrInvalidInput("contentInfo: 字段缺失")
	}
	oid, err := oidToString(kids[0])
	if err != nil {
		return nil, err
	}
	inner, err := unwrapContext(kids[1])
	if err != nil {
		return nil, err
	}
	value, _, err := parseTLV(inner, 0)
	if err != nil {
		return nil, err
	}
	return &contentInfoNode{oid: oid, content: value.Full}, nil
}

// collectSafeBags 解析 SafeContents（SEQUENCE OF SafeBag），提取私钥与证书。
func collectSafeBags(safeContents []byte, password string, out *PfxContent) error {
	seq, err := expectSeq(safeContents)
	if err != nil {
		return ErrInvalidInput("pfx: SafeContents 应为 SEQUENCE")
	}
	bags, err := parseTLVs(seq.Content)
	if err != nil {
		return ErrInvalidInput("pfx: SafeBag 列表解析失败")
	}
	for _, bag := range bags {
		kids, err := parseTLVs(bag.Content)
		if err != nil || len(kids) < 2 {
			continue
		}
		bagID, err := oidToString(kids[0])
		if err != nil {
			continue
		}
		valueInner, err := unwrapContext(kids[1])
		if err != nil {
			continue
		}
		value, _, err := parseTLV(valueInner, 0)
		if err != nil {
			continue
		}
		switch bagID {
		case "1.2.840.113549.1.12.10.1.3": // certBag
			ckids, err := parseTLVs(value.Content)
			if err != nil || len(ckids) < 2 {
				continue
			}
			// certValue [0] EXPLICIT OCTET STRING(cert DER)
			cv, err := unwrapContext(ckids[1])
			if err != nil {
				continue
			}
			der, _, err := parseTLV(cv, 0)
			if err != nil || der.Tag != 0x04 {
				continue
			}
			out.CertDER = append([]byte(nil), der.Content...)
		case "1.2.840.113549.1.12.10.1.2": // pkcs8ShroudedKeyBag
			// value = EncryptedPrivateKeyInfo SEQ{ alg, OCTET STRING }
			vkids, err := parseTLVs(value.Content)
			if err != nil || len(vkids) < 2 {
				continue
			}
			plain, err := decryptPBE(vkids[0], vkids[1].Content, password)
			if err != nil {
				return err
			}
			d, err := extractDFromPKCS8(plain)
			if err != nil {
				return err
			}
			out.PrivateKeyD = d
		case "1.2.840.113549.1.12.10.1.1": // keyBag
			d, err := extractDFromPKCS8(value.Content)
			if err != nil {
				return err
			}
			out.PrivateKeyD = d
		}
	}
	return nil
}

// decryptEncryptedData 解析并解密 PKCS7 EncryptedData，返回明文 SafeContents。
func decryptEncryptedData(node []byte, password string) ([]byte, error) {
	seq, err := expectSeq(node)
	if err != nil {
		return nil, ErrInvalidInput("pfx: EncryptedData 应为 SEQUENCE")
	}
	kids, err := parseTLVs(seq.Content)
	if err != nil || len(kids) < 2 {
		return nil, ErrInvalidInput("pfx: EncryptedData 字段缺失")
	}
	// kids[0]=version, kids[1]=EncryptedContentInfo SEQ{ contentType, alg, [0] content }
	eciKids, err := parseTLVs(kids[1].Content)
	if err != nil || len(eciKids) < 3 {
		return nil, ErrInvalidInput("pfx: EncryptedContentInfo 字段缺失")
	}
	encContent, err := unwrapContext(eciKids[2])
	if err != nil {
		return nil, err
	}
	return decryptPBE(eciKids[1], encContent, password)
}

// decryptPBE 按算法标识解密 PBE 加密数据并去 PKCS7 填充。
func decryptPBE(algNode tlvNode, data []byte, password string) ([]byte, error) {
	kids, err := parseTLVs(algNode.Content)
	if err != nil || len(kids) < 1 {
		return nil, ErrInvalidInput("pbe: 算法标识非法")
	}
	oid, err := oidToString(kids[0])
	if err != nil {
		return nil, err
	}
	switch oid {
	case "1.2.840.113549.1.12.1.3": // pbeWithSHAAnd3-KeyTripleDES-CBC
		salt, iter, err := parsePBES1Params(kids)
		if err != nil {
			return nil, err
		}
		key := pkcs12KDF(password, salt, iter, 1, 24)
		iv := pkcs12KDF(password, salt, iter, 2, 8)
		return desCBCDecrypt(key, iv, data)
	case "1.2.840.113549.1.12.1.6": // pbeWithSHAAnd40BitRC2-CBC
		salt, iter, err := parsePBES1Params(kids)
		if err != nil {
			return nil, err
		}
		key := pkcs12KDF(password, salt, iter, 1, 5)
		iv := pkcs12KDF(password, salt, iter, 2, 8)
		return rc2CBCDecrypt40(key, iv, data)
	case "1.2.840.113549.1.5.13": // PBES2
		return decryptPBES2(kids, data, password)
	}
	return nil, ErrInvalidInput("pbe: 不支持的算法 " + oid)
}

func parsePBES1Params(kids []tlvNode) (salt []byte, iter int, err error) {
	if len(kids) < 2 {
		return nil, 0, ErrInvalidInput("pbe: 缺少参数")
	}
	params, err := parseTLVs(kids[1].Content)
	if err != nil || len(params) < 2 || params[0].Tag != 0x04 {
		return nil, 0, ErrInvalidInput("pbe: 参数非法")
	}
	iter = int(new(big.Int).SetBytes(params[1].Content).Int64())
	return params[0].Content, iter, nil
}

func decryptPBES2(kids []tlvNode, data []byte, password string) ([]byte, error) {
	if len(kids) < 2 {
		return nil, ErrInvalidInput("pbes2: 缺少参数")
	}
	outer, err := parseTLVs(kids[1].Content)
	if err != nil || len(outer) < 2 {
		return nil, ErrInvalidInput("pbes2: 参数非法")
	}
	kdfOID, err := oidToString(outer[0])
	if err != nil {
		return nil, err
	}
	if kdfOID != "1.2.840.113549.1.5.12" { // PBKDF2
		return nil, ErrInvalidInput("pbes2: 不支持的 KDF")
	}
	kp, err := parseTLVs(outer[1].Content)
	if err != nil || len(kp) < 2 || kp[0].Tag != 0x04 {
		return nil, ErrInvalidInput("pbes2: PBKDF2 参数非法")
	}
	salt := kp[0].Content
	iter := int(new(big.Int).SetBytes(kp[1].Content).Int64())
	keyLen := 0
	prf := "1.2.840.113549.2.7" // 默认 hMAC-SHA1
	for _, p := range kp[2:] {
		if p.Tag == 0x02 { // keyLength INTEGER
			keyLen = int(new(big.Int).SetBytes(p.Content).Int64())
		} else if p.Tag == 0x30 { // prf AlgorithmIdentifier
			ids, _ := parseTLVs(p.Content)
			if len(ids) > 0 {
				if s, err := oidToString(ids[0]); err == nil {
					prf = s
				}
			}
		}
	}
	var h func() hash.Hash
	switch prf {
	case "1.2.840.113549.2.7":
		h = sha1.New
	case "1.2.840.113549.2.9":
		h = sha256.New
	default:
		return nil, ErrInvalidInput("pbes2: 不支持的 PRF " + prf)
	}
	if keyLen == 0 {
		return nil, ErrInvalidInput("pbes2: 缺少 keyLength")
	}
	key := pbkdf2Key([]byte(password), salt, iter, keyLen, h)

	// 第二个元素 = encryptionScheme
	encKids, err := parseTLVs(outer[2].Content)
	if err != nil || len(encKids) < 2 {
		return nil, ErrInvalidInput("pbes2: 加密方案参数非法")
	}
	encOID, err := oidToString(encKids[0])
	if err != nil {
		return nil, err
	}
	var blockSize int
	switch encOID {
	case "2.16.840.1.101.3.4.1.2":
		blockSize = 16 // aes128-CBC
	case "2.16.840.1.101.3.4.1.22":
		blockSize = 16 // aes192-CBC
	case "2.16.840.1.101.3.4.1.42":
		blockSize = 16 // aes256-CBC
	default:
		return nil, ErrInvalidInput("pbes2: 不支持的加密方案 " + encOID)
	}
	if encKids[1].Tag != 0x04 {
		return nil, ErrInvalidInput("pbes2: IV 非法")
	}
	block, err2 := aes.NewCipher(key)
	if err2 != nil {
		return nil, err2
	}
	return cbcDecrypt(block, encKids[1].Content, data, blockSize)
}

// extractDFromPKCS8 从 PKCS8 PrivateKeyInfo（内含 ECPrivateKey）提取 32 字节私钥标量。
func extractDFromPKCS8(plain []byte) ([]byte, error) {
	pki, err := expectSeq(plain)
	if err != nil {
		return nil, ErrInvalidInput("pkcs8: 结构非法")
	}
	kids, err := parseTLVs(pki.Content)
	if err != nil || len(kids) < 3 || kids[2].Tag != 0x04 {
		return nil, ErrInvalidInput("pkcs8: 字段缺失")
	}
	// kids[2].Content = ECPrivateKey DER
	ec, err := expectSeq(kids[2].Content)
	if err != nil {
		return nil, ErrInvalidInput("pkcs8: ECPrivateKey 非法")
	}
	eKids, err := parseTLVs(ec.Content)
	if err != nil || len(eKids) < 2 || eKids[1].Tag != 0x04 {
		return nil, ErrInvalidInput("pkcs8: ECPrivateKey 字段缺失")
	}
	d := eKids[1].Content
	if len(d) < 32 {
		padded := make([]byte, 32)
		copy(padded[32-len(d):], d)
		d = padded
	}
	return append([]byte(nil), d...), nil
}

// pkcs12KDF RFC 7292 附录 B 的 PKCS#12 口令派生（SHA-1）。
// purpose: 1=加密密钥 2=IV 3=MAC 密钥。
func pkcs12KDF(password string, salt []byte, iterations, purpose, keyLen int) []byte {
	pwdBytes := []byte{}
	if password != "" {
		for _, r := range password {
			pwdBytes = append(pwdBytes, byte(r>>8), byte(r))
		}
	}
	pwdBytes = append(pwdBytes, 0x00, 0x00)

	const u = 20 // SHA-1 输出长度
	const v = 64 // SHA-1 分组大小

	d := make([]byte, v)
	for i := range d {
		d[i] = byte(purpose)
	}
	fill := func(buf []byte, length int) []byte {
		if len(buf) == 0 {
			return nil
		}
		n := (length + len(buf) - 1) / len(buf)
		out := make([]byte, 0, n*len(buf))
		for i := 0; i < n; i++ {
			out = append(out, buf...)
		}
		return out[:length]
	}
	sLen := (len(salt) + v - 1) / v * v
	pLen := (len(pwdBytes) + v - 1) / v * v
	i := concat(fill(salt, sLen), fill(pwdBytes, pLen))

	c := (keyLen + u - 1) / u
	out := []byte{}
	for idx := 0; idx < c; idx++ {
		a := sha1Sum(concat(d, i))
		for k := 1; k < iterations; k++ {
			a = sha1Sum(a)
		}
		out = append(out, a...)
		b := fill(a, v)
		bInt := new(big.Int).SetBytes(b)
		newI := make([]byte, 0, len(i))
		for off := 0; off < len(i); off += v {
			chunk := new(big.Int).SetBytes(i[off : off+v])
			chunk.Add(chunk, bInt)
			chunk.Add(chunk, big.NewInt(1))
			chunk.And(chunk, new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), uint(8*v)), big.NewInt(1)))
			newI = append(newI, leftPad(chunk, v)...)
		}
		i = newI
	}
	return out[:keyLen]
}

func leftPad(v *big.Int, size int) []byte {
	b := v.Bytes()
	if len(b) >= size {
		return b[len(b)-size:]
	}
	out := make([]byte, size)
	copy(out[size-len(b):], b)
	return out
}

func sha1Sum(b []byte) []byte {
	s := sha1.Sum(b)
	return s[:]
}

func desCBCDecrypt(key, iv, data []byte) ([]byte, error) {
	block, err := des.NewTripleDESCipher(key)
	if err != nil {
		return nil, err
	}
	return cbcDecrypt(block, iv, data, des.BlockSize)
}

func cbcDecrypt(block cipher.Block, iv, data []byte, blockSize int) ([]byte, error) {
	if len(data) == 0 || len(data)%blockSize != 0 {
		return nil, ErrInvalidInput("cbc: 密文长度非法")
	}
	mode := cipher.NewCBCDecrypter(block, iv)
	out := make([]byte, len(data))
	mode.CryptBlocks(out, data)
	return pkcs7Unpad(out, blockSize)
}
