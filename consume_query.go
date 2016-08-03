package unionpay

import "net/url"

type ConsumeQueryResponse struct {
	Version            string
	Encoding           string
	CertID             string
	Signature          string
	SignMethod         string
	TxnType            string
	TxnSubType         string
	AccessType         string
	MerID              string
	OrderID            string
	TxnTime            string
	PayType            string
	CurrencyCode       string
	AccNo              string
	PayCardType        string
	TxnAmt             string // 长度为1到12字节的变长整型数值，以分为单位
	ReqReserved        string
	Reserved           string
	IssuerIdentifyMode string
	QueryID            string
	TraceNo            string
	TraceTime          string
	SettleDate         string
	SettleCurrencyCode string
	SettleAmt          string
	OrigRespCode       string
	OrigRespMsg        string
	RespCode           string
	RespMsg            string
}

func (up *UnionPay) ConsumeQuery(orderID, queryID, txnTime, reserved string) (resp *ConsumeQueryResponse, err error) {
	kvs := KVpairs{}

	kvs = append(kvs, KVpair{K: "version", V: "5.0.0"})
	kvs = append(kvs, KVpair{K: "encoding", V: "UTF-8"})
	kvs = append(kvs, KVpair{K: "certId", V: up.publicKey.SerialNumber.String()})
	kvs = append(kvs, KVpair{K: "signMethod", V: "01"})
	kvs = append(kvs, KVpair{K: "txnType", V: "00"})
	kvs = append(kvs, KVpair{K: "txnSubType", V: "00"})
	kvs = append(kvs, KVpair{K: "bizType", V: "000000"})
	kvs = append(kvs, KVpair{K: "accessType", V: "0"})
	kvs = append(kvs, KVpair{K: "channelType", V: "07"})
	kvs = append(kvs, KVpair{K: "merId", V: up.mchID})
	kvs = append(kvs, KVpair{K: "orderId", V: orderID})
	kvs = append(kvs, KVpair{K: "txnTime", V: txnTime})
	kvs = append(kvs, KVpair{K: "reserved", V: reserved})
	kvs = append(kvs, KVpair{K: "queryId", V: queryID})

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
	u, err = url.Parse(up.getHost() + queryTrans)
	if err != nil {
		return
	}

	var result ConsumeQueryResponse
	err = up.client.PostForm(u, data, &result)
	if err != nil {
		return
	}

	resp = &result
	return
}
