// AI 付接入配置：从工程根目录的 aipay.env 读取全部可替换配置项
// （与 Java/Python/Node.js 版同一份配置文件、同一批键名）。
package aipay

import (
	"fmt"
	"os"
	"path/filepath"
)

// Config aipay.env 解析结果。
type Config struct {
	Env           string // pre | prod | sandbox
	EndpointURL   string
	AppID         string
	AgentID       string
	MerchantNo    string
	AcqMerchantNo string
	AccessType    string
	SecretKey     string
	PFXPassword   string
	PFXBase64     string
	Sm2JdPub      string
}

// LoadConfig 从当前工作目录逐级向上查找 aipay.env 并解析；文件缺失或缺项返回明确错误。
func LoadConfig() (*Config, error) {
	path, err := findEnvFile()
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	values := map[string]string{}
	for _, line := range stringsSplitLines(string(raw)) {
		line = trimSpace(line)
		if line == "" || line[0] == '#' {
			continue
		}
		idx := -1
		for i := 0; i < len(line); i++ {
			if line[i] == '=' {
				idx = i
				break
			}
		}
		if idx < 0 {
			continue
		}
		values[trimSpace(line[:idx])] = trimSpace(line[idx+1:])
	}
	get := func(key string) (string, error) {
		v, ok := values[key]
		if !ok || v == "" {
			return "", fmt.Errorf("aipay.env 缺少配置项: %s，请补齐后重试", key)
		}
		return v, nil
	}
	cfg := &Config{}
	var err2 error
	if cfg.Env, err2 = get("AIPAY_ENV"); err2 != nil {
		return nil, err2
	}
	if cfg.EndpointURL, err2 = get("AIPAY_ENDPOINT_URL"); err2 != nil {
		return nil, err2
	}
	if cfg.AppID, err2 = get("AIPAY_APP_ID"); err2 != nil {
		return nil, err2
	}
	if cfg.AgentID, err2 = get("AIPAY_AGENT_ID"); err2 != nil {
		return nil, err2
	}
	if cfg.MerchantNo, err2 = get("AIPAY_MERCHANT_NO"); err2 != nil {
		return nil, err2
	}
	if cfg.AcqMerchantNo, err2 = get("AIPAY_ACQ_MERCHANT_NO"); err2 != nil {
		return nil, err2
	}
	if cfg.AccessType, err2 = get("AIPAY_ACCESS_TYPE"); err2 != nil {
		return nil, err2
	}
	if cfg.SecretKey, err2 = get("AIPAY_SECRET_KEY"); err2 != nil {
		return nil, err2
	}
	if cfg.PFXPassword, err2 = get("AIPAY_MERCHANT_PFX_PASSWORD"); err2 != nil {
		return nil, err2
	}
	if cfg.PFXBase64, err2 = get("AIPAY_MERCHANT_PFX"); err2 != nil {
		return nil, err2
	}
	if cfg.Sm2JdPub, err2 = get("AIPAY_PUBLIC_KEY"); err2 != nil {
		return nil, err2
	}
	return cfg, nil
}

// LoadPfx 加载并解析商户 PFX。
func (c *Config) LoadPfx() (*PfxContent, error) {
	return ParsePfxFromBase64(c.PFXBase64, c.PFXPassword)
}

func findEnvFile() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := wd
	for {
		candidate := filepath.Join(dir, "aipay.env")
		if st, err := os.Stat(candidate); err == nil && !st.IsDir() {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("未找到配置文件 aipay.env（应在工程根目录）。该文件承载全部可替换配置项，请勿删除或改名")
}

func stringsSplitLines(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			line := s[start:i]
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			out = append(out, line)
			start = i + 1
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}

func trimSpace(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t' || s[0] == '\r' || s[0] == '\n') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t' || s[len(s)-1] == '\r' || s[len(s)-1] == '\n') {
		s = s[:len(s)-1]
	}
	return s
}
