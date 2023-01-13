package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"k8s.io/klog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"
	"webhookDo/common"
)

func main() {
	var param common.WebhookSrvParam
	//生产环境很多都是加密
	//加密端口 证书 私钥文件
	flag.IntVar(&param.Port, "port", 443, "webhook server port")
	flag.StringVar(&param.CertFile, "tlsCertFile", "/etc/webhook/certs/tls.crt", "x509 certfile")
	flag.StringVar(&param.KeyFile, "tlsKeyFile", "/etc/webhook/certs/tls.key", "x509 private key file")
	flag.Parse()

	//处理证书和私钥,类似与颁发证书
	cert, err := tls.LoadX509KeyPair(param.CertFile, param.KeyFile)
	if err != nil {
		klog.Errorf("获取证书失败: %v", err)
		return
	}

	WebhookServer := common.WebhookSrv{
		Server: &http.Server{
			Addr: fmt.Sprintf(":%d", param.Port),
			TLSConfig: &tls.Config{
				Certificates: []tls.Certificate{
					cert,
				},
			},
		},
		RegistryWhiteIp: strings.Split(os.Getenv("WHITEIPS_REGISTRY"), ","),
	}

	// 创建http服务的handler
	// http请求的多路复用器
	mux := http.NewServeMux()
	mux.HandleFunc("/validate", WebhookServer.Handler)
	mux.HandleFunc("/mutate", WebhookServer.Handler)
	WebhookServer.Server.Handler = mux

	go func() {
		// 本身这里就会阻塞
		// 上面已经定义好了
		if err := WebhookServer.Server.ListenAndServeTLS("", ""); err != nil {
			klog.Errorf("启动server失败 %v", err)
		}
	}()

	klog.Info("server启动了")

	quit := make(chan os.Signal, 2)
	//SIGINT SIGTERM
	signal.Notify(quit, os.Interrupt)
	<-quit
	klog.Errorf("优雅关闭...")
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := WebhookServer.Server.Shutdown(ctx); err != nil {
			klog.Errorf("server关闭失败")
		}
		klog.Errorf("server优雅关闭完成")
	}()
	<-quit
	os.Exit(1)
}
