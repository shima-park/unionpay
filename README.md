# unionpay
银联支付相关API封装 Golang实现

## 快速开始
### 获取安装
    go get -u github.com/shima-park/unionpay

### 推荐使用localtunnel测试回调通知
可以先安装一个[localtunnel](https://localtunnel.github.io/www/)
可以方便快捷的实现你的本地web服务通过外网访问，无需修改DNS和防火墙设置

```console
$ npm install -g localtunnel
```

## 示例

#### 通过localtunnel获取外网地址:

```console
$ lt --port 9090
your url is: http://eygytquvvu.localtunnel.me
```

#### 修改示例代码中的配置:
记得修改示例中的对应的pub, pri, cert, mchID配置
项目目录下已经存在银联的测试相关的公钥密钥以及证书
要注意的是go run main.go的时候取的是当前运行目录路径。
如果在example下运行该命令会导致找不到公钥密钥及证书

```golang
var (
    pub   = "key.cert" //加密密钥路径(openssl pkcs12 -in PM_700000000000001_acp.pfx -clcerts -nokeys -out key.cert)
	pri   = "key.pem"  //加密证书路径(openssl pkcs12 -in PM_700000000000001_acp.pfx -nocerts -nodes -out key.pem)
	cert  = "acp_test_verify_sign_new.cer"
	mchID = "700000000000001"

    // 默认调用银联正式环境的地址,访问银联测试环境调用 SetTestEnv(true)
	up = unionpay.NewPayment(mchID, pub, pri, cert).SetTestEnv(true)

    // 示例监听的端口
	port = ":9090"

    // 通过 lt --port 9090 获取的外网地址
	localTunnel = "http://eqfssupbgz.localtunnel.me"
    ...
)
```

#### 启动示例程序:

```console
$ go run example/main.go
```

#### 在浏览器中访问本地服务:
[http://localhost:9090/index](http://localhost:9090/index)

具体如何使用请查看[example/main.go](https://github.com/shima-park/unionpay/blob/master/example/main.go)
