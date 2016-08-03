package unionpay

import (
	"crypto"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/mitchellh/mapstructure"

	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"

	"io/ioutil"

	"strings"
)

const (
	frontTransReq = "/gateway/api/frontTransReq.do"
	backTransReq  = "/gateway/api/backTransReq.do"
	queryTrans    = "/gateway/api/queryTrans.do"
	appTransReq   = "/gateway/api/appTransReq.do"
)

var ErrNotifyDataIsEmpty = errors.New("notify data is empty")

type UnionPay struct {
	testEnv bool

	mchID string // 测试商户号 700000000000001
	/*
	   卡号	卡性质	机构名称	手机号码	密码	CVN2	有效期	证件号	姓名
	   6216261000000000018	借记卡	平安银行	13552535506	123456			341126197709218366	全渠道
	   6221558812340000	贷记卡	平安银行	13552535506	123456	123	1711	341126197709218366	互联网
	   短信验证码	111111

	*/
	verifySignCert *x509.Certificate //verify_sign_acp.cer
	publicKey      *x509.Certificate //加密密钥路径(openssl pkcs12 -in PM_700000000000001_acp.pfx -clcerts -nokeys -out key.cert)
	privateKey     *rsa.PrivateKey   //加密证书路径(openssl pkcs12 -in PM_700000000000001_acp.pfx -nocerts -nodes -out key.pem)

	client *unionPayClient
}

func (up *UnionPay) getHost() string {
	if up.testEnv {
		return "https://101.231.204.80:5000"
	}
	return "https://gateway.95516.com"
}

func (up *UnionPay) SetTestEnv(b bool) *UnionPay {
	up.testEnv = b
	return up
}

type unionPayClient struct {
	client         *http.Client
	verifySignCert *x509.Certificate
}

func (c *unionPayClient) PostForm(u *url.URL, form map[string][]string, ret interface{}) error {
	msg := url.Values(form).Encode()

	req, err := http.NewRequest("POST", u.String(), strings.NewReader(msg))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.ContentLength = int64(len(msg))

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	return c.Parse(ret, resp)
}

func (upp *unionPayClient) Parse(ret interface{}, resp *http.Response) (err error) {
	defer resp.Body.Close()

	var body []byte
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	if resp.StatusCode != 200 {
		err = fmt.Errorf("response code:%d", resp.StatusCode)
		return
	}

	var fields []string
	fields = strings.Split(string(body), "&")

	vals := url.Values{}
	data := map[string]string{}
	for _, field := range fields {
		f := strings.SplitN(field, "=", 2)
		if len(f) >= 2 {
			key, val := f[0], f[1]
			data[key] = val
			vals.Set(key, val)
		}
	}

	if err = verify(upp.verifySignCert.PublicKey.(*rsa.PublicKey), vals, nil); err != nil {
		return
	}

	if data["respCode"] != "00" {
		err = fmt.Errorf("response error:%s", data["respMsg"])
		return
	}

	err = mapstructure.Decode(data, ret)
	return err
}

func NewPayment(mchID, pubPath, priPath, certPath string) (up *UnionPay) {
	var (
		err        error
		cert       *x509.Certificate
		publicKey  *x509.Certificate
		privateKey *rsa.PrivateKey
	)

	publicKey, err = newPublicKey(pubPath)
	if err != nil {
		log.Fatal(err)
	}

	privateKey, err = newPrivateKey(priPath)
	if err != nil {
		log.Fatal(err)
	}

	cert, err = newCertificate(certPath)
	if err != nil {
		log.Fatal(err)
	}

	client, err := newHTTPSClient()
	if err != nil {
		log.Fatal(err)
	}

	up = &UnionPay{
		mchID: mchID,

		verifySignCert: cert,
		publicKey:      publicKey,
		privateKey:     privateKey,

		client: &unionPayClient{
			client:         client,
			verifySignCert: cert,
		},
	}

	return
}

func newHTTPSClient() (c *http.Client, err error) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	c = &http.Client{Transport: tr}
	return
}

func newPublicKey(path string) (cert *x509.Certificate, err error) {
	// Read the verify sign certification key
	pemData, err := ioutil.ReadFile(path)
	if err != nil {
		return
	}

	// Extract the PEM-encoded data block
	block, _ := pem.Decode(pemData)
	if block == nil {
		err = fmt.Errorf("bad key data: %s", "not PEM-encoded")
		return
	}
	if got, want := block.Type, "CERTIFICATE"; got != want {
		err = fmt.Errorf("unknown key type %q, want %q", got, want)
		return
	}

	// Decode the certification
	cert, err = x509.ParseCertificate(block.Bytes)
	if err != nil {
		err = fmt.Errorf("bad private key: %s", err)
		return
	}

	return
}

func newPrivateKey(path string) (priKey *rsa.PrivateKey, err error) {
	// Read the private key
	pemData, err := ioutil.ReadFile(path)
	if err != nil {
		err = fmt.Errorf("read key file: %s", err)
		return
	}

	// Extract the PEM-encoded data block
	block, _ := pem.Decode(pemData)
	if block == nil {
		err = fmt.Errorf("bad key data: %s", "not PEM-encoded")
		return
	}
	if got, want := block.Type, "RSA PRIVATE KEY"; got != want {
		err = fmt.Errorf("unknown key type %q, want %q", got, want)
		return
	}

	// Decode the RSA private key
	priKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		err = fmt.Errorf("bad private key: %s", err)
		return
	}

	return
}

func newCertificate(certpath string) (cert *x509.Certificate, err error) {
	pemData, err := ioutil.ReadFile(certpath)
	if err != nil {
		return
	}

	block, _ := pem.Decode(pemData)
	if block == nil {
		err = errors.New("cannot decode the pem file")
		return
	}
	if got, want := block.Type, "CERTIFICATE"; got != want {
		err = fmt.Errorf("unknown key type %q, want %q", got, want)
		return
	}

	// Decode the certificate
	cert, err = x509.ParseCertificate(block.Bytes)
	if err != nil {
		err = fmt.Errorf("bad private key: %s", err)
		return
	}
	return
}

func verify(certPubKey *rsa.PublicKey, vals url.Values, fields []string) (err error) {
	var signature string
	kvs := KVpairs{}
	for k := range vals {
		if len(fields) > 0 && !Contains(fields, k) {
			continue
		}

		if k == "signature" {
			signature = vals.Get(k)
			continue
		}

		if vals.Get(k) == "" {
			continue
		}

		kvs = append(kvs, KVpair{K: k, V: vals.Get(k)})
	}

	sig := SHA1([]byte(kvs.RemoveEmpty().Sort().Join("&")))

	hashed := SHA1([]byte(fmt.Sprintf("%x", sig)))

	var inSign []byte
	inSign, err = base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return
	}

	err = rsa.VerifyPKCS1v15(certPubKey, crypto.SHA1, hashed, inSign)
	if err != nil {
		return
	}
	return
}

func signature(priKey *rsa.PrivateKey, kvs KVpairs) (sig string, err error) {
	sha1ParamsStr := SHA1([]byte(kvs.RemoveEmpty().Sort().Join("&")))

	hashed := SHA1([]byte(fmt.Sprintf("%x", sha1ParamsStr)))

	rsaSign, err := rsa.SignPKCS1v15(nil, priKey, crypto.SHA1, hashed)
	if err != nil {
		return
	}

	sig = base64.StdEncoding.EncodeToString(rsaSign)
	return
}
