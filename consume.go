package unionpay

import (
	"bytes"
	"crypto/rsa"
	"fmt"
	"net/http"
	"text/template"
	"time"
)

var frontConsumeParamMap = map[string]bool{
	"version":         true,  // 版本号 固定填写5.0.0
	"encoding":        true,  // 编码方式 默认值 UTF-8
	"certId":          true,  // 证书id
	"signature":       true,  // 签名 填写对报文摘要的签名
	"signMethod":      true,  // 签名方式 取值：01 表示采用的是RSA
	"txnType":         true,  // 交易类型 取值：01
	"txnSubType":      true,  // 交易子类 01:自主消费，通过地址的方式区分前台消费和后台消费（含无跳转支付） 03:分期付款
	"bizType":         true,  // 产品类型 000201
	"channelType":     true,  // 渠道类型
	"frontUrl":        false, // 前台通知地址 前台返回商户结果时使用，前台类交易需上送
	"backUrl":         true,  // 后台通知地址 后台返回商户结果时使用，如上送，则发送商户后台交易结果通知
	"accessType":      true,  // 接入类型 0:普通商户直接接入 2:平台类商户接入
	"merId":           true,  // 商户代码
	"subMerId":        false, // 二级商户代码 商户类型为平台商户接入时必须上送
	"subMerName":      false, // 二级商户全称 商户类型为平台商户接入时必须上送
	"subMerAbbr":      false, // 二级商户简称 商户类型为平台商户接入时必须上送
	"orderId":         true,  // 商户订单号 商户端生成
	"txnTime":         true,  // 订单发送时间 商户发送交易时间
	"accType":         false, // 账号类型 后台类交易且卡号上送; 跨行收单且收单机构收集银行卡 信息时上送 01: 02: 03:IC  默认取值: 取值“03”表示以 IC 终端发起的 IC 卡交易,IC 作为普通银行卡进行支 付时,此域填写为“01”
	"accNo":           false, // 账号 1、 后台类消费交易时上送全卡号 2、 跨行收单且收单机构收集银行 卡信息时上送、 3、前台类交易可通过配置后返回, 卡号可选上送
	"txnAmt":          true,  // 交易金额 单位为分
	"currencyCode":    true,  // 交易币种 默认为156
	"customerInfo":    false, // 银行卡验证信息及身法信息 1、后台类消费交易时上送 2、认证支付 2.0,后台交易时可选 Key=value 格式
	"orderTimeout":    false, // 账号接受超时时间（防钓鱼使用）1、前台类消费交易时上送 2、认证支付 2.0,后台交易时可选
	"payTimeout":      false, // 订单支付超时时间 超过此时间用户支付成功的交易, 不通知商户,系统自动退款,大约 5 个工作日金额返还到用户账户
	"termId":          false, // 终端号
	"reqReserved":     false, // 请求方保留域 商户自定义保留域，交易应答时会原样返回
	"reserved":        false, // 保留域
	"riskRateInfo":    false, // 风险信息域
	"encryptCertId":   false, // 加密证书
	"frontFailUrl":    false, // 失败交易前台跳转地址 前台消费交易弱商户上送此字段，则在支付失败时，页面跳转至商户该URL（不带交易信息，仅跳转）
	"instalTransInfo": false, // 分期付款信息域 分期付款交易，商户端选择分期信息时，需上送组合域，填法见数据元说明
	"defaultPayType":  false, // 默认支付方式 取值参考数据字典
	"issInsCode":      false, // 发卡机构代码 1、当账号类型为 02-存折时需填写 2、在前台类交易时填写默认银行 代码,支持直接跳转到网银。银行简码列表参考附录：C.1,C.2，其中C.2银行列表仅支持借记卡
	"supPayType":      false, // 支持支付方式 仅仅 pc 使用,使用哪种支付方式 由收单机构填写,取值为以下内容 的一种或多种,通过逗号(,)分 割。取值参考数据字典
	"userMac":         false, // 终端信息域 移动支付业务需要上送
	"customerIp":      false, // 持卡人IP 前台交易，有IP防钓鱼要求的商户上送
	"cardTransData":   false, // 有卡交易信息域 有卡交易必填
	"orderDesc":       false, // 订单描述 移动支付上送
}

