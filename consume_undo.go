package unionpay

import (
	"crypto/rsa"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type ConsumeUndoResponse struct {
	Version     string
	Encoding    string
	CertID      string
	Signature   string
	SignMethod  string
	TxnType     string
	TxnSubType  string
	BizType     string
	AccessType  string
	MerID       string
	OrderID     string
	TxnTime     string
	TxnAmt      string // 长度为1到12字节的变长整型数值，以分为单位
	ReqReserved string
	Reserved    string
	QueryID     string
	OrigQryID   string
	RespCode    string
	RespMsg     string
}

func (up *UnionPay) ConsumeUndo(orderID, returnURL string, amount int64, originQueryID, reqReserved, reserved string) (resp *ConsumeUndoResponse, err error) {
	kvs := KVpairs{}

	kvs = append(kvs, KVpair{K: "version", V: "5.0.0"})
	kvs = append(kvs, KVpair{K: "encoding", V: "UTF-8"})
	kvs = append(kvs, KVpair{K: "certId", V: up.publicKey.SerialNumber.String()})
	kvs = append(kvs, KVpair{K: "signMethod", V: "01"})
	kvs = append(kvs, KVpair{K: "txnType", V: "31"})
	kvs = append(kvs, KVpair{K: "txnSubType", V: "00"})
	kvs = append(kvs, KVpair{K: "bizType", V: "000201"})
	kvs = append(kvs, KVpair{K: "backUrl", V: returnURL})
	kvs = append(kvs, KVpair{K: "accessType", V: "0"})
	kvs = append(kvs, KVpair{K: "merId", V: up.mchID})
	kvs = append(kvs, KVpair{K: "orderId", V: orderID})
	kvs = append(kvs, KVpair{K: "txnTime", V: time.Now().Format("20060102150405")})
	kvs = append(kvs, KVpair{K: "txnAmt", V: fmt.Sprint(amount)})
	kvs = append(kvs, KVpair{K: "reqReserved", V: reqReserved})
	kvs = append(kvs, KVpair{K: "reserved", V: reserved})
	kvs = append(kvs, KVpair{K: "origQryId", V: originQueryID})
	kvs = append(kvs, KVpair{K: "channelType", V: "07"})

	var sig string
	sig, err = signature(up.privateKey, kvs)
	if err != nil {
		return
	}

	kvs = append(kvs, KVpair{K: "signature", V: sig})

	data := url.Values{}
	for _, v := range kvs {
		data.Set(v.K, v.V)
	}

	var u *url.URL
	u, err = url.Parse(up.getHost() + backTransReq)
	if err != nil {
		return
	}

	var result ConsumeUndoResponse
	err = up.client.PostForm(u, data, &result)
	if err != nil {
		return
	}
	resp = &result
	return
}

type ConsumeUndoNotifyResponse struct {
	Version            string
	Encoding           string
	CertID             string
	Signature          string
	SignMethod         string
	TxnType            string
	TxnSubType         string
	BizType            string
	AccessType         string
	MerID              string
	OrderID            string
	TxnTime            string
	CurrencyCode       string
	TxnAmt             string // 长度为1到12字节的变长整型数值，以分为单位
	ReqReserved        string
	Reserved           string
	QueryID            string
	OrigQryID          string
	TraceNo            string
	TraceTime          string
	SettleDate         string
	SettleCurrencyCode string
	SettleAmt          string
	RespCode           string
	RespMsg            string
}

func (up *UnionPay) ConsumeUndoNotify(req *http.Request) (resp *ConsumeUndoNotifyResponse, err error) {
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
		"currencyCode",
		"txnAmt",
		"reqReserved",
		"reserved",
		"queryId",
		"origQryId",
		"traceNo",
		"traceTime",
		"settleDate",
		"settleCurrencyCode",
		"settleAmt",
		"respCode",
		"respMsg",
	}
	if err = verify(up.verifySignCert.PublicKey.(*rsa.PublicKey), vals, fields); err != nil {
		return
	}

	resp = &ConsumeUndoNotifyResponse{
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
		CurrencyCode:       vals.Get("currencyCode"),
		TxnAmt:             vals.Get("txnAmt"),
		ReqReserved:        vals.Get("reqReserved"),
		Reserved:           vals.Get("reserved"),
		QueryID:            vals.Get("queryId"),
		OrigQryID:          vals.Get("origQryId"),
		TraceNo:            vals.Get("traceNo"),
		TraceTime:          vals.Get("traceTime"),
		SettleDate:         vals.Get("settleDate"),
		SettleCurrencyCode: vals.Get("settleCurrencyCode"),
		SettleAmt:          vals.Get("settleAmt"),
		RespCode:           vals.Get("respCode"),
		RespMsg:            vals.Get("respMsg"),
	}

	return
}
