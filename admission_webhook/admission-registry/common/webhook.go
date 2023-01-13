package common

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	admissionV1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/klog"
	"net/http"
	"strings"
)

var (
	runtimeSchema = runtime.NewScheme()
	// admission数据使用的序列化
	codeFactory = serializer.NewCodecFactory(runtimeSchema)
	//反序列化,也就是解码
	deserializer = codeFactory.UniversalDeserializer()
)

// webhook server的参数
type WebhookSrvParam struct {
	Port     int
	CertFile string
	KeyFile  string
}

// 使用原生net/http实现,也可以使用框架gin等
// 封装一个webhook server
type WebhookSrv struct {
	Server          *http.Server
	RegistryWhiteIp []string
	// 请求的白名单
}

// handler 函数中会根据传入的 PATH 来决定调用的逻辑
func (w *WebhookSrv) Handler(writer http.ResponseWriter, request *http.Request) {
	var body []byte // nil
	if request.Body != nil {
		//:=不同body,注意处理上面的容器为nil
		if data, err := ioutil.ReadAll(request.Body); err != nil {
			if len(data) == 0 {
				klog.Error("空的body")
				// http响应
				http.Error(writer, "空的body", http.StatusBadRequest)
				return
			}
			klog.Error("获取body信息失败")
			http.Error(writer, "获取body信息失败", http.StatusBadRequest)
			return
		} else {
			body = data
		}
	}
	// 校验content-type
	// 从准入控制器传递过来给我们自定义的admission webhook 是json字符串,但实际格式是admission review
	if contentType := request.Header.Get("Content-Type"); contentType != "application/json" {
		klog.Errorf("content-type is %s ,but expect application/json", contentType)
		http.Error(writer, "content-type error ,expect application/json", http.StatusBadRequest)
	}

	// validate mutate请求的数据(交互,与admission controller之间,请求响应都是AdmissionReview)
	// 数据编码
	//准入控制器发送request也就是AdmissionRequstReview给我们admission webhook,反之就是admissionReponseReview

	//var admissionResponse *admissionV1.AdmissionResponse
	admissionResponse := new(admissionV1.AdmissionResponse)
	// 获取请求过来的admissionReview
	requestAdmissionReview := admissionV1.AdmissionReview{}
	if _, _, err := deserializer.Decode(body, nil, &requestAdmissionReview); err != nil {
		klog.Error("未能正确解码body成admissionReview数据")
		admissionResponse = &admissionV1.AdmissionResponse{
			Result: &metav1.Status{
				Code:    http.StatusInternalServerError,
				Message: err.Error(),
			},
		}
	} else {
		// 序列化成功,也就是成功拿到AdmissionRequestReview
		if request.URL.Path == "/mutate" {

		} else if request.URL.Path == "/validate" {
			admissionResponse = w.validate(&requestAdmissionReview)
		}
	}

	// 构造返回的admissionReview
	responseAdmissionReview := admissionV1.AdmissionReview{}
	// admission/v1
	// 指定版本
	responseAdmissionReview.APIVersion = requestAdmissionReview.APIVersion
	responseAdmissionReview.Kind = requestAdmissionReview.Kind

	// 有返回值
	if admissionResponse != nil {
		responseAdmissionReview.Response = admissionResponse
		if requestAdmissionReview.Request != nil {
			// 请求和返回的admission review的uid要一致,表明是一次请求
			responseAdmissionReview.Response.UID = requestAdmissionReview.Request.UID
		}
	}
	klog.Info(fmt.Sprintf("正在返回reponse数据: %v", responseAdmissionReview.Response))
	respBytes, err := json.Marshal(responseAdmissionReview)
	if err != nil {
		klog.Error("response admission review不能正确编码json")
		http.Error(writer, fmt.Sprintf("response admission review不能正确编码json: %v", err), http.StatusInternalServerError)
		return
	}

	klog.Info("响应准备好了...")
	// 返回响应
	if _, err := writer.Write(respBytes); err != nil {
		klog.Errorf("返回响应失败: %v", err)
		http.Error(writer, fmt.Sprintf("返回响应失败: %v", err), http.StatusInternalServerError)
	}
}

// 返回admissionResponse
func (w *WebhookSrv) validate(ar *admissionV1.AdmissionReview) *admissionV1.AdmissionResponse {
	req := ar.Request
	var (
		allowed = true
		code    = http.StatusOK
		msg     = ""
	)
	// req.Kind是admission review的group version kind
	klog.Infof("request admission review: kind=%s,ns=%s,name=%v,uid=%v,Operation=%v UserInfo=%v", req.Kind.Kind, req.Namespace, req.Name, req.UID, req.Operation, req.UserInfo)

	// 规则,判断镜像仓库,也就是校验pod
	var pod corev1.Pod
	// 获取原始的数据
	if err := json.Unmarshal(req.Object.Raw, &pod); err != nil {
		klog.Errorf("未能获取object的raw数据: %v", err)
		allowed = false
		code = http.StatusBadRequest
		return &admissionV1.AdmissionResponse{
			Allowed: allowed,
			Result: &metav1.Status{
				Code:    int32(code),
				Message: err.Error(),
			},
		}
	}

	// 业务逻辑
	// InitContainers
	for _, container := range pod.Spec.Containers {
		var whiteList = false
		for _, whiteIP := range w.RegistryWhiteIp {
			// 前缀
			if strings.HasPrefix(container.Image, whiteIP) {
				// 命中白名单
				whiteList = true
			}
			// 没有命中白名单
			if !whiteList {
				allowed = false
				code = http.StatusForbidden
				msg = fmt.Sprintf("%s image来自的镜像仓库不受信任,只信任来自%s的镜像", container.Image, w.RegistryWhiteIp)
				break // 任意一个都可以处理了
			}
		}
	}
	return &admissionV1.AdmissionResponse{
		Allowed: allowed,
		Result: &metav1.Status{
			Code:    int32(code),
			Message: msg,
		},
	}
}
