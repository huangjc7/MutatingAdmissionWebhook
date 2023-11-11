package main

import (
	"MutatingAdmissionWebhook/pkg"
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"github.com/gin-gonic/gin"
	"k8s.io/klog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

func main() {
	var param pkg.WhSvrParam
	flag.IntVar(&param.Port, "port", 443, "Web hook service port")
	flag.StringVar(&param.CertFile, "tlsCertFile", "/etc/webhook/certs/tls.crt", "x509 cert file")
	flag.StringVar(&param.KeyFile, "tlsKeyFile", "/etc/webhook/certs/tls.key", "x509 key file")
	flag.Parse()

	cert, err := tls.LoadX509KeyPair(param.CertFile, param.KeyFile)
	if err != nil {
		klog.Errorf("Failed to load certificate file:", err)
		return
	}

	//创建tls配置
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{
			cert,
		},
	}
	//实例化一个Whsvr
	whsvr := pkg.WhSvr{
		Server: &http.Server{
			Addr:      fmt.Sprintf(":%d", param.Port),
			TLSConfig: tlsConfig,
		},
		WhiteListRegistries: strings.Split(os.Getenv("WHITELIST_REGISTRY"), ","),
	}

	//定义http server handler
	g := gin.Default()

	g.GET("/validate", pkg.HandlerFunc)
	g.GET("/mutate", pkg.HandlerFunc)
	whsvr.Server.Handler = g
	//在一个新的goroutine去启动webhook server

	go func() {
		if whsvr.Server.ListenAndServeTLS("", ""); err != nil {
			klog.Errorf("Failed to listen and service webhook: %v", err)
		}
	}()
	klog.Info("Server started")

	//监听OS关闭信号
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan

	klog.Infof("Got OS shutdown...")
	//关闭后台线程
	if err := whsvr.Server.Shutdown(context.Background()); err != nil {
		klog.Errorf("Http server shutdown error: %v", err)
	}
}
