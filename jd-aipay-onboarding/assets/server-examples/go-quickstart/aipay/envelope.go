// 京东私有 SM2 数字信封（等价服务端 SM2EnvelopUtil2）：
//
//	Base64( version(1B=0x01) || DER(SEQ(c11 INTEGER, c12 INTEGER, c3 OCTET STRING, c2 OCTET STRING)) || IV(16B) || SM4-CBC-PKCS7(data, key, iv) )
//
// 其中 key/iv 各 16 字节随机；c11/c12 为 SM2 密文 C1 点坐标，c2=SM2 密文载荷，
// c3=SM3 MAC。请求/响应的 bizContent 均使用该信封。
package aipay

// Sm2EncryptEnvelop 用京东 SM2 公钥证书（Base64 DER）对 data 做数字信封加密，返回 Base64。
func Sm2EncryptEnvelop(data []byte, jdCertB64 string) (string, error) {
	cert, err := ParseCertFromBase64DER(jdCertB64)
	if err != nil {
		return "", err
	}
	return Sm2EncryptEnvelopWithCert(data, cert)
}

// Sm2EncryptEnvelopWithCert 同上，接收已解析证书（避免重复解析）。
func Sm2EncryptEnvelopWithCert(data []byte, cert *CertInfo) (string, error) {
	key := randomBytes(16)
	iv := randomBytes(16)

	cipher, err := sm2EncryptC1C2C3(cert.Pub65, key)
	if err != nil {
		return "", err
	}
	c1 := cipher[:64]
	c2Body := cipher[64 : len(cipher)-32]
	c3 := cipher[len(cipher)-32:]

	enkey := DerSeq(
		DerRawInteger(c1[:32]),
		DerRawInteger(c1[32:]),
		DerOctetString(c3),
		DerOctetString(c2Body),
	)
	result := concat([]byte{0x01}, enkey, iv, Sm4CBCEncrypt(key, iv, data))
	return Base64Encode(result), nil
}

// Sm2DecryptEnvelop 用商户私钥标量 d 解数字信封，返回明文。
func Sm2DecryptEnvelop(envelopB64 string, privateD []byte) ([]byte, error) {
	raw, err := Base64Decode(envelopB64)
	if err != nil {
		return nil, ErrInvalidInput("envelop: Base64 非法")
	}
	if len(raw) < 1+2+16+16 {
		return nil, ErrInvalidInput("envelop: 数据过短")
	}
	if raw[0] != 0x01 {
		return nil, ErrInvalidInput("envelop: 不支持的版本")
	}
	body := raw[1:]
	if body[0] != 0x30 {
		return nil, ErrInvalidInput("envelop: 期望 SEQUENCE")
	}
	enkey, next, err := parseTLV(body, 0)
	if err != nil {
		return nil, err
	}
	rest := body[next:]
	if len(rest) < 32 {
		return nil, ErrInvalidInput("envelop: IV/密文缺失")
	}
	iv := rest[:16]
	endata := rest[16:]

	kids, err := parseTLVs(enkey.Content)
	if err != nil || len(kids) != 4 {
		return nil, ErrInvalidInput("envelop: 信封密钥结构非法")
	}
	c11 := leftPad(bigIntFromBytes(kids[0].Content), 32)
	c12 := leftPad(bigIntFromBytes(kids[1].Content), 32)
	c2b := kids[3].Content
	c3b := kids[2].Content

	// 重组为 C1(64) || C2 || C3（无 0x04 前缀）
	sm2Cipher := concat(c11, c12, c2b, c3b)
	key, err := sm2DecryptC1C2C3(privateD, sm2Cipher)
	if err != nil {
		return nil, err
	}
	if len(key) != 16 {
		return nil, ErrInvalidInput("envelop: 解出的对称密钥长度非法")
	}
	return Sm4CBCDecrypt(key, iv, endata)
}
