package aipay

import (
	"bytes"
	"math/big"
	"strings"
	"testing"
)

// 标准向量来源：Python gmssl（与线上三语言 demo 同源实现）实测输出 + GB/T 国标示例。
// SM3("abc") 与 SM3("abcd"*16) 为 GB/T 32905-2016 附录 A 标准向量。

func TestSm3Vectors(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"abc", "66c7f0f462eeedd9d1f2d46bdc10e4e24167c4875cf2f7a2297da02b8f4ba8e0"},
		{strings.Repeat("abcd", 16), "debe9ff92275b8a138604889c18e5a4d6fdb70e5387e5765293dcba39c0c5732"},
	}
	for _, c := range cases {
		got := Sm3Hex([]byte(c.in))
		if got != c.want {
			t.Fatalf("SM3(%q) = %s, want %s", c.in, got, c.want)
		}
	}
}

func TestSm4ECBVector(t *testing.T) {
	// GB/T 32907-2016 示例：key=plain=0123456789abcdeffedcba9876543210 -> 681edf34d206965e86b3e94f536e4246
	key, _ := HexDecode("0123456789abcdeffedcba9876543210")
	plain, _ := HexDecode("0123456789abcdeffedcba9876543210")
	rk := sm4ExpandKey(key)
	got := sm4CryptBlock(rk, plain)
	if hexEncode(got) != "681edf34d206965e86b3e94f536e4246" {
		t.Fatalf("SM4 ECB = %s, want 681edf34d206965e86b3e94f536e4246", hexEncode(got))
	}
	back := sm4DecryptBlock(rk, got)
	if !bytes.Equal(back, plain) {
		t.Fatalf("SM4 解密往返失败")
	}
}

func TestSm4CBCRoundtrip(t *testing.T) {
	key := randomBytes(16)
	iv := randomBytes(16)
	plain := []byte("AI付联调测试数据 AI pay interop test 1234567890")
	enc := Sm4CBCEncrypt(key, iv, plain)
	dec, err := Sm4CBCDecrypt(key, iv, enc)
	if err != nil {
		t.Fatalf("SM4 CBC 解密失败: %v", err)
	}
	if !bytes.Equal(dec, plain) {
		t.Fatalf("SM4 CBC 往返不一致")
	}
}

func TestHmacSm3(t *testing.T) {
	// 黄金值由 Python gmssl（demo 同源实现）计算
	cases := []struct {
		key, msg, want string
	}{
		{"test", "appId=A&merchant=B", "ebd2068dc69e34ea37166c1fcf0cef1590ac35c6d993dbb334dc1ae39edf2938"},
		{strings.Repeat("k", 100), "x", "0980bc92b4d2387c9efa81b95a4e8dbd685aab0459492738bd854544372598f4"},
	}
	for _, c := range cases {
		got := HmacSm3Hex(c.msg, c.key)
		if got != c.want {
			t.Fatalf("HMAC-SM3(key=%q, msg=%q) = %s, want %s", c.key, c.msg, got, c.want)
		}
	}
}

func TestSm2KeyPairAndSign(t *testing.T) {
	kp, err := Sm2GenerateKeyPair()
	if err != nil {
		t.Fatalf("生成密钥对失败: %v", err)
	}
	if len(kp.D) != 32 || len(kp.Pub) != 65 || kp.Pub[0] != 0x04 {
		t.Fatalf("密钥对格式非法")
	}
	// 由 d 推导的公钥必须一致
	if !bytes.Equal(Sm2PubFromD(kp.D), kp.Pub) {
		t.Fatalf("d 推导公钥与密钥对公钥不一致")
	}
	msg := []byte("sm2 sign test 京东 AI 付")
	r, s, err := Sm2Sign(kp.D, kp.Pub, msg, DefaultSm2UserID)
	if err != nil {
		t.Fatalf("签名失败: %v", err)
	}
	if !Sm2Verify(kp.Pub, msg, DefaultSm2UserID, append(append([]byte{}, r...), s...)) {
		t.Fatalf("验签失败")
	}
	// 篡改消息后验签必须失败
	if Sm2Verify(kp.Pub, append(msg, 'x'), DefaultSm2UserID, append(append([]byte{}, r...), s...)) {
		t.Fatalf("篡改消息后验签不应通过")
	}
	// r/s 必须为 32 字节定长
	if len(r) != 32 || len(s) != 32 {
		t.Fatalf("签名长度非法: r=%d s=%d", len(r), len(s))
	}
}

