package unionpay

import (
	"crypto/rsa"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

var mobilePaymentParamMap = map[string]bool{
	"version":    true, //  	版本号	version	NS5	M	固定填写5.0.0	固定填写5.0.0
	"encoding":   true, //  	编码方式	encoding	ANS1..20	M	填写报文使用的字符编码，支持UTF-8与GBK编码	支持UTF-8、GBK
	"certId":     true, //  	证书ID	certId	N1..128	M	填写签名私钥证书的Serial Number，该值可通过SDK获取	SDK代码从证书中读取
	"signMethod": true, //  	签名方法	signMethod	N1..12	M	01：表示采用RSA	固定填写01
	"signature":  true, //  	签名	signature	ANS1..1024	M	填写对报文摘要的签名，可通过SDK生成签名	SDK代码调用签名函数时自动计算
	"txnType":    true, //  	交易类型	txnType	N2	M	取值：
	//  0：查询交易，01：消费，02：预授权，03：预授权完成，04：退货，05：圈存，11：代收，12：代付，13：账单支付，14：转账（保留），21：批量交易，22：批量查询，31：消费撤销，32：预授权撤销，33：预授权完成撤销，71：余额查询，72：实名认证-建立绑定关系，73：账单查询，74：解除绑定关系，75：查询绑定关系，77：发送短信验证码交易，78：开通查询交易，79：开通交易，94：IC卡脚本通知	固定填写01
	"txnSubType":  true, //  	交易子类	txnSubType	N2	M	依据实际交易类型填写。	固定填写01
	"bizType":     true, //  	产品类型	bizType	N6	M	取值：000101：基金业务之股票基金；000102：基金业务之货币基金；000201：B2C网关支付；000301：无跳转（商户侧）；000302：评级支付；000401：代付；000501：代收；000601：账单支付；000801：无跳转（机构侧）；000901：绑定支付；000902: Token支付；001001：订购；000202：B2B  以上产品外其他接口默认送000000，对账文件下载接口必送000000	固定填写000201
	"channelType": true, //  	渠道类型	channelType	N2	M	05：语音 07：互联网 08：移动	固定填写08  户信息
	"accessType":  true, //  	接入类型	accessType	N1	M	0：商户直连接入1：收单机构接入	固定填写0
	"merId":       true, //  	商户代码	merId	AN15	M	已被批准加入银联互联网系统的商户代码	如已签约，请使用真实商户号。如未签约，可以在本平台右上角点我的测试-测试参数获取测试环境商户号
	"backUrl":     true, //  	后台通知地址	backUrl	ANS1..256	M	用于接收后台通知报文，必须上送完整的互联网可访问地址，支持HTTP与HTTPS协议（如：https://xxx.xxx.xxx....），地址中不能包含~	支持http和单向https，必须外网可以访问，例：https://xxx.xxx.c
	//  单信息
	"orderId":      true,  //  	商户订单号	orderId	AN8..32	M	商户订单号，仅能用大小写字母与数字，不能用特殊字符	商户端生成，例：12345asdf
	"currencyCode": true,  //  	交易币种	currencyCode	AN3	M	币种格式必须为3位代码，境内客户取值：156（人民币）	固定156
	"txnAmt":       true,  //  	交易金额	txnAmt	N1..12	M	单位为分，不能带小数点，样例：1元送100	整数，单位为分，例：1元填写100
	"txnTime":      true,  //  	订单发送时间	txnTime	YYYYMMDDHHmmss	M	必须使用当前北京时间（年年年年月月日日时时分分秒秒）24小时制，样例：20151123152540，北京时间	取当前时间，例：20151118100505
	"payTimeout":   false, //  	支付超时时间	payTimeout	YYYYMMDDHHmmss	O	超过此时间客户查询结果为非成功的交易，持卡人可能被扣账，系统会自动退款，大约5个工作日金额返还到持卡人账户（一般在网银支付情况下会出现超时后持卡人可能被扣账情况），此时间建议取支付时的北京时间加15分钟	订单支付超时时间，例：20151118101505
	"accNo":        false, //  	账号	accNo	AN1..512	C	银行卡号。请求时使用加密公钥对交易账号加密，并做Base64编码后上送；应答时如需返回，则使用签名私钥进行解密。前台交易可由银联页面采集，也可由商户上送并返显，如需锁定返显卡号，应通过保留域（reserved）上送卡号锁定标识。	业务运营中心开启了锁卡权限的情况下，送此字段可以指定用户在控件中输入的卡号。
	"reqReserved":  false, //  	请求方自定义域	reqReserved	ANS1..1024	O	商户自定义保留域，交易应答时会原样返回	商户自定义保留域，交易应答时会原样返回
	"orderDesc":    false, //  	订单描述	orderDesc	ANS1..32	C	描述订单信息，显示在银联支付控件或客户端支付界面中	上送时可在控件内显示该信息，但仅用于控件显示，不会在商户和用户的对账单中出现。
}

type MobilePaymentResponse struct {
	Version     string
	Encoding    string
	CertID      string
	SignMethod  string
	Signature   string
	TxnType     string
	TxnSubType  string
	BizType     string
	AccessType  string
	MerID       string
	OrderID     string
	TxnTime     string
	ReqReserved string
	Reserved    string
	RespCode    string
	RespMsg     string
	TN          string
}

func (up *UnionPay) MobilePayment(orderID string, amount int64, notifyURL string, extraParams map[string]string) (resp *MobilePaymentResponse, err error) {
	params := up.initMobilePaymentParams(orderID, amount, notifyURL, extraParams)
	kvs, err := GenKVpairs(mobilePaymentParamMap, params, "signature")
	if err != nil {
		return
	}

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
	u, err = url.Parse(up.getHost() + appTransReq)
	if err != nil {
		return
	}

	var result MobilePaymentResponse
	err = up.client.PostForm(u, data, &result)
	if err != nil {
		return
	}

	resp = &result

	return
}

func (up *UnionPay) initMobilePaymentParams(orderID string, amount int64, notifyURL string, extraParams map[string]string) (params map[string]string) {
	params = make(map[string]string)

	params["version"] = "5.0.0"                             //版本号
	params["encoding"] = "utf-8"                            //编码方式
	params["certId"] = up.publicKey.SerialNumber.String()   //证书id
	params["signMethod"] = "01"                             //签名方法
	params["txnType"] = "01"                                //交易类型
	params["txnSubType"] = "01"                             //交易子类
	params["bizType"] = "000201"                            //业务类型
	params["channelType"] = "08"                            //渠道类型，07-PC，08-手机
	params["accessType"] = "0"                              //接入类型
	params["merId"] = up.mchID                              //商户代码，请改自己的测试商户号
	params["backUrl"] = notifyURL                           //后台通知地址
	params["orderId"] = orderID                             //商户订单号
	params["currencyCode"] = "156"                          //交易币种
	params["txnTime"] = time.Now().Format("20060102150405") //订单发送时间
	params["txnAmt"] = fmt.Sprintf("%d", amount)            //交易金额，单位分

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

type MobilePaymentNotifyResponse struct {
	Version    string //  1	版本号	version	NS5	R	固定填写5.0.0
	Encoding   string //  2	编码方式	encoding	ANS1..20	R	填写报文使用的字符编码，支持UTF-8与GBK编码
	CertID     string //  3	证书ID	certId	N1..128	M	填写签名私钥证书的Serial Number，该值可通过SDK获取
	SignMethod string //  4	签名方法	signMethod	N1..12	M	01：表示采用RSA
	Signature  string //  5	签名	signature	ANS1..1024	M	填写对报文摘要的签名，可通过SDK生成签名
	TxnType    string //  6	交易类型	txnType	N2	R	取值：  00：查询交易，01：消费，02：预授权，03：预授权完成，04：退货，05：圈存，11：代收，12：代付，13：账单支付，14：转账（保留），21：批量交易，22：批量查询，31：消费撤销，32：预授权撤销，33：预授权完成撤销，71：余额查询，72：实名认证-建立绑定关系，73：账单查询，74：解除绑定关系，75：查询绑定关系，77：发送短信验证码交易，78：开通查询交易，79：开通交易，94：IC卡脚本通知
	TxnSubType string //  7	交易子类	txnSubType	N2	R	依据实际交易类型填写。
	BizType    string //  8	产品类型	bizType	N6	R	取值：000101：基金业务之股票基金；000102：基金业务之货币基金；000201：B2C网关支付；000301：无跳转（商户侧）；000302：评级支付；000401：代付；000501：代收；000601：账单支付；000801：无跳转（机构侧）；000901：绑定支付；000902: Token支付；001001：订购；000202：B2B
	//  除以上产品外其他接口默认送000000，对账文件下载接口必送000000
	//  商户信息
	AccessType string //  1	接入类型	accessType	N1	R	0：商户直连接入1：收单机构接入
	MerID      string //  2	商户代码	merId	AN15	R	已被批准加入银联互联网系统的商户代码
	//  订单信息
	OrderID      string //  1	商户订单号	orderId	AN8..32	R	商户订单号，仅能用大小写字母与数字，不能用特殊字符
	CurrencyCode string //  2	交易币种	currencyCode	AN3	M	币种格式必须为3位代码，境内客户取值：156（人民币）	默认为156
	TxnAmt       string //  3	交易金额	txnAmt	N1..12	R	单位为分，不能带小数点，样例：1元送100
	TxnTime      string //  4	订单发送时间	txnTime	YYYYMMDDHHmmss	R	必须使用当前北京时间（年年年年月月日日时时分分秒秒）24小时制，样例：20151123152540，北京时间
	PayType      string //  5	支付方式	payType	N4	C	默认不返回此域，如需要返此域，需要提交申请，视商户配置返回，可在消费类交易中返回以下中的一种： 0001：认证支付 0002：快捷支付 0004：储值卡支付 0005：IC卡支付 0201：网银支付 1001：牡丹畅通卡支付 1002：中铁银通卡支付 0401：信用卡支付——暂定 0402：小额临时支付 0403：认证支付2.0 0404：互联网订单手机支付 9000：其他无卡支付(如手机客户端支付)	根据商户配置返回
	AccNo        string //  6	账号	accNo	AN1..512	C	银行卡号。请求时使用加密公钥对交易账号加密，并做Base64编码后上送；应答时如需返回，则使用签名私钥进行解密。前台交易可由银联页面采集，也可由商户上送并返显，如需锁定返显卡号，应通过保留域（reserved）上送卡号锁定标识。	根据商户配置返回
	PayCardType  string //  7	支付卡类型	payCardType	N2	C	消费交易，视商户配置返回。该域取值为： 00：未知 01：借记账户 02：贷记账户 03：准贷记账户 04：借贷合一账户 05：预付费账户 06：半开放预付费账户	根据商户配置返回
	ReqReserved  string //  8	请求方自定义域	reqReserved	ANS1..1024	R	商户自定义保留域，交易应答时会原样返回
	Reserved     string //  9	保留域	reserved	ANS1..2048	O	保留域包含多个子域，所有子域需用“{}”包含，子域间以“&”符号链接。
	//  格式如下：{子域名1=值&子域名2=值&子域名3=值}。
	//  通知信息
	QueryID            string //  1	查询流水号	queryId	AN21	M	由银联返回，用于在后续类交易中唯一标识一笔交易	消费交易的流水号，供后续查询用
	TraceNO            string //  2	系统跟踪号	traceNo	N6	M	收单机构对账时使用，该域由银联系统产生
	TraceTime          string //  3	交易传输时间	traceTime	MMDDHHmmss	M	（月月日日时时分分秒秒）24小时制收单机构对账时使用，该域由银联系统产生
	SettleDate         string //  4	清算日期	settleDate	MMDD	M	为银联和入网机构间的交易结算日期。一般前一日23点至当天23点为一个清算日。也就是23点前的交易，当天23点之后开始结算，23点之后的交易，要第二天23点之后才会结算。测试环境为测试需要，13:30左右日切，所以13:30到13:30为一个清算日，测试环境今天下午为今天的日期，今天上午为昨天的日期。
	SettleCurrencyCode string //  5	清算币种	settleCurrencyCode	AN3	M	境内返回156
	SettleAmt          string //  6	清算金额	settleAmt	N1..12	M	取值同交易金额
	RespCode           string //  7	应答码	respCode	AN2	M	具体参见应答码定义章节
	RespMsg            string //  8	应答信息	respMsg	ANS1..256	M	填写具体的应答信息
	PayCardNo          string //  9	支付卡标识	payCardNo	ANS1..19	C	移动支付交易时，根据客户配置返回	业务运营中心开启此字段权时，此字段会返回打码卡号。
	PayCardIssueName   string //  10	支付卡名称	payCardIssueName	ANS1..64	C	移动支付交易时，根据客户配置返回	业务运营中心开启此字段权时，此字段会返回支付卡中文名称。
}

func (up *UnionPay) MobilePaymentNotify(req *http.Request) (resp *MobilePaymentNotifyResponse, err error) {
	if err = req.ParseForm(); err != nil {
		return
	}
	vals := req.Form

	if len(vals) == 0 {
		err = ErrNotifyDataIsEmpty
		return
	}

	var fields = []string{
		"version",
		"encoding",
		"certId",
		"signMethod",
		"signature",
		"txnType",
		"txnSubType",
		"bizType",
		"accessType",
		"merId",
		"orderId",
		"currencyCode",
		"txnAmt",
		"txnTime",
		"payType",
		"accNo",
		"payCardType",
		"reqReserved",
		"reserved",
		"queryId",
		"traceNo",
		"traceTime",
		"settleDate",
		"settleCurrencyCode",
		"settleAmt",
		"respCode",
		"respMsg",
		"payCardNo",
		"payCardIssueName",
		"tn",
	}

	if err = verify(up.verifySignCert.PublicKey.(*rsa.PublicKey), vals, fields); err != nil {
		return
	}

	resp = &MobilePaymentNotifyResponse{
		Version:            vals.Get("version"),
		Encoding:           vals.Get("encoding"),
		CertID:             vals.Get("certId"),
		SignMethod:         vals.Get("signMethod"),
		Signature:          vals.Get("signature"),
		TxnType:            vals.Get("txnType"),
		TxnSubType:         vals.Get("txnSubType"),
		BizType:            vals.Get("bizType"),
		AccessType:         vals.Get("accessType"),
		MerID:              vals.Get("merId"),
		OrderID:            vals.Get("orderId"),
		CurrencyCode:       vals.Get("currencyCode"),
		TxnAmt:             vals.Get("txnAmt"),
		TxnTime:            vals.Get("txnTime"),
		PayType:            vals.Get("payType"),
		AccNo:              vals.Get("accNo"),
		PayCardType:        vals.Get("payCardType"),
		ReqReserved:        vals.Get("reqReserved"),
		Reserved:           vals.Get("reserved"),
		QueryID:            vals.Get("queryId"),
		TraceNO:            vals.Get("traceNo"),
		TraceTime:          vals.Get("traceTime"),
		SettleDate:         vals.Get("settleDate"),
		SettleCurrencyCode: vals.Get("settleCurrencyCode"),
		SettleAmt:          vals.Get("settleAmt"),
		RespCode:           vals.Get("respCode"),
		RespMsg:            vals.Get("respMsg"),
		PayCardNo:          vals.Get("payCardNo"),
		PayCardIssueName:   vals.Get("payCardIssueName"),
	}

	return
}
