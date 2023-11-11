package pkg

import (
	"net/http"
)

type WhSvrParam struct {
	Port     int
	CertFile string
	KeyFile  string
}

type WhSvr struct {
	Server              *http.Server //http server
	WhiteListRegistries []string     //白名单列表
}
