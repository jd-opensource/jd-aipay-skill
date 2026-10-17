// SM2 椭圆曲线公钥密码算法（GB/T 32918-2016）纯 Go 实现。
// 使用 crypto/elliptic 的通用曲线实现承载 sm2p256v1 参数。
package aipay

import (
	"crypto/elliptic"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"math/big"
)

// Sm2Curve sm2p256v1 曲线参数
var Sm2Curve = elliptic.CurveParams{
	Name:    "sm2p256v1",
	P:       bigFromHex("FFFFFFFEFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF00000000FFFFFFFFFFFFFFFF"),
	N:       bigFromHex("FFFFFFFEFFFFFFFFFFFFFFFFFFFFFFFF7203DF6B21C6052B53BBF40939D54123"),
	B:       bigFromHex("28E9FA9E9D9F5E344D5A9E4BCF6509A7F39789F515AB8F92DDBCBD414D940E93"),
	Gx:      bigFromHex("32C4AE2C1F1981195F9904466A39C9948FE30BBFF2660BE1715A4589334C74C7"),
	Gy:      bigFromHex("BC3736A2F4F6779C59BDCEE36B692153D0A9877CC62A474002DF32E52139F0A0"),
	BitSize: 256,
}

// DefaultSm2UserID SM2 签名默认用户身份 ID（GB/T 32918 第 5 部分 A.1）
const DefaultSm2UserID = "1234567812345678"

// ErrInvalidInput 输入非法错误（用于包内统一错误判断）
type ErrInvalidInput string

func (e ErrInvalidInput) Error() string { return string(e) }

var errKDFFailure = errors.New("sm2: KDF 输出全零，需重试")

func bigFromHex(s string) *big.Int {
	v, ok := new(big.Int).SetString(s, 16)
	if !ok {
		panic("aipay: 非法曲线参数 hex: " + s)
	}
	return v
}

// Sm2KeyPair SM2 密钥对：d 为 32 字节私钥标量，Pub 为 65 字节未压缩公钥（0x04 前缀）。
type Sm2KeyPair struct {
	D   []byte
	Pub []byte
}

// Sm2GenerateKeyPair 生成随机 SM2 密钥对。
func Sm2GenerateKeyPair() (*Sm2KeyPair, error) {
	d, err := rand.Int(rand.Reader, new(big.Int).Sub(Sm2Curve.N, big.NewInt(1)))
	if err != nil {
		return nil, err
	}
	d.Add(d, big.NewInt(1)) // [1, n-1]
	x, y := Sm2Curve.ScalarBaseMult(d.Bytes())
	return &Sm2KeyPair{D: pad32(d), Pub: uncompressedPoint(x, y)}, nil
}

// Sm2PubFromD 由私钥标量推导 65 字节未压缩公钥。
func Sm2PubFromD(d []byte) []byte {
	x, y := Sm2Curve.ScalarBaseMult(new(big.Int).SetBytes(d).Bytes())
	return uncompressedPoint(x, y)
}

// uncompressedPoint 0x04 || X || Y（65 字节）
func uncompressedPoint(x, y *big.Int) []byte {
	return append([]byte{0x04}, pointToBytes(x, y)...)
}

func pointToBytes(x, y *big.Int) []byte {
	return append(pad32(x), pad32(y)...)
}

func pad32(v *big.Int) []byte {
	b := v.Bytes()
	if len(b) >= 32 {
		return b[len(b)-32:]
	}
	out := make([]byte, 32)
	copy(out[32-len(b):], b)
	return out
}

// sm2KDF GB/T 32918.4 KDF(Z, klen)（基于 SM3）
func sm2KDF(z []byte, klen int) ([]byte, error) {
	need := (klen + 31) / 32
	var out []byte
	var ct [4]byte
	for i := 1; i <= need; i++ {
		binary.BigEndian.PutUint32(ct[:], uint32(i))
		out = append(out, Sm3(append(z, ct[:]...))...)
	}
	out = out[:klen]
	for _, b := range out {
		if b != 0 {
			return out, nil
		}
	}
	return nil, errKDFFailure
}

// sm2EncryptC1C2C3 SM2 加密（C1C2C3 排列，C1 为 64 字节裸点，无 0x04 前缀）。
// pub 为 65 字节未压缩公钥（0x04 前缀）。
func sm2EncryptC1C2C3(pub []byte, msg []byte) ([]byte, error) {
	if len(pub) != 65 || pub[0] != 0x04 {
		return nil, ErrInvalidInput("sm2: 公钥必须为 65 字节未压缩点")
	}
	pb := new(big.Int).SetBytes(pub[1:33])
	pbY := new(big.Int).SetBytes(pub[33:65])
	for try := 0; try < 100; try++ {
		k, err := rand.Int(rand.Reader, new(big.Int).Sub(Sm2Curve.N, big.NewInt(1)))
		if err != nil {
			return nil, err
		}
		k.Add(k, big.NewInt(1))
		x1, y1 := Sm2Curve.ScalarBaseMult(k.Bytes())
		x2, y2 := Sm2Curve.ScalarMult(pb, pbY, k.Bytes())
		t, err := sm2KDF(append(pad32(x2), pad32(y2)...), len(msg))
		if err != nil {
			continue // KDF 全零重试
		}
		c2 := make([]byte, len(msg))
		for i := range msg {
			c2[i] = msg[i] ^ t[i]
		}
		c3 := Sm3(concat(pad32(x2), msg, pad32(y2)))
		return append(concat(pointToBytes(x1, y1), c2), c3...), nil
	}
	return nil, errors.New("sm2: 加密重试次数耗尽")
}

