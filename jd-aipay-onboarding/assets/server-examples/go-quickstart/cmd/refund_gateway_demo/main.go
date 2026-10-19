// AI 付 refund 接口 Demo（Go 版）。
package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"aipay-quickstart/aipay"
)

// 业务参数（由 render_server_example.sh 替换占位符；金额未渲染时回退 1 分）
var (
	// 原下单商户订单号
	originalOutTradeNo = "__ORIGINAL_OUT_TRADE_NO__"
	// 退款金额（分）
	refundAmount = mustAmount("__REFUND_AMOUNT__")
)

// mustAmount 解析金额占位符（渲染后为数字字面量）。
func mustAmount(s string) int64 {
	if n, err := strconv.ParseInt(s, 10, 64); err == nil && n > 0 {
		return n
	}
	return 1
}

// genRefundNo REFUND + yyyyMMddHHmmss + 3 位随机数字，保证幂等。
func genRefundNo() string {
	return fmt.Sprintf("REFUND%s%03d", time.Now().Format("20060102150405"), rand.Intn(1000))
}

// refundBiz 业务报文
type refundBiz struct {
	AcqMerchantNo      string `json:"acqMerchantNo"`
	AccessType         string `json:"accessType"`
	OriginalOutTradeNo string `json:"originalOutTradeNo"`
	RefundNo           string `json:"refundNo"`
	RefundAmount       int64  `json:"refundAmount"`
	Currency           string `json:"currency"`
	RefundReason       string `json:"refundReason"`
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

	bizJSON, err := json.Marshal(refundBiz{
		AcqMerchantNo:      cfg.AcqMerchantNo,
		AccessType:         cfg.AccessType,
		OriginalOutTradeNo: originalOutTradeNo,
		RefundNo:           genRefundNo(),
		RefundAmount:       refundAmount,
		Currency:           "CNY",
		RefundReason:       "AI付退款测试",
	})
	if err != nil {
		fmt.Println("[错误] bizContent 序列化失败:", err)
		return
	}
	aipay.RunDemo(cfg, pfx, string(bizJSON))
}
