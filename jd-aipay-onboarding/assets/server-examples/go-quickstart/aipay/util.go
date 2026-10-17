// 包内通用小工具：Base64、TLV 定位、PBKDF2、OID 转换。
package aipay

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/asn1"
	"encoding/base64"
	"hash"
	"math/big"
)

// hmacHash HMAC 底层哈希构造器类型。
type hmacHash func() hash.Hash

var _ = sha256.New // 保留引用（PBES2 PRF 用）

// randomBytes 生成 n 个密码学安全随机字节。
func randomBytes(n int) []byte {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic("aipay: 随机数生成失败: " + err.Error())
	}
	return b
}

func bigIntFromBytes(b []byte) *big.Int { return new(big.Int).SetBytes(b) }

// Base64Decode 标准 Base64 解码（容忍空白与换行）。
func Base64Decode(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(compactBase64(s))
}

// Base64Encode 标准 Base64 编码。
func Base64Encode(b []byte) string { return base64.StdEncoding.EncodeToString(b) }

func compactBase64(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '\n' || c == '\r' || c == ' ' || c == '\t' {
			continue
		}
		out = append(out, c)
	}
	return string(out)
}

// expectSeq 校验节点为 SEQUENCE 并返回。
func expectSeq(data []byte) (tlvNode, error) {
	node, _, err := parseTLV(data, 0)
	if err != nil {
		return tlvNode{}, err
	}
	if node.Tag != 0x30 {
		return tlvNode{}, ErrInvalidInput("der: 期望 SEQUENCE")
	}
	return node, nil
}

// oidToString 将 OID TLV 转为点分十进制字符串。
func oidToString(node tlvNode) (string, error) {
	if node.Tag != 0x06 {
		return "", ErrInvalidInput("der: 期望 OID")
	}
	var oid asn1.ObjectIdentifier
	if rest, err := asn1.Unmarshal(node.Full, &oid); err != nil || len(rest) != 0 {
		return "", ErrInvalidInput("der: OID 解析失败")
	}
	s := ""
	for i, v := range oid {
		if i > 0 {
			s += "."
		}
		s += itoa(v)
	}
	return s, nil
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}

// pbkdf2Key 标准 PBKDF2（RFC 2898）。
func pbkdf2Key(password, salt []byte, iter, keyLen int, h hmacHash) []byte {
	prf := hmac.New(h, password)
	hashLen := prf.Size()
	numBlocks := (keyLen + hashLen - 1) / hashLen

	var buf [4]byte
	dk := make([]byte, 0, numBlocks*hashLen)
	u := make([]byte, hashLen)
	for block := 1; block <= numBlocks; block++ {
		prf.Reset()
		prf.Write(salt)
		buf[0] = byte(block >> 24)
		buf[1] = byte(block >> 16)
		buf[2] = byte(block >> 8)
		buf[3] = byte(block)
		prf.Write(buf[:4])
		dk = prf.Sum(dk)
		t := dk[len(dk)-hashLen:]
		copy(u, t)
		for n := 2; n <= iter; n++ {
			prf.Reset()
			prf.Write(u)
			u = u[:0]
			u = prf.Sum(u)
			for x := range u {
				t[x] ^= u[x]
			}
		}
	}
	return dk[:keyLen]
}
