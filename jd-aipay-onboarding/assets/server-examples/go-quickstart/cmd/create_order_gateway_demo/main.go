// AI 付 createOrder 接口 Demo（Go 版）：
// bizContent 使用 SM2 数字信封加密（encType=SM2），外层用 HMAC-SM3 计算 sign，
// 组装 data.content 结构后通过 HTTPS POST 调用网关接口。
package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"aipay-quickstart/aipay"
)

// ============================================================================
// 业务参数（由 render_server_example.sh 替换占位符；未渲染时使用默认值）
// ============================================================================

var (
	// 商户外部订单号
	outTradeNo = "__OUT_TRADE_NO__"
	// 用户 ID
	userID = "__USER_ID__"
	// 交易金额（分）；渲染后占位符为数字字符串，未渲染/非法时回退 99（0.99 元）
	tradeAmount = mustAmount("__TRADE_AMOUNT__")
)

// mustAmount 解析金额占位符（渲染后为数字字符串）。
func mustAmount(s string) int64 {
	if n, err := strconv.ParseInt(s, 10, 64); err == nil && n > 0 {
		return n
	}
	return 99
}

// deviceInfo 设备信息（协议要求：紧凑 JSON 字符串形态）
type deviceInfo struct {
	Vendor        string `json:"vendor"`
	DeviceID      string `json:"deviceId"`
	DeviceAccount string `json:"deviceAccount"`
	DeviceName    string `json:"deviceName"`
	DeviceType    string `json:"deviceType"`
	DeviceSn      string `json:"deviceSn"`
	DeviceModel   string `json:"deviceModel"`
}

// createOrderBiz 业务报文（字段顺序与 Java/Python 版一致）
type createOrderBiz struct {
	TradeSubject  string          `json:"tradeSubject"`
	ClientType    string          `json:"clientType"`
	DeviceInfo    string          `json:"deviceInfo"`
	TradeAmount   int64           `json:"tradeAmount"`
	CreateDate    string          `json:"createDate"`
	AcqMerchantNo string          `json:"acqMerchantNo"`
	AccessType    string          `json:"accessType"`
	OutTradeNo    string          `json:"outTradeNo"`
	TradeType     string          `json:"tradeType"`
	UserIp        string          `json:"userIp"`
	UserID        string          `json:"userId"`
	TradeRemark   string          `json:"tradeRemark"`
	NotifyUrl     string          `json:"notifyUrl"`
	Currency      string          `json:"currency"`
	ExpiryTime    string          `json:"expiryTime"`
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

	deviceBytes, _ := json.Marshal(deviceInfo{
		Vendor:        "TEST",
		DeviceID:      "SN-TEST-01231231",
		DeviceAccount: "test_device_001",
		DeviceName:    "测试设备",
		DeviceType:    "SMART_DEVICE",
		DeviceSn:      "SN-TEST-01231231",
		DeviceModel:   "TestModel",
	})
	biz := createOrderBiz{
		TradeSubject:  "AI付测试订单",
		ClientType:    "APP",
		DeviceInfo:    string(deviceBytes),
		TradeAmount:   tradeAmount,
		CreateDate:    time.Now().Format("20060102150405"),
		AcqMerchantNo: cfg.AcqMerchantNo,
		AccessType:    cfg.AccessType,
		OutTradeNo:    outTradeNo,
		TradeType:     "Aipay",
		UserIp:        "127.0.0.1",
		UserID:        userID,
		TradeRemark:   "AI付接口测试",
		NotifyUrl:     "https://merchant.example.com/notify/pay",
		Currency:      "CNY",
		ExpiryTime:    "604800",
	}
	bizJSON, err := json.Marshal(biz)
	if err != nil {
		fmt.Println("[错误] bizContent 序列化失败:", err)
		return
	}
	aipay.RunDemo(cfg, pfx, string(bizJSON))
}