func (up *UnionPay) FrontConsume(orderID string, amount int64, returnURL, notifyURL string, extraParams map[string]string) (html string, err error) {
	params := up.initFrontConsumeParams(orderID, amount, returnURL, notifyURL, extraParams)
	kvs, err := GenKVpairs(frontConsumeParamMap, params, "signature")
	if err != nil {
		return
	}

	var sig string
	sig, err = signature(up.privateKey, kvs)
	if err != nil {
		return
	}

	kvs = append(kvs, KVpair{K: "signature", V: sig})
	html = up.checkoutHTML(kvs)

	return
}

func (up *UnionPay) checkoutHTML(kvs KVpairs) string {
	var tpl = `
<html>
<head>
<meta http-equiv="Content-Type" content="text/html; charset=utf-8" /></head>
<body onload="OnLoadSubmit();">
<form id="pay_form" action="{{.Action}}" method="post">
{{range .Data}}
<input type="hidden" name="{{.K}}" id="{{.K}}" value="{{.V}}" />
{{end}}
</form>
<script type="text/javascript">
<!--
function OnLoadSubmit()
{
document.getElementById("pay_form").submit();
}
//-->
</script>
</body>
</html>
`
	buff := bytes.NewBufferString("")
	t, _ := template.New("").Parse(tpl)
	t.Execute(buff, map[string]interface{}{
		"Data":   kvs,
		"Action": up.getHost() + frontTransReq,
	})
	return buff.String()
}

func (up *UnionPay) initFrontConsumeParams(orderID string, amount int64, returnURL, notifyURL string, extraParams map[string]string) (params map[string]string) {
	params = make(map[string]string)

	params["certId"] = up.publicKey.SerialNumber.String() //证书id
	params["merId"] = up.mchID                            //商户代码，请改自己的测试商户号

	params["frontUrl"] = returnURL                          //前台通知地址
	params["backUrl"] = notifyURL                           //后台通知地址
	params["orderId"] = orderID                             //商户订单号
	params["txnTime"] = time.Now().Format("20060102150405") //订单发送时间
	params["txnAmt"] = fmt.Sprintf("%d", amount)            //交易金额，单位分
	params["signMethod"] = "01"                             //签名方法

	params["version"] = "5.0.0"       //版本号
	params["encoding"] = "utf-8"      //编码方式
	params["txnType"] = "01"          //交易类型
	params["txnSubType"] = "01"       //交易子类
	params["bizType"] = "000201"      //业务类型
	params["channelType"] = "08"      //渠道类型，07-PC，08-手机
	params["accessType"] = "0"        //接入类型
	params["currencyCode"] = "156"    //交易币种
	params["defaultPayType"] = "0001" //默认支付方式

	if extraParams != nil {
		for k, v := range extraParams {
			_, ok := frontConsumeParamMap[k]
			if ok {
				params[k] = v
			}
		}
	}
	return
}

