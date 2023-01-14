package main

import (
	"bytes"
	"context"
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	log "github.com/sirupsen/logrus"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"math/big"
	"os"
	"time"
)

func main() {
	var caPEM, serverCertPEM, serverPrivKeyPEM *bytes.Buffer
	// CA config 配置,用于生成ca证书
	ca := &x509.Certificate{
		//序列号 标识
		SerialNumber: big.NewInt(2021),
		Subject: pkix.Name{
			Organization: []string{"cjq.io"},
			//还可以有
			Country:            []string{"CN"},
			Province:           []string{"Beijing"},
			Locality:           []string{"Beijing"},
			OrganizationalUnit: []string{"cjq.io"},
		},
		// 证书从什么时候开始有效
		NotBefore: time.Now(),
		// 什么时候开始无效
		NotAfter: time.Now().AddDate(1, 0, 0),
		// 根ca,用于颁发证书
		IsCA: true,
		// 私钥拓展用途 客户端加密,服务端加密
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		// 私钥用途 数字签名 证书签名
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	// 生成ca私钥 CA private key
	// io.reader 4096为加密
	caPrivKey, err := rsa.GenerateKey(cryptorand.Reader, 4096)
	if err != nil {
		fmt.Println(err)
	}

	// 生成自签名的ca证书的 Self signed CA certificate
	// 两个ca相同也就是自签名
	// 非对称加密 私钥容易推出公钥
	caBytes, err := x509.CreateCertificate(cryptorand.Reader, ca, ca, &caPrivKey.PublicKey, caPrivKey)
	if err != nil {
		fmt.Println(err)
	}

	// pem编码证书文件 PEM encode CA cert
	caPEM = new(bytes.Buffer)
	_ = pem.Encode(caPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	})

	// 以上就相当于实现了一个ca机构

	// 服务端(admission webhook)的域名信息
	dnsNames := []string{"admission-registry",
		"admission-registry.default", "admission-registry.default.svc",
		"admission-registry.default.svc.cluster.local"}
	commonName := "admission-registry.default.svc"

	// 服务端的证书配置server cert config
	cert := &x509.Certificate{
		DNSNames:     dnsNames,
		SerialNumber: big.NewInt(1658),
		Subject: pkix.Name{
			CommonName:   commonName,
			Organization: []string{"cjq.io"},

			Country:            []string{"CN"},
			Province:           []string{"Beijing"},
			Locality:           []string{"Beijing"},
			OrganizationalUnit: []string{"cjq.io"},
		},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(1, 0, 0),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	// 生成服务端私钥 server private key
	serverPrivKey, err := rsa.GenerateKey(cryptorand.Reader, 4096)
	if err != nil {
		fmt.Println(err)
	}

	// ca用私钥给服务端证书签名,证书里面有服务端的公钥 sign the server cert
	serverCertBytes, err := x509.CreateCertificate(cryptorand.Reader, cert, ca, &serverPrivKey.PublicKey, caPrivKey)
	if err != nil {
		fmt.Println(err)
	}

	// 给服务端证书做pem编码 PEM encode the server cert and key
	serverCertPEM = new(bytes.Buffer)
	_ = pem.Encode(serverCertPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: serverCertBytes,
	})

	serverPrivKeyPEM = new(bytes.Buffer)
	_ = pem.Encode(serverPrivKeyPEM, &pem.Block{
		// 行首
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(serverPrivKey),
	})

	// 写入tls.crt和tls.key
	// 创建目录
	err = os.MkdirAll("/etc/webhook/certs/", 0666)
	if err != nil {
		log.Panic(err)
	}
	err = WriteFile("/etc/webhook/certs/tls.crt", serverCertPEM)
	if err != nil {
		log.Panic(err)
	}

	err = WriteFile("/etc/webhook/certs/tls.key", serverPrivKeyPEM)
	if err != nil {
		log.Panic(err)
	}

	log.Println("webhook server的tls服务完成")

	// caBundle设置需要caPEM
	if err := CreateAdmissionConfig(caPEM); err != nil {
		log.Panic(err)
	}
	log.Println("webhook的configuration资源对象生成成功")

}

