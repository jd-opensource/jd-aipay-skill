// 极简 DER(TLV) 编解码辅助：用于手工组装/解析京东私有信封与 CMS 结构，
// 避免 encoding/asn1 在字段可选性（算法参数缺省 vs NULL）上的差异。
package aipay

import (
	"encoding/asn1"
	"encoding/hex"
	"math/big"
)

// HexEncode 字节转小写 hex 字符串。
func HexEncode(b []byte) string { return hex.EncodeToString(b) }

func hexEncode(b []byte) string { return hex.EncodeToString(b) }

// HexDecode hex 字符串转字节。
func HexDecode(s string) ([]byte, error) { return hex.DecodeString(s) }

// derLength 编码 DER 长度字段。
func derLength(n int) []byte {
	if n < 0x80 {
		return []byte{byte(n)}
	}
	var buf []byte
	for v := n; v > 0; v >>= 8 {
		buf = append([]byte{byte(v)}, buf...)
	}
	return append([]byte{byte(0x80 | len(buf))}, buf...)
}

// DerTLV 组装单个 TLV。
func DerTLV(tag byte, content []byte) []byte {
	return append([]byte{tag}, append(derLength(len(content)), content...)...)
}

// DerSeq SEQUENCE(content)
func DerSeq(items ...[]byte) []byte { return DerTLV(0x30, concat(items...)) }

// DerSet SET(content)（单元素；多元素调用方需自行保证 DER 排序）
func DerSet(items ...[]byte) []byte { return DerTLV(0x31, concat(items...)) }

// DerOctetString OCTET STRING
func DerOctetString(b []byte) []byte { return DerTLV(0x04, b) }

// DerOID OBJECT IDENTIFIER
func DerOID(oid string) []byte {
	o, err := parseOID(oid)
	if err != nil {
		panic("aipay: 非法 OID: " + oid)
	}
	tlv, err2 := asn1.Marshal(o)
	if err2 != nil {
		panic("aipay: OID 编码失败: " + oid)
	}
	return tlv
}

// DerAlgorithmIdentifier 算法标识 SEQUENCE{OID}（参数缺省，与 asn1crypto 行为一致）
func DerAlgorithmIdentifier(oid string) []byte { return DerSeq(DerOID(oid)) }

// DerInteger big.Int -> INTEGER TLV
func DerInteger(v *big.Int) []byte {
	tlv, err := asn1.Marshal(v)
	if err != nil {
		panic("aipay: INTEGER 编码失败")
	}
	return tlv
}

// DerInt int -> INTEGER TLV
func DerInt(v int) []byte { return DerInteger(big.NewInt(int64(v))) }

// DerRawInteger 直接按大端字节编码非负整数（自动补高位 0）
func DerRawInteger(b []byte) []byte {
	v := new(big.Int).SetBytes(b)
	return DerInteger(v)
}

func parseOID(s string) (asn1.ObjectIdentifier, error) {
	var out asn1.ObjectIdentifier
	part := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == '.' {
			v := 0
			for _, c := range s[part:i] {
				if c < '0' || c > '9' {
					return nil, ErrInvalidInput("oid: 非法字符")
				}
				v = v*10 + int(c-'0')
			}
			out = append(out, v)
			part = i + 1
		}
	}
	return out, nil
}

// tlvNode 解析出的单个 TLV 节点
type tlvNode struct {
	Tag     byte
	Content []byte
	Full    []byte
	// Children 仅当 Tag 为构造类型时按需解析
}

// parseTLVs 解析一段 DER 中的连续 TLV。
func parseTLVs(data []byte) ([]tlvNode, error) {
	var out []tlvNode
	off := 0
	for off < len(data) {
		node, next, err := parseTLV(data, off)
		if err != nil {
			return nil, err
		}
		out = append(out, node)
		off = next
	}
	return out, nil
}

func parseTLV(data []byte, off int) (tlvNode, int, error) {
	if off+2 > len(data) {
		return tlvNode{}, 0, ErrInvalidInput("der: 数据截断")
	}
	tag := data[off]
	idx := off + 1
	l := int(data[idx])
	idx++
	if l&0x80 != 0 {
		n := l & 0x7f
		if n == 0 || n > 4 || idx+n > len(data) {
			return tlvNode{}, 0, ErrInvalidInput("der: 长度字段非法")
		}
		l = 0
		for i := 0; i < n; i++ {
			l = l<<8 | int(data[idx+i])
		}
		idx += n
	}
	if idx+l > len(data) {
		return tlvNode{}, 0, ErrInvalidInput("der: 内容越界")
	}
	return tlvNode{Tag: tag, Content: data[idx : idx+l], Full: data[off : idx+l]}, idx + l, nil
}

// unwrapContext 取 context-specific 标签里的内容字节。
// 兼容构造（0xA0-0xAF，EXPLICIT/IMPLICIT 构造）与原语（0x80-0x8F，
// PKCS7 EncryptedContentInfo 的 [0] IMPLICIT OCTET STRING 常编码为 0x80）两种形态。
func unwrapContext(node tlvNode) ([]byte, error) {
	if node.Tag&0x80 == 0 {
		return nil, ErrInvalidInput("der: 期望 context 标签")
	}
	return node.Content, nil
}

func concat(parts ...[]byte) []byte {
	n := 0
	for _, p := range parts {
		n += len(p)
	}
	out := make([]byte, 0, n)
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

func equalBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
