package main

import (
	"admissionRegistry2/pkg"
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
)

func main() {
	var param pkg.WebhookSrvParam

	// 命令行参数
	flag.IntVar(&param.Port, "port", 443, "Webhook server port.")
	flag.StringVar(&param.CertFile, "tlsCertFile", "/etc/webhook/certs/tls.crt", "File containing the x509 Certificate for HTTPS.")
	flag.StringVar(&param.KeyFile, "tlsKeyFile", "/etc/webhook/certs/tls.key", "File containing the x509 private key to --tlsCertFile.")
	flag.Parse()

	klog.Info(fmt.Sprintf("port=%d, cert-file=%s, key-file=%s", param.Port, param.CertFile, param.KeyFile))

	pair, err := tls.LoadX509KeyPair(param.CertFile, param.KeyFile)
	if err != nil {
		klog.Errorf("Failed to load key pair: %v", err)
		return
	}

	whsvr := &pkg.WebhookServer{
		Server: &http.Server{
			Addr:      fmt.Sprintf(":%v", param.Port),
			TLSConfig: &tls.Config{Certificates: []tls.Certificate{pair}},
		},
		RegistryWhiteIp: strings.Split(os.Getenv("WHITELIST_REGISTRIES"), ","),
	}

	// 定义 http server 和 handler
	mux := http.NewServeMux()
	mux.HandleFunc("/validate", whsvr.Serve)
	mux.HandleFunc("/mutate", whsvr.Serve)
	whsvr.Server.Handler = mux

	// 在一个新的 goroutine 中启动 webhook server
	go func() {
		if err := whsvr.Server.ListenAndServeTLS("", ""); err != nil {
			klog.Errorf("Failed to listen and serve webhook server: %v", err)
		}
	}()

	klog.Info("Server started")
	quit := make(chan os.Signal, 2)
	//SIGINT SIGTERM
	signal.Notify(quit, os.Interrupt)
	<-quit
	klog.Errorf("优雅关闭...")
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := whsvr.Server.Shutdown(ctx); err != nil {
			klog.Errorf("server关闭失败")
		}
		klog.Errorf("server优雅关闭完成")
	}()
	<-quit
	os.Exit(1)
}
