package pkg

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	admv1 "k8s.io/api/admission/v1"
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
type WebhookServer struct {
	Server          *http.Server
	RegistryWhiteIp []string
	// 请求的白名单
}

// Serve method for webhook server
func (serv *WebhookServer) Serve(w http.ResponseWriter, r *http.Request) {
	var body []byte
	if r.Body != nil {
		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		}
	}
	if len(body) == 0 {
		klog.Error("empty body")
		http.Error(w, "empty body", http.StatusBadRequest)
		return
	}

	// verify the content type is accurate
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		klog.Errorf("Content-Type=%s, expect application/json", contentType)
		http.Error(w, "invalid Content-Type, expect `application/json`", http.StatusUnsupportedMediaType)
		return
	}

	var admissionResponse *admv1.AdmissionResponse
	requestedAdmissionReview := admv1.AdmissionReview{}
	if _, _, err := deserializer.Decode(body, nil, &requestedAdmissionReview); err != nil {
		klog.Errorf("Can't decode body: %v", err)
		admissionResponse = &admv1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	} else {
		if r.URL.Path == "/mutate" {
			//admissionResponse = serv.mutate(&requestedAdmissionReview)
		} else if r.URL.Path == "/validate" {
			admissionResponse = serv.validate(&requestedAdmissionReview)
		}
	}

	// 构造返回的 AdmissionReview 结构
	responseAdmissionReview := admv1.AdmissionReview{}
	// admission.k8s.io/v1 版本需要指定对应的 APIVersion
	responseAdmissionReview.APIVersion = requestedAdmissionReview.APIVersion
	responseAdmissionReview.Kind = requestedAdmissionReview.Kind
	if admissionResponse != nil {
		// 设置 response 属性
		responseAdmissionReview.Response = admissionResponse
		if requestedAdmissionReview.Request != nil {
			// 返回相同的 UID
			responseAdmissionReview.Response.UID = requestedAdmissionReview.Request.UID
		}
	}

	klog.Info(fmt.Sprintf("sending response: %v", responseAdmissionReview.Response))

	respBytes, err := json.Marshal(responseAdmissionReview)
	if err != nil {
		klog.Errorf("Can't encode response: %v", err)
		http.Error(w, fmt.Sprintf("could not encode response: %v", err), http.StatusInternalServerError)
	}
	klog.Infof("Ready to write response ...")
	if _, err := w.Write(respBytes); err != nil {
		klog.Errorf("Can't write response: %v", err)
		http.Error(w, fmt.Sprintf("could not write response: %v", err), http.StatusInternalServerError)
	}
}

// validate pod
func (serv *WebhookServer) validate(ar *admv1.AdmissionReview) *admv1.AdmissionResponse {
	req := ar.Request
	var (
		allowed = true
		code    = 200
		message = ""
	)

	klog.Infof("AdmissionReview for Kind=%s, Namespace=%s Name=%v UID=%v Operation=%v UserInfo=%v",
		req.Kind.Kind, req.Namespace, req.Name, req.UID, req.Operation, req.UserInfo)

	var pod corev1.Pod
	if err := json.Unmarshal(req.Object.Raw, &pod); err != nil {
		klog.Errorf("Could not unmarshal raw object: %v", err)
		allowed = false
		code = 400
		return &admv1.AdmissionResponse{
			Allowed: allowed,
			Result: &metav1.Status{
				Code:    int32(code),
				Message: err.Error(),
			},
		}
	}
	for _, container := range pod.Spec.Containers {
		var whitelisted = false
		for _, reg := range serv.RegistryWhiteIp {
			if strings.HasPrefix(container.Image, reg) {
				whitelisted = true
			}
		}
		if !whitelisted {
			allowed = false
			code = 403
			message = fmt.Sprintf("%s image comes from an untrusted registry! Only images from %v are allowed.",
				container.Image, serv.RegistryWhiteIp)
			break
		}
	}

	return &admv1.AdmissionResponse{
		Allowed: allowed,
		Result: &metav1.Status{
			Code:    int32(code),
			Message: message,
		},
	}
}
