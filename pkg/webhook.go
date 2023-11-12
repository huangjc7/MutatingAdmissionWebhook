package pkg

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"io/ioutil"
	admissionV1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/klog"
	"net/http"
	"strings"
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

var (
	runtimeScheme = runtime.NewScheme()
	codeFactory   = serializer.NewCodecFactory(runtimeScheme)
	deserializer  = codeFactory.UniversalDecoder()
)

func (s *WhSvr) HandlerFunc(g *gin.Context) {
	var body []byte
	if g.Request.Body != nil {
		if data, err := ioutil.ReadAll(g.Request.Body); err != nil {
			body = data
		}
		//serializer.NewCodeFactory
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
		return
	}
	//数据序列化 (validate mutate)请求数据都是admissionReview
	// admissionResponse用于响应k8s的请求
	var admissionResponse *admissionV1.AdmissionResponse
	requestAdmissionReview := admissionV1.AdmissionReview{}
	if _, _, err := deserializer.Decode(body, nil, &requestAdmissionReview); err != nil {
		klog.Errorf("Can't decode body %v", err)
		admissionResponse = &admissionV1.AdmissionResponse{
			Result: &metav1.Status{
				Code:    http.StatusInternalServerError,
				Message: err.Error(),
			},
		}
	} else {
		//序列化成功 也就是说获取到了请求的admissionReview的数据
		if g.Request.URL.Path == "/mutate" {
			// TODO
		} else if g.Request.URL.Path == "/validate" {
			s.validate(&requestAdmissionReview)
		}
	}
	// 构造返回的admissionreview结构体
	responseAdmissionReview := admissionV1.AdmissionReview{}
	responseAdmissionReview.APIVersion = requestAdmissionReview.APIVersion
	responseAdmissionReview.Kind = requestAdmissionReview.Kind
	if admissionResponse == nil {
		responseAdmissionReview.Response = admissionResponse
		if requestAdmissionReview.Request != nil { //返回相同的uuid
			responseAdmissionReview.Response.UID = requestAdmissionReview.Request.UID
		}
	}
	klog.Info(fmt.Sprintf("sending response: %v", responseAdmissionReview.Request))
	//发送response
	respBytes, err := json.Marshal(responseAdmissionReview)
	if err != nil {
		klog.Error("Can't encode response err: %v", err)
		g.String(http.StatusBadRequest, "Can't encode response err: %v", err)
		return
	}
	klog.Info("Ready to write response...")
	if _, err := g.Writer.Write(respBytes); err != nil {
		klog.Errorf("can't write response %v", err)
		g.String(http.StatusBadRequest, "can't write response %v", err)
	}
}

func (s *WhSvr) validate(ar *admissionV1.AdmissionReview) *admissionV1.AdmissionResponse {
	req := ar.Request
	var (
		allowd  = true
		code    = http.StatusOK
		message = ""
	)

	klog.Infof("AdmissionReview for kind=%s, Namespace=%s, Name=%s, UID=%s",
		req.Kind, req.Namespace, req.Name, req.UID)

	var pod corev1.Pod

	if err := json.Unmarshal(req.Object.Raw, &pod); err != nil {
		klog.Errorf("Can't unmarshal object raw: %v", err)
		allowd = false
		code = http.StatusBadRequest
		return &admissionV1.AdmissionResponse{
			Allowed: allowd,
			Result: &metav1.Status{
				Code:    int32(code),
				Message: err.Error(),
			},
		}
	}

	//处理真正扽业务逻辑

	for _, container := range pod.Spec.Containers {
		var whitelisted = false
		for _, reg := range s.WhiteListRegistries {
			if strings.HasPrefix(container.Image, reg) {
				whitelisted = true
			}
		}

		if !whitelisted {
			allowd = false
			code = http.StatusForbidden
			message = fmt.Sprintf(container.Image)
			break
		}
	}

	return &admissionV1.AdmissionResponse{
		Allowed: allowd,
		Result: &metav1.Status{
			Code:    int32(code),
			Message: message,
		},
	}
}