type FrontConsumeReturnResponse struct {
	Version            string // 版本号 R
	Encoding           string // 编码方式 R
	CertID             string // 证书id  M
	Signature          string // 签名 M
	SignMethod         string // 签名方式 M
	TxnType            string // 交易类型 R
	TxnSubType         string // 交易子类 R
	BizType            string // 产品类型 R
	AccessType         string // 接入类型 R
	MerID              string // 商户代码 R
	OrderID            string // 商户订单号 R
	TxnTime            string // 订单发送时间 R
	TxnAmt             string // 交易金额 R
	CurrencyCode       string // 交易币种 R
	ReqReserved        string // 请求方保留域 R
	Reserved           string // 保留域 O
	QueryID            string // 交易查询流水号 M 消费交易的流水号，供后续查询用
	RespCode           string // 响应码 M
	RespMsg            string // 响应消息 M
	AccNo              string // 账号 C 根据商户配置返回
	PayCardType        string // 支付卡类型 C 根据商户配置返回
	PayType            string // 支付方式 C 根据商户配置返回
	TN                 string // 银联订单号 C 商户推送订单后银联移动支付系统返回该流水号，商户调用支付控件时使用
	TraceNo            string // 系统跟踪号
	TraceTime          string // 交易传输时间
	SettleDate         string // 清算日期
	SettleCurrencyCode string // 清算货币
	SettleAmt          string // 清算金额
}

func (up *UnionPay) FrontConsumeReturn(req *http.Request) (resp *FrontConsumeReturnResponse, err error) {
	if err = req.ParseForm(); err != nil {
		return
	}
	vals := req.Form

	var fields = []string{
		"version",
		"encoding",
		"certId",
		"signature",
		"signMethod",
		"txnType",
		"txnSubType",
		"bizType",
		"accessType",
		"merId",
		"orderId",
		"txnTime",
		"txnAmt",
		"currencyCode",
		"reqReserved",
		"reserved",
		"queryId",
		"respCode",
		"respMsg",
		"accNo",
		"payCardType",
		"payType",
		"tn",
		"traceNo",
		"traceTime",
		"settleDate",
		"settleCurrencyCode",
		"settleAmt",
	}

	if err = verify(up.verifySignCert.PublicKey.(*rsa.PublicKey), vals, fields); err != nil {
		return
	}

	resp = &FrontConsumeReturnResponse{
		Version:            vals.Get("version"),
		Encoding:           vals.Get("encoding"),
		CertID:             vals.Get("certId"),
		Signature:          vals.Get("signature"),
		SignMethod:         vals.Get("signMethod"),
		TxnType:            vals.Get("txnType"),
		TxnSubType:         vals.Get("txnSubType"),
		BizType:            vals.Get("bizType"),
		AccessType:         vals.Get("accessType"),
		MerID:              vals.Get("merId"),
		OrderID:            vals.Get("orderId"),
		TxnTime:            vals.Get("txnTime"),
		TxnAmt:             vals.Get("txnAmt"),
		CurrencyCode:       vals.Get("currencyCode"),
		ReqReserved:        vals.Get("reqReserved"),
		Reserved:           vals.Get("reserved"),
		QueryID:            vals.Get("queryId"),
		RespCode:           vals.Get("respCode"),
		RespMsg:            vals.Get("respMsg"),
		AccNo:              vals.Get("accNo"),
		PayCardType:        vals.Get("payCardType"),
		PayType:            vals.Get("payType"),
		TN:                 vals.Get("tn"),
		TraceNo:            vals.Get("traceNo"),
		TraceTime:          vals.Get("traceTime"),
		SettleDate:         vals.Get("settleDate"),
		SettleCurrencyCode: vals.Get("settleCurrencyCode"),
		SettleAmt:          vals.Get("settleAmt"),
	}

	if resp.RespCode != "00" {
		err = fmt.Errorf("[unionpay] response code:%s, error:%s", resp.RespCode, resp.RespMsg)
		return
	}

	return
}

