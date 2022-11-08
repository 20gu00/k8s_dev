package file

import (
	"context"
	"github.com/minio/minio-go/v7"

	"github.com/minio/minio-go/v7/pkg/credentials"
)

type s3Uploader struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
}

func NewS3Uploader(Endpoint, AK, SK string) *s3Uploader {
	return &s3Uploader{
		Endpoint:        Endpoint,
		AccessKeyID:     AK,
		SecretAccessKey: SK,
	}
}

// 初使化 minio client 对象
func (su *s3Uploader) InitClient() (*minio.Client, error) {
	return minio.New(su.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(su.AccessKeyID, su.SecretAccessKey, ""),
		Secure: true,
	})
}

func (su *s3Uploader) Upload(ctx context.Context, filePath string) (int64, error) {
	client, err := su.InitClient()
	if err != nil {
		return 0, err
	}
	bucketName := "testback"         // todo
	objectName := "etcd-snapshot.db" // todo
	uploadInfo, err := client.FPutObject(ctx, bucketName, objectName, filePath, minio.PutObjectOptions{})
	if err != nil {
		return 0, err
	}
	return uploadInfo.Size, nil
}
