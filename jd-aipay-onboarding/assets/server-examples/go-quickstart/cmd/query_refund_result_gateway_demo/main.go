// AI 付 queryRefundResult 接口 Demo（Go 版）。
package main

import (
	"encoding/json"
	"fmt"

	"aipay-quickstart/aipay"
)

// 业务参数（由 render_server_example.sh 替换占位符）
var (
	// 退款单号
	refundNo = "__REFUND_NO__"
)

// queryRefundResultBiz 业务报文
type queryRefundResultBiz struct {
	AcqMerchantNo string `json:"acqMerchantNo"`
	AccessType    string `json:"accessType"`
	RefundNo      string `json:"refundNo"`
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

	bizJSON, err := json.Marshal(queryRefundResultBiz{
		AcqMerchantNo: cfg.AcqMerchantNo,
		AccessType:    cfg.AccessType,
		RefundNo:      refundNo,
	})
	if err != nil {
		fmt.Println("[错误] bizContent 序列化失败:", err)
		return
	}
	aipay.RunDemo(cfg, pfx, string(bizJSON))
}
