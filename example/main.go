package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"strconv"
	"time"

	"io"

	"github.com/shima-park/unionpay"
)

var (
	pub   = "key.cert" //加密密钥路径(openssl pkcs12 -in PM_700000000000001_acp.pfx -clcerts -nokeys -out key.cert)
	pri   = "key.pem"  //加密证书路径(openssl pkcs12 -in PM_700000000000001_acp.pfx -nocerts -nodes -out key.pem)
	cert  = "acp_test_verify_sign_new.cer"
	mchID = "700000000000001"
	/*
		测试商户号 700000000000001
		卡号	卡性质	机构名称	手机号码	密码	CVN2	有效期	证件号	姓名
		6216261000000000018	借记卡	平安银行	13552535506	123456			341126197709218366	全渠道
		6221558812340000	贷记卡	平安银行	13552535506	123456	123	1711	341126197709218366	互联网
		短信验证码	111111
	*/

	// 默认调用银联正式环境的地址,访问银联测试环境调用 SetTestEnv(true)
	up = unionpay.NewPayment(mchID, pub, pri, cert).SetTestEnv(true)

	// 示例监听的端口
	port = ":9090"

	// 通过 lt --port 9090 获取的外网地址
	localTunnel = "http://eqfssupbgz.localtunnel.me"

	returnURL       = fmt.Sprintf("%s/%s", localTunnel, "alipay/return")
	notifyURL       = fmt.Sprintf("%s/%s", localTunnel, "alipay/notify")
	returnNotifyURL = fmt.Sprintf("%s/%s", localTunnel, "alipay/return-notify")
)

type MyServeMux struct {
	*http.ServeMux
}

func NewServeMux() *MyServeMux { return &MyServeMux{http.NewServeMux()} }