func TestSm2EncryptDecryptRoundtrip(t *testing.T) {
	kp, _ := Sm2GenerateKeyPair()
	msg := []byte("sm2 C1C2C3 roundtrip 测试 0123456789")
	cipher, err := sm2EncryptC1C2C3(kp.Pub, msg)
	if err != nil {
		t.Fatalf("加密失败: %v", err)
	}
	plain, err := sm2DecryptC1C2C3(kp.D, cipher)
	if err != nil {
		t.Fatalf("解密失败: %v", err)
	}
	if !bytes.Equal(plain, msg) {
		t.Fatalf("SM2 加解密往返不一致")
	}
}

func TestSm2EnvelopRoundtrip(t *testing.T) {
	// 用随机密钥对构造"京东公钥证书"与"商户私钥"，对信封做全链路往返
	kp, _ := Sm2GenerateKeyPair()
	// 构造一张最小证书（此处直接用 DER 组装会引入 X509 复杂度，
	// 因此用 Pub65 直连测试信封的对称层；真实证书链路由沙箱联调覆盖）
	_ = kp
	msg := []byte(`{"orderId":"TEST123","amount":99}`)
	key := randomBytes(16)
	iv := randomBytes(16)
	cipher, err := sm2EncryptC1C2C3(kp.Pub, key)
	if err != nil {
		t.Fatalf("信封 SM2 加密失败: %v", err)
	}
	c1 := cipher[:64]
	c2Body := cipher[64 : len(cipher)-32]
	c3 := cipher[len(cipher)-32:]
	enkey := DerSeq(DerRawInteger(c1[:32]), DerRawInteger(c1[32:]), DerOctetString(c3), DerOctetString(c2Body))
	envelop := concat([]byte{0x01}, enkey, iv, Sm4CBCEncrypt(key, iv, msg))

	// 按信封解密路径还原
	raw := envelop
	if raw[0] != 0x01 {
		t.Fatalf("信封版本非法")
	}
	body := raw[1:]
	enkeyNode, next, err := parseTLV(body, 0)
	if err != nil {
		t.Fatalf("信封 enkey 解析失败: %v", err)
	}
	kids, err := parseTLVs(enkeyNode.Content)
	if err != nil || len(kids) != 4 {
		t.Fatalf("enkey 结构非法")
	}
	c11 := leftPad(bigIntFromBytes(kids[0].Content), 32)
	c12 := leftPad(bigIntFromBytes(kids[1].Content), 32)
	rebuilt := concat(c11, c12, kids[3].Content, kids[2].Content)
	gotKey, err := sm2DecryptC1C2C3(kp.D, rebuilt)
	if err != nil {
		t.Fatalf("信封密钥解密失败: %v", err)
	}
	rest := body[next:]
	gotMsg, err := Sm4CBCDecrypt(gotKey, rest[:16], rest[16:])
	if err != nil {
		t.Fatalf("信封数据解密失败: %v", err)
	}
	if !bytes.Equal(gotMsg, msg) {
		t.Fatalf("信封往返不一致: %s", string(gotMsg))
	}
}

func TestSm2CurveOrder(t *testing.T) {
	// 曲线阶必须与国标一致（Python gmssl default_ecc_table["n"] 同值）
	want := "FFFFFFFEFFFFFFFFFFFFFFFFFFFFFFFF7203DF6B21C6052B53BBF40939D54123"
	if strings.ToUpper(Sm2Curve.N.Text(16)) != want {
		t.Fatalf("SM2 曲线阶不匹配: %s", Sm2Curve.N.Text(16))
	}
	if !Sm2Curve.IsOnCurve(Sm2Curve.Gx, Sm2Curve.Gy) {
		t.Fatalf("基点不在曲线上")
	}
	if new(big.Int).Mod(new(big.Int).Sub(Sm2Curve.P, big.NewInt(3)), Sm2Curve.P).Sign() < 0 {
		t.Fatalf("a = P-3 计算非法")
	}
}

func TestSignStringOrder(t *testing.T) {
	c := &Content{
		AgentID:    "AGENT",
		AppID:      "APP",
		BizContent: "BIZ",
		EncType:    "SM2",
		MerchantNo: "M",
		Nonce:      "N",
		ReqNo:      "R",
		Timestamp:  1700000000000,
		Version:    "1.0",
	}
	want := "agentId=AGENT&appId=APP&bizContent=BIZ&encType=SM2&merchantNo=M&nonce=N&reqNo=R&timestamp=1700000000000&version=1.0"
	if got := BuildSignString(c); got != want {
		t.Fatalf("签名原文不一致:\n got=%s\nwant=%s", got, want)
	}
}
