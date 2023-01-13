package main

import (
	"bytes"
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	log "github.com/sirupsen/logrus"
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