// 自动注册准入控制器,即validate mutate两种资源
func CreateAdmissionConfig(caCert *bytes.Buffer) error {
	// 针对前面的webhook处理,配置已知,也可以从环境变量中获取
	webhookNamespace, _ := os.LookupEnv("WEBHOOK_NAMESPACE")
	validateConfigName, _ := os.LookupEnv("VALIDATE_CONFIG")
	mutateConfigName, _ := os.LookupEnv("MUTATE_CONFIG")
	webhookService, _ := os.LookupEnv("WEBHOOK_SERVICE")
	validatePath, _ := os.LookupEnv("VALIDATE_PATH")
	mutatePath, _ := os.LookupEnv("MUTATE_PATH")

	ctx := context.Background()
	clientset, _ := InitKubernetesCli()
	// 不为空则创建
	// admissionregistration
	if validateConfigName != "" {
		validateConfig := &admissionregistrationv1.ValidatingWebhookConfiguration{
			ObjectMeta: metav1.ObjectMeta{
				Name: validateConfigName,
			},
			Webhooks: []admissionregistrationv1.ValidatingWebhook{
				{
					Name: "io.ydzs.admission-registry",
					ClientConfig: admissionregistrationv1.WebhookClientConfig{
						// []byte yaml就是ca.pem的base64编码
						CABundle: caCert.Bytes(),
						Service: &admissionregistrationv1.ServiceReference{
							Name:      webhookService,
							Namespace: webhookNamespace,
							Path:      &validatePath,
						},
					},
					Rules: []admissionregistrationv1.RuleWithOperations{
						{
							// create update delete
							Operations: []admissionregistrationv1.OperationType{admissionregistrationv1.Create},
							Rule: admissionregistrationv1.Rule{
								APIGroups:   []string{""},
								APIVersions: []string{"v1"},
								Resources:   []string{"pods"},
							},
						},
					},
					FailurePolicy: func() *admissionregistrationv1.FailurePolicyType {
						pt := admissionregistrationv1.Fail
						return &pt
					}(),
					AdmissionReviewVersions: []string{"v1"},
					// none
					SideEffects: func() *admissionregistrationv1.SideEffectClass {
						se := admissionregistrationv1.SideEffectClassNone
						return &se
					}(),
				},
			},
		}
		validateAdmissionClient := clientset.AdmissionregistrationV1().ValidatingWebhookConfigurations()
		_, err := validateAdmissionClient.Get(ctx, validateConfigName, metav1.GetOptions{})
		if err != nil {
			// ValidatingWebhookConfiguration错误是不存在
			if errors.IsNotFound(err) {
				if _, err = validateAdmissionClient.Create(ctx, validateConfig, metav1.CreateOptions{}); err != nil {
					return err
				}
			} else {
				return err
			}
		} else { // 存在,更新
			if _, err = validateAdmissionClient.Update(ctx, validateConfig, metav1.UpdateOptions{}); err != nil {
				return err
			}
		}
	}

	if mutateConfigName != "" {
		mutateConfig := &admissionregistrationv1.MutatingWebhookConfiguration{
			ObjectMeta: metav1.ObjectMeta{
				Name: mutateConfigName,
			},
			Webhooks: []admissionregistrationv1.MutatingWebhook{
				{
					Name: "io.ydzs.admission-registry-mutate",
					ClientConfig: admissionregistrationv1.WebhookClientConfig{
						CABundle: caCert.Bytes(), // CA bundle created earlier
						Service: &admissionregistrationv1.ServiceReference{
							Name:      webhookService,
							Namespace: webhookNamespace,
							Path:      &mutatePath,
						},
					},
					Rules: []admissionregistrationv1.RuleWithOperations{
						{
							Operations: []admissionregistrationv1.OperationType{
								admissionregistrationv1.Create},
							Rule: admissionregistrationv1.Rule{
								APIGroups:   []string{"apps", ""},
								APIVersions: []string{"v1"},
								Resources:   []string{"deployments", "services"},
							},
						},
					},
					FailurePolicy: func() *admissionregistrationv1.FailurePolicyType {
						pt := admissionregistrationv1.Fail
						return &pt
					}(),
					AdmissionReviewVersions: []string{"v1"},
					SideEffects: func() *admissionregistrationv1.SideEffectClass {
						se := admissionregistrationv1.SideEffectClassNone
						return &se
					}(),
				},
			},
		}
	}

	mutateAdmissionClient := clientset.AdmissionregistrationV1().MutatingWebhookConfigurations()
	_, err := mutateAdmissionClient.Get(ctx, mutationCfgName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			if _, err = mutateAdmissionClient.Create(ctx, mutateConfig, metav1.CreateOptions{}); err != nil {
				return err
			}
		} else {
			return err
		}
	} else {
		if _, err = mutateAdmissionClient.Update(ctx, mutateConfig, metav1.UpdateOptions{}); err != nil {
			return err
		}
	}
}
}

// WriteFile writes data in the file at the given path
func WriteFile(filepath string, sCert *bytes.Buffer) error {
	f, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(sCert.Bytes())
	if err != nil {
		return err
	}
	return nil
}

// 初始化kubernetes的客户端工具,操作集群资源
func InitKubernetesCli() (*kubernetes.Clientset, error) {
	var (
		config *rest.Config
		err    error
	)
	if config, err = rest.InClusterConfig(); err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return clientset, nil
}