func (mux *MyServeMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	dump, _ := httputil.DumpRequest(r, true)
	log.Println(string(dump))
	if r.RequestURI == "*" {
		if r.ProtoAtLeast(1, 1) {
			w.Header().Set("Connection", "close")
		}
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	h, _ := mux.Handler(r)
	h.ServeHTTP(w, r)
}

func main() {
	mux := NewServeMux()
	mux.HandleFunc("/index", IndexServer)
	mux.HandleFunc("/unionpay/return", ReturnWebServer)
	mux.HandleFunc("/unionpay/payment-web", PaymentWebServer)
	mux.HandleFunc("/unionpay/payment-mobile", PaymentAPPServer)
	mux.HandleFunc("/unionpay/refund", RefundServer)
	mux.HandleFunc("/unionpay/undo", UndoServer)
	mux.HandleFunc("/unionpay/query", QueryServer)
	mux.HandleFunc("/unionpay/notify-web", NotifyWebServer)
	mux.HandleFunc("/unionpay/notify-mobile", NotifyAppServer)
	mux.HandleFunc("/unionpay/notify-refund", NotifyRefundServer)
	mux.HandleFunc("/unionpay/notify-undo", NotifyUndoServer)

	log.Println("Listen", port)
	log.Fatal(http.ListenAndServe(port, mux))
}

func IndexServer(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("content-type", "text/html; charset=utf-8")
	var html = `
<h1>Oh!!! It works!</h1>
<h2><a href="/unionpay/payment-web">银联网页支付</a></h2>
<h2><a href="/unionpay/payment-mobile">银联移动支付</a></h2>
`
	fmt.Fprintf(w, html)
	return
}

func PaymentWebServer(w http.ResponseWriter, req *http.Request) {
	var (
		orderID           = time.Now().Format("20060102150405999")
		amount      int64 = 1
		extraParams       = map[string]string{
			"orderDesc": "test body",
		}
	)
	checkourHTML, err := up.FrontConsume(orderID, amount, returnURL, notifyURL, extraParams)
	if err != nil {
		fmt.Fprintf(w, "Error:%s", err.Error())
		return
	}

	w.Header().Set("content-type", "text/html; charset=utf-8")
	fmt.Fprintf(w, checkourHTML)
	return
}

func PaymentAPPServer(w http.ResponseWriter, req *http.Request) {
	var (
		orderID           = time.Now().Format("20060102150405999")
		amount      int64 = 1
		extraParams       = map[string]string{
			"orderDesc": "test body",
		}
	)

	paymentParams, err := up.MobilePayment(orderID, amount, notifyURL, extraParams)
	if err != nil {
		fmt.Fprintf(w, "Error:%s", err.Error())
		return
	}
	fmt.Fprintf(w, "%s", paymentParams)
	return
}

func ReturnWebServer(w http.ResponseWriter, req *http.Request) {
	r, err := up.FrontConsumeReturn(req)
	if err != nil {
		fmt.Fprintf(w, "Error:%s", err.Error())
		return
	}

	w.Header().Set("content-type", "text/html; charset=utf-8")

	var html = fmt.Sprintf(`
Result:%+v<br>
<a href="/unionpay/consume-query?order_id=%s&query_id=%s&txn_time=%s">查询订单</a><br>
<a href="/unionpay/consume-undo?query_id=%s&amount=%s">订单取消</a><br>
<a href="/unionpay/consume-refund?query_id=%s&amount=%s">订单退款</a><br>
`, r,
		r.OrderID, r.QueryID, r.TxnTime,
		r.QueryID, r.TxnAmt,
		r.QueryID, r.TxnAmt)
	fmt.Fprintf(w, html)
	return
}

func QueryServer(w http.ResponseWriter, req *http.Request) {
	var (
		orderID  = req.URL.Query().Get("order_id")
		queryID  = req.URL.Query().Get("query_id")
		txnTime  = req.URL.Query().Get("txn_time")
		reserved = req.URL.Query().Get("reserved")
	)

	queryResp, err := up.ConsumeQuery(orderID, queryID, txnTime, reserved)
	if err != nil {
		fmt.Fprintf(w, "Error:%s", err.Error())
		return
	}

	fmt.Fprintf(w, "%s", queryResp)
	return
}

func RefundServer(w http.ResponseWriter, req *http.Request) {
	var (
		orderID       = req.URL.Query().Get("order_id")
		amount, _     = strconv.ParseInt(req.URL.Query().Get("amount"), 10, 64)
		originQueryID = req.URL.Query().Get("origin_query_id")
		reqReserved   = req.URL.Query().Get("req_reserved")
		reserved      = req.URL.Query().Get("reserved")
	)

	refundResp, err := up.ConsumeRefund(orderID, returnURL, amount, originQueryID, reqReserved, reserved)
	if err != nil {
		fmt.Fprintf(w, "Error:%s", err.Error())
		return
	}

	fmt.Fprintf(w, "%s", refundResp)
	return
}

func UndoServer(w http.ResponseWriter, req *http.Request) {
	var (
		orderID       = req.URL.Query().Get("order_id")
		amount, _     = strconv.ParseInt(req.URL.Query().Get("amount"), 10, 64)
		originQueryID = req.URL.Query().Get("origin_query_id")
		reqReserved   = req.URL.Query().Get("req_reserved")
		reserved      = req.URL.Query().Get("reserved")
	)

	refundResp, err := up.ConsumeUndo(orderID, returnURL, amount, originQueryID, reqReserved, reserved)
	if err != nil {
		fmt.Fprintf(w, "Error:%s", err.Error())
		return
	}

	fmt.Fprintf(w, "%s", refundResp)
	return
}

func NotifyWebServer(w http.ResponseWriter, req *http.Request) {
	notifyResp, err := up.FrontConsumeNotify(req)
	if err != nil {
		log.Println(err)
		io.WriteString(w, "FAIL")
		return
	}

	if notifyResp.RespCode != "00" {
		log.Println(notifyResp)
		io.WriteString(w, "FAIL")
		return
	}

	io.WriteString(w, "SUCCESS")
	return
}

func NotifyAppServer(w http.ResponseWriter, req *http.Request) {
	notifyResp, err := up.MobilePaymentNotify(req)
	if err != nil {
		log.Println(err)
		io.WriteString(w, "FAIL")
		return
	}

	if notifyResp.RespCode != "00" {
		log.Println(notifyResp)
		io.WriteString(w, "FAIL")
		return
	}

	io.WriteString(w, "SUCCESS")
	return
}

func NotifyRefundServer(w http.ResponseWriter, req *http.Request) {
	notifyResp, err := up.ConsumeRefundNotify(req)
	if err != nil {
		log.Println(err)
		io.WriteString(w, "FAIL")
		return
	}

	if notifyResp.RespCode != "00" {
		log.Println(notifyResp)
		io.WriteString(w, "FAIL")
		return
	}

	io.WriteString(w, "SUCCESS")
	return
}

func NotifyUndoServer(w http.ResponseWriter, req *http.Request) {
	notifyResp, err := up.ConsumeUndoNotify(req)
	if err != nil {
		log.Println(err)
		io.WriteString(w, "FAIL")
		return
	}

	if notifyResp.RespCode != "00" {
		log.Println(notifyResp)
		io.WriteString(w, "FAIL")
		return
	}

	io.WriteString(w, "SUCCESS")
	return
}
