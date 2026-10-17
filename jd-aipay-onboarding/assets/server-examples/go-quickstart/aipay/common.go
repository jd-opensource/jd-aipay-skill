// AI 付网关公共辅助：HMAC-SM3 签名、content 组装、HTTP 调用、响应 bizContent 解密。
// 与 Java/Python/Node.js 版行为逐字段对齐（tradeAmount/timestamp 为 JSON 数字类型，
// 与官方最新模板一致；签名原文按字段值的十进制字符串参与计算，数字与字符串同值同签）。
package aipay

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Content 请求 content 层（字段按 ASCII 升序声明，序列化顺序即协议顺序）。
// Timestamp 使用字符串形态（与 Python 版一致；tradeAmount 仍为 JSON 数字）。
type Content struct {
	AgentID    string `json:"agentId"`
	AppID      string `json:"appId"`
	BizContent string `json:"bizContent"`
	EncType    string `json:"encType"`
	MerchantNo string `json:"merchantNo"`
	Nonce      string `json:"nonce"`
	ReqNo      string `json:"reqNo"`
	SignType   string `json:"signType"`
	Timestamp  int64  `json:"timestamp"`
	Version    string `json:"version"`
	Sign       string `json:"sign,omitempty"`
}

// contentBody 请求根结构 {"data":{"content":{...}}}
type contentBody struct {
	Data struct {
		Content *Content `json:"content"`
	} `json:"data"`
}

// HmacSm3Hex HMAC-SM3，返回小写 hex 字符串（block size 64，长密钥先做一次 SM3）。
func HmacSm3Hex(stringToSign, secretKey string) string {
	key := []byte(secretKey)
	const blockSize = 64
	if len(key) > blockSize {
		key = Sm3(key)
	}
	if len(key) < blockSize {
		key = append(key, make([]byte, blockSize-len(key))...)
	}
	// 手工实现 HMAC（SM3 未适配标准库 hash 接口体系）
	ipad := make([]byte, blockSize)
	opad := make([]byte, blockSize)
	for i := 0; i < blockSize; i++ {
		ipad[i] = key[i] ^ 0x36
		opad[i] = key[i] ^ 0x5c
	}
	inner := Sm3(concat(ipad, []byte(stringToSign)))
	outer := Sm3(concat(opad, inner))
	return hexEncode(outer)
}

// BuildContent 组装 content 层字段（含随机 nonce/reqNo、毫秒时间戳）。
func BuildContent(bizContentEncrypted string, cfg *Config) *Content {
	return &Content{
		AgentID:    cfg.AgentID,
		AppID:      cfg.AppID,
		BizContent: bizContentEncrypted,
		EncType:    "SM2",
		MerchantNo: cfg.MerchantNo,
		Nonce:      RandomHex(16),
		ReqNo:      GenReqNo(),
		SignType:   "SM3",
		Timestamp:  time.Now().UnixNano() / int64(time.Millisecond),
		Version:    "1.0",
	}
}

// BuildSignString 拼 content 层签名原文：ASCII 升序 k=v&k=v；空值与 signType 不参与。
func BuildSignString(c *Content) string {
	fields := []struct{ k, v string }{
		{"agentId", c.AgentID},
		{"appId", c.AppID},
		{"bizContent", c.BizContent},
		{"encType", c.EncType},
		{"merchantNo", c.MerchantNo},
		{"nonce", c.Nonce},
		{"reqNo", c.ReqNo},
		{"timestamp", strconv.FormatInt(c.Timestamp, 10)},
		{"version", c.Version},
	}
	parts := make([]string, 0, len(fields))
	for _, f := range fields {
		if f.v == "" {
			continue
		}
		parts = append(parts, f.k+"="+f.v)
	}
	return strings.Join(parts, "&")
}

// BuildHttpHeaders 网关要求的 HTTP 头（⚠️ 头名必须全小写，网关大小写敏感）。
func BuildHttpHeaders(appID string) map[string][]string {
	return map[string][]string{
		"app-id":        {appID},
		"encrypt-type":  {"NONE"},
		"source-type":   {"H5"},
		"login-type":    {"0"},
		"cache-control": {"no-cache"},
		"content-type":  {"application/json"},
		"stream-type":   {"false"},
	}
}

