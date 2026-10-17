// AI 付 queryPayResult 接口 Demo（Go 版）。
package main

import (
	"encoding/json"
	"fmt"

	"aipay-quickstart/aipay"
)

// 业务参数（由 render_server_example.sh 替换占位符）
var (
	// 商户外部订单号
	outTradeNo = "__OUT_TRADE_NO__"
)

// queryPayResultBiz 业务报文
type queryPayResultBiz struct {
	AcqMerchantNo string `json:"acqMerchantNo"`
	AccessType    string `json:"accessType"`
	OutTradeNo    string `json:"outTradeNo"`
}

func main() {
	cfg, err := aipay.LoadConfig()
	if err != nil {
		fmt.Println("[错误]", err)
		return
	}
	pfx, err := cfg.LoadPfx()
	if err != nil {
		fmt.Println("[错误] 商户 PFX 解析失败:", err)
		return
	}

	bizJSON, err := json.Marshal(queryPayResultBiz{
		AcqMerchantNo: cfg.AcqMerchantNo,
		AccessType:    cfg.AccessType,
		OutTradeNo:    outTradeNo,
	})
	if err != nil {
		fmt.Println("[错误] bizContent 序列化失败:", err)
		return
	}
	aipay.RunDemo(cfg, pfx, string(bizJSON))
}
