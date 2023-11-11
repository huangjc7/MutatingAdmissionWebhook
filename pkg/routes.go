package pkg

import (
	"github.com/gin-gonic/gin"
	"io/ioutil"
	"k8s.io/klog"
	"net/http"
)

func HandlerFunc(g *gin.Context) {
	var body []byte
	if g.Request.Body != nil {
		if data, err := ioutil.ReadAll(g.Request.Body); err != nil {
			body = data
		}
	}
	if len(body) == 0 {
		klog.Error("empty data body")
		g.String(http.StatusBadRequest, "empty data body")
		return
	}

	//校验content-type 必须要为json格式

	contentType := g.GetHeader("Content-Type")
	if contentType != "application/json" {
		klog.Error("Content-Type is %s, but expect application/json", contentType)
		g.String(http.StatusBadRequest, "Content-Type is %s, but expect application/json", contentType)
	}

}