// PostJSON 发送 POST（JSON），返回 "HTTP <code> | <body>"。
// ⚠️ 头名大小写敏感：直接写 Header map 键，绕过 Go 标准库的首字母大写规范化，
// 保证 app-id / stream-type 等以全小写原样发出（否则网关按 SSE 链路处理会报 10000403）。
func PostJSON(url string, headers map[string][]string, bodyJSON string, timeout time.Duration) (string, error) {
	tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}} // 与 Java/Python demo 行为一致
	client := &http.Client{Transport: tr, Timeout: timeout}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader([]byte(bodyJSON)))
	if err != nil {
		return "", err
	}
	for k, vs := range headers {
		for _, v := range vs {
			req.Header[k] = append(req.Header[k], v)
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("HTTP %d | %s", resp.StatusCode, string(body)), nil
}

// TryDecryptResponseBizContent 尝试解析响应并解密打印 bizContent（成功/失败响应形态均兼容）。
func TryDecryptResponseBizContent(responseText string, pfx *PfxContent) {
	idx := strings.Index(responseText, "{")
	if idx < 0 {
		return
	}
	var root struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal([]byte(responseText[idx:]), &root); err != nil {
		fmt.Printf("[响应 bizContent 解密失败] 非 JSON 响应: %v\n", err)
		return
	}
	var dataNode struct {
		Content json.RawMessage `json:"content"`
	}
	if len(root.Data) == 0 || json.Unmarshal(root.Data, &dataNode) != nil || len(dataNode.Content) == 0 {
		return
	}
	var contentObj map[string]interface{}
	if err := json.Unmarshal(dataNode.Content, &contentObj); err != nil {
		// 网关成功响应时 data.content 是 JSON 字符串（转义串），需再 parse 一层
		var s string
		if json.Unmarshal(dataNode.Content, &s) != nil {
			return
		}
		if json.Unmarshal([]byte(s), &contentObj) != nil {
			return
		}
	}
	bizContent, _ := contentObj["bizContent"].(string)
	if bizContent == "" {
		return
	}
	encType, _ := contentObj["encType"].(string)
	if encType == "" {
		encType, _ = contentObj["encryptType"].(string)
	}
	fmt.Println("=================== 响应 encType ===================")
	fmt.Println(encType)
	fmt.Println("=================== 响应 bizContent 明文 ===================")
	if strings.EqualFold(encType, "SM2") {
		plain, err := DecodeBizContent(bizContent, pfx)
		if err != nil {
			fmt.Printf("[响应 bizContent 解密失败] %v\n", err)
			return
		}
		fmt.Println(plain)
	} else {
		if raw, err := Base64Decode(bizContent); err == nil {
			fmt.Println(string(raw))
		} else {
			fmt.Println(bizContent)
		}
	}
}

// RunDemo 执行一次完整网关调用：CMS P7 签名 + 信封加密 bizContent → 组装 content →
// 签名 → POST → 解密响应。输出分节与 Java/Python/Node.js 版一致。
// ⚠️ bizContent 的信封明文是 CMS(P7) 签名值而非业务 JSON 裸文（与三语言版一致）。
func RunDemo(cfg *Config, pfx *PfxContent, bizJSON string) {
	p7, err := Sm2P7SignAttached([]byte(bizJSON), pfx)
	if err != nil {
		fmt.Printf("[错误] bizContent CMS 签名失败: %v\n", err)
		return
	}
	bizContent, err := Sm2EncryptEnvelop(p7, cfg.Sm2JdPub)
	if err != nil {
		fmt.Printf("[错误] bizContent 信封加密失败: %v\n", err)
		return
	}
	content := BuildContent(bizContent, cfg)
	signString := BuildSignString(content)
	sign := HmacSm3Hex(signString, cfg.SecretKey)
	content.Sign = sign

	var bodyBuf bytes.Buffer
	body := contentBody{}
	body.Data.Content = content
	if err := json.NewEncoder(&bodyBuf).Encode(body); err != nil {
		fmt.Printf("[错误] 请求体序列化失败: %v\n", err)
		return
	}
	// json.Encoder 结尾会补换行，去掉
	bodyJSON := strings.TrimRight(bodyBuf.String(), "\n")

	headers := BuildHttpHeaders(cfg.AppID)

	fmt.Println("=================== bizContent 明文 ===================")
	fmt.Println(bizJSON)
	fmt.Println("=================== 签名原文 ===================")
	fmt.Println(signString)
	fmt.Println("=================== 签名结果 ===================")
	fmt.Println(sign)
	fmt.Println("=================== HTTP Header ===================")
	for k, vs := range headers {
		fmt.Printf("%s:%s\n", k, strings.Join(vs, ","))
	}
	fmt.Println("=================== HTTP Body ===================")
	fmt.Println(bodyJSON)

	resp, err := PostJSON(cfg.EndpointURL, headers, bodyJSON, 30*time.Second)
	if err != nil {
		fmt.Printf("[错误] HTTP 调用失败: %v\n", err)
		return
	}
	fmt.Println("=================== HTTP Response ===================")
	fmt.Println(resp)

	TryDecryptResponseBizContent(resp, pfx)
}

// RandomHex 生成 length 位大写十六进制随机串。
func RandomHex(length int) string {
	const chars = "0123456789ABCDEF"
	raw := randomBytes(length)
	out := make([]byte, length)
	for i := range out {
		out[i] = chars[int(raw[i])%16]
	}
	return string(out)
}

// GenReqNo 生成 32 位大写 hex 请求号（等价 UUID hex）。
func GenReqNo() string { return RandomHex(32) }

// EncodeBizContent SM2 数字信封加密 bizContent（明文 JSON 字符串），返回 Base64。
func EncodeBizContent(bizJSON string, jdCertB64 string) (string, error) {
	return Sm2EncryptEnvelop([]byte(bizJSON), jdCertB64)
}

// DecodeBizContent 解密响应 bizContent：先解京东数字信封，再剥 CMS 层取业务明文。
func DecodeBizContent(bizContentB64 string, pfx *PfxContent) (string, error) {
	p7, err := Sm2DecryptEnvelop(bizContentB64, pfx.PrivateKeyD)
	if err != nil {
		return "", err
	}
	plain, err := CmsExtractContent(p7)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}