// sm2DecryptC1C2C3 SM2 解密（C1C2C3 排列，C1 为 64 字节裸点）。
func sm2DecryptC1C2C3(d []byte, cipher []byte) ([]byte, error) {
	if len(cipher) < 96 {
		return nil, ErrInvalidInput("sm2: 密文过短")
	}
	c1x := new(big.Int).SetBytes(cipher[:32])
	c1y := new(big.Int).SetBytes(cipher[32:64])
	if !Sm2Curve.IsOnCurve(c1x, c1y) {
		return nil, ErrInvalidInput("sm2: C1 不在曲线上")
	}
	x2, y2 := Sm2Curve.ScalarMult(c1x, c1y, d)
	c2 := cipher[64 : len(cipher)-32]
	c3 := cipher[len(cipher)-32:]
	t, err := sm2KDF(append(pad32(x2), pad32(y2)...), len(c2))
	if err != nil {
		return nil, err
	}
	msg := make([]byte, len(c2))
	for i := range c2 {
		msg[i] = c2[i] ^ t[i]
	}
	if !equalBytes(Sm3(concat(pad32(x2), msg, pad32(y2))), c3) {
		return nil, ErrInvalidInput("sm2: C3 校验失败（密文被篡改或私钥不符）")
	}
	return msg, nil
}

// Sm2ZA 计算 SM2 签名预处理值 ZA = SM3(ENTLA || ID || a || b || Gx || Gy || Px || Py)。
func Sm2ZA(userID string, pub []byte) []byte {
	id := []byte(userID)
	entla := []byte{byte(len(id) * 8 >> 8), byte(len(id) * 8)}
	a := pad32(Sm2Curve.P)
	a[len(a)-1] -= 3 // a = P - 3（末字节 FFFFFFFF → FFFFFFC）
	parts := concat(entla, id, a, pad32(Sm2Curve.B),
		pad32(Sm2Curve.Gx), pad32(Sm2Curve.Gy), pub[1:33], pub[33:65])
	return Sm3(parts)
}

// Sm2Sign SM2 数字签名：输入原始消息 M，内部完成 e = SM3(ZA||M)，返回 (r, s) 各 32 字节。
func Sm2Sign(d, pub []byte, msg []byte, userID string) (r, s []byte, err error) {
	za := Sm2ZA(userID, pub)
	e := new(big.Int).SetBytes(Sm3(concat(za, msg)))
	dv := new(big.Int).SetBytes(d)
	n := Sm2Curve.N
	one := big.NewInt(1)
	for try := 0; try < 100; try++ {
		k, err2 := rand.Int(rand.Reader, new(big.Int).Sub(n, one))
		if err2 != nil {
			return nil, nil, err2
		}
		k.Add(k, one)
		x1, _ := Sm2Curve.ScalarBaseMult(k.Bytes())
		rv := new(big.Int).Add(e, new(big.Int).Mod(x1, n))
		rv.Mod(rv, n)
		if rv.Sign() == 0 {
			continue
		}
		if new(big.Int).Add(rv, k).Cmp(n) == 0 {
			continue
		}
		// s = (1+d)^-1 * (k - r*d) mod n
		denom := new(big.Int).ModInverse(new(big.Int).Add(dv, one), n)
		if denom == nil {
			continue
		}
		sv := new(big.Int).Sub(k, new(big.Int).Mul(rv, dv))
		sv.Mul(sv, denom)
		sv.Mod(sv, n)
		if sv.Sign() == 0 {
			continue
		}
		return pad32(rv), pad32(sv), nil
	}
	return nil, nil, errors.New("sm2: 签名重试次数耗尽")
}

// Sm2Verify SM2 验签：sig 为 (r, s) 各 32 字节拼接。
func Sm2Verify(pub []byte, msg []byte, userID string, sig []byte) bool {
	if len(sig) != 64 {
		return false
	}
	za := Sm2ZA(userID, pub)
	e := new(big.Int).SetBytes(Sm3(concat(za, msg)))
	rv := new(big.Int).SetBytes(sig[:32])
	sv := new(big.Int).SetBytes(sig[32:])
	n := Sm2Curve.N
	if rv.Sign() <= 0 || rv.Cmp(n) >= 0 || sv.Sign() <= 0 || sv.Cmp(n) >= 0 {
		return false
	}
	pbx := new(big.Int).SetBytes(pub[1:33])
	pby := new(big.Int).SetBytes(pub[33:65])
	t := new(big.Int).Add(rv, sv)
	t.Mod(t, n)
	if t.Sign() == 0 {
		return false
	}
	sbx, sby := Sm2Curve.ScalarBaseMult(sv.Bytes())
	tx, ty := Sm2Curve.ScalarMult(pbx, pby, t.Bytes())
	x, _ := Sm2Curve.Add(sbx, sby, tx, ty)
	x1 := new(big.Int).Mod(x, n)
	expect := new(big.Int).Add(e, x1)
	expect.Mod(expect, n)
	return expect.Cmp(rv) == 0
}
