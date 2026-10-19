// Package aipay 提供京东 AI 付网关接入所需的全套国密能力（零第三方依赖）：
// SM3 摘要、SM4 分组加密、SM2 加解密/签名、京东私有 SM2 数字信封、
// PKCS#12(PFX) 解析、CMS(P7) 签名与 HMAC-SM3。
package aipay

// SM3 密码杂凑算法（GB/T 32905-2016）纯 Go 实现。
import "encoding/binary"

const sm3BlockSize = 64

var sm3IV = [8]uint32{
	0x7380166f, 0x4914b2b9, 0x172442d7, 0xda8a0600,
	0xa96f30bc, 0x163138aa, 0xe38dee4d, 0xb0fb0e4e,
}

// Sm3 计算 SM3 摘要，返回 32 字节。
func Sm3(data []byte) []byte {
	d := &sm3Digest{}
	d.reset()
	d.write(data)
	return d.sum()
}

// Sm3Hex 计算 SM3 摘要并返回小写 hex 字符串。
func Sm3Hex(data []byte) string {
	return hexEncode(Sm3(data))
}

type sm3Digest struct {
	state  [8]uint32
	buf    [sm3BlockSize]byte
	bufLen int
	total  uint64
}

func (d *sm3Digest) reset() {
	d.state = sm3IV
	d.bufLen = 0
	d.total = 0
}

func (d *sm3Digest) write(p []byte) {
	d.total += uint64(len(p))
	if d.bufLen > 0 {
		n := copy(d.buf[d.bufLen:], p)
		d.bufLen += n
		p = p[n:]
		if d.bufLen == sm3BlockSize {
			d.compress(d.buf[:])
			d.bufLen = 0
		}
	}
	for len(p) >= sm3BlockSize {
		d.compress(p[:sm3BlockSize])
		p = p[sm3BlockSize:]
	}
	if len(p) > 0 {
		d.bufLen = copy(d.buf[:], p)
	}
}

func (d *sm3Digest) compress(block []byte) {
	sm3Compress(&d.state, block)
}

func (d *sm3Digest) sum() []byte {
	// 填充：0x80 + 0x00... + 64bit 大端比特长度
	bitLen := d.total * 8
	pad := make([]byte, 0, sm3BlockSize*2)
	pad = append(pad, 0x80)
	for (d.bufLen+len(pad))%sm3BlockSize != 56 {
		pad = append(pad, 0x00)
	}
	var lenBytes [8]byte
	binary.BigEndian.PutUint64(lenBytes[:], bitLen)
	pad = append(pad, lenBytes[:]...)

	// 在副本上做最终压缩，避免破坏调用方的 d.buf
	final := make([]byte, 0, sm3BlockSize*2)
	final = append(final, d.buf[:d.bufLen]...)
	final = append(final, pad...)
	state := d.state
	for off := 0; off < len(final); off += sm3BlockSize {
		sm3Compress(&state, final[off:off+sm3BlockSize])
	}
	out := make([]byte, 32)
	for i, v := range state {
		binary.BigEndian.PutUint32(out[i*4:], v)
	}
	return out
}

func sm3Compress(state *[8]uint32, block []byte) {
	var w [68]uint32
	var w1 [64]uint32
	for i := 0; i < 16; i++ {
		w[i] = binary.BigEndian.Uint32(block[i*4:])
	}
	for i := 16; i < 68; i++ {
		w[i] = sm3P1(w[i-16]^w[i-9]^rotl32(w[i-3], 15)) ^ rotl32(w[i-13], 7) ^ w[i-6]
	}
	for i := 0; i < 64; i++ {
		w1[i] = w[i] ^ w[i+4]
	}
	a, b, c, dd, e, f, g, h := state[0], state[1], state[2], state[3], state[4], state[5], state[6], state[7]
	for j := 0; j < 64; j++ {
		t := uint32(0x79cc4519)
		if j >= 16 {
			t = 0x7a879d8a
		}
		ss1 := rotl32(rotl32(a, 12)+e+rotl32(t, uint(j)%32), 7)
		ss2 := ss1 ^ rotl32(a, 12)
		var ff, gg uint32
		if j < 16 {
			ff = a ^ b ^ c
			gg = e ^ f ^ g
		} else {
			ff = (a & b) | (a & c) | (b & c)
			gg = (e & f) | (^e & g)
		}
		tt1 := ff + dd + ss2 + w1[j]
		tt2 := gg + h + ss1 + w[j]
		dd = c
		c = rotl32(b, 9)
		b = a
		a = tt1
		h = g
		g = rotl32(f, 19)
		f = e
		e = sm3P0(tt2)
	}
	state[0] ^= a
	state[1] ^= b
	state[2] ^= c
	state[3] ^= dd
	state[4] ^= e
	state[5] ^= f
	state[6] ^= g
	state[7] ^= h
}

func sm3P0(x uint32) uint32 { return x ^ rotl32(x, 9) ^ rotl32(x, 17) }
func sm3P1(x uint32) uint32 { return x ^ rotl32(x, 15) ^ rotl32(x, 23) }

func rotl32(x uint32, n uint) uint32 {
	return (x << n) | (x >> (32 - n))
}