type FrontConsumeNotifyResponse struct {
	Version            string // 版本号 R
	Encoding           string // 编码方式 R
	CertID             string // 证书id  M
	Signature          string // 签名 M
	SignMethod         string // 签名方式 M
	TxnType            string // 交易类型 R
	TxnSubType         string // 交易子类 R
	BizType            string // 产品类型 R
	AccessType         string // 接入类型 R
	MerID              string // 商户代码 R
	OrderID            string // 商户订单号 R
	TxnTime            string // 订单发送时间 R
	TxnAmt             string // 交易金额 R
	CurrencyCode       string // 交易币种 R
	ReqReserved        string // 请求方保留域 R
	Reserved           string // 保留域 O
	QueryID            string // 交易查询流水号 M 消费交易的流水号，供后续查询用
	RespCode           string // 响应码 M
	RespMsg            string // 响应消息 M
	SettleAmt          string // 清算金额 M
	SettleCurrencyCode string // 清算币种 M
	SettleDate         string // 清算日期 M
	TraceNo            string // 系统跟踪号 M
	TraceTime          string // 交易传输时间 M
	ExchangeDate       string // 兑换日期 C 境外交易时返回
	ExchangeRate       string // 汇率 C 境外交易时返回
	AccNo              string // 账号 C 根据商户配置返回
	PayCardType        string // 支付卡类型 根据商户配置返回
	PayType            string // 支付方式 C 根据商户配置返回
	PayCardNo          string // 支付卡标示 C 移动支付交易时，根据商户配置返回
	PayCardIssueName   string // 支付卡名称 C 移动支付交易时，根据商户配置返回
	BindID             string // 绑定标示号 R 绑定支付时，根据商户配置返回
}

func (up *UnionPay) FrontConsumeNotify(req *http.Request) (resp *FrontConsumeNotifyResponse, err error) {
	if err = req.ParseForm(); err != nil {
		return
	}
	vals := req.Form

	var fields = []string{
		"version",
		"encoding",
		"certId",
		"signature",
		"signMethod",
		"txnType",
		"txnSubType",
		"bizType",
		"accessType",
		"merId",
		"orderId",
		"txnTime",
		"txnAmt",
		"currencyCode",
		"reqReserved",
		"reserved",
		"queryId",
		"respCode",
		"respMsg",
		"settleAmt",
		"settleCurrencyCode",
		"settleDate",
		"traceNo",
		"traceTime",
		"exchangeDate",
		"exchangeRate",
		"accNo",
		"payCardType",
		"payType",
		"payCardNo",
		"payCardIssueName",
		"bindId",
	}

	if err = verify(up.verifySignCert.PublicKey.(*rsa.PublicKey), vals, fields); err != nil {
		return
	}

	resp = &FrontConsumeNotifyResponse{
		Version:            vals.Get("version"),
		Encoding:           vals.Get("encoding"),
		CertID:             vals.Get("certId"),
		Signature:          vals.Get("signature"),
		SignMethod:         vals.Get("signMethod"),
		TxnType:            vals.Get("txnType"),
		TxnSubType:         vals.Get("txnSubType"),
		BizType:            vals.Get("bizType"),
		AccessType:         vals.Get("accessType"),
		MerID:              vals.Get("merId"),
		OrderID:            vals.Get("orderId"),
		TxnTime:            vals.Get("txnTime"),
		TxnAmt:             vals.Get("txnAmt"),
		CurrencyCode:       vals.Get("currencyCode"),
		ReqReserved:        vals.Get("reqReserved"),
		Reserved:           vals.Get("reserved"),
		QueryID:            vals.Get("queryId"),
		RespCode:           vals.Get("respCode"),
		RespMsg:            vals.Get("respMsg"),
		SettleAmt:          vals.Get("settleAmt"),
		SettleCurrencyCode: vals.Get("settleCurrencyCode"),
		SettleDate:         vals.Get("settleDate"),
		TraceNo:            vals.Get("traceNo"),
		TraceTime:          vals.Get("traceTime"),
		ExchangeDate:       vals.Get("exchangeDate"),
		ExchangeRate:       vals.Get("exchangeRate"),
		AccNo:              vals.Get("accNo"),
		PayCardType:        vals.Get("payCardType"),
		PayType:            vals.Get("payType"),
		PayCardNo:          vals.Get("payCardNo"),
		PayCardIssueName:   vals.Get("payCardIssueName"),
		BindID:             vals.Get("bindId"),
	}

	return
}
