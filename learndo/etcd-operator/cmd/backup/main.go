package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/clientv3/snapshot"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func loggedError(log logr.Logger, err error, message string) error {
	log.Error(err, message)
	return fmt.Errorf("%s: %s", message, err)
}

func main() {
	var (
		backupTempDir          string
		etcdURL                string
		etcdDialTimeoutSeconds int64
		timeoutSeconds         int64
	)

	flag.StringVar(&backupTempDir, "backup-tmp-dir", os.TempDir(), "The directory to temporarily place backups before they are uploaded to their destination.")
	flag.StringVar(&etcdURL, "etcd-url", "http://localhost:2379", "URL for etcd.")
	flag.Int64Var(&etcdDialTimeoutSeconds, "etcd-dial-timeout-seconds", 5, "Timeout, in seconds, for dialing the Etcd API.")
	flag.Int64Var(&timeoutSeconds, "timeout-seconds", 60, "Timeout, in seconds, of the whole restore operation.")
	flag.Parse()

	zapLogger := zap.NewRaw(zap.UseDevMode(true))
	ctrl.SetLogger(zapr.NewLogger(zapLogger))

	log := ctrl.Log.WithName("backup-agent")
	ctx, ctxCancel := context.WithTimeout(context.Background(), time.Second*time.Duration(timeoutSeconds))
	defer ctxCancel()

	log.Info("Connecting to Etcd and getting snapshot")
	localPath := filepath.Join(backupTempDir, "snapshot.db")
	etcdClient := snapshot.NewV3(zapLogger.Named("etcd-client"))
	err := etcdClient.Save(
		ctx,
		clientv3.Config{
			Endpoints:   []string{etcdURL},
			DialTimeout: time.Second * time.Duration(etcdDialTimeoutSeconds),
		},
		localPath,
	)
	if err != nil {
		panic(loggedError(log, err, "failed to get etcd snapshot"))
	}

	// 临时测试
	endpoint := "play.min.io"
	accessKeyID := "Q3AM3UQ867SPQQA43P2F"
	secretAccessKey := "zuf+tfteSlswRu7BJ86wekitnifILbZam1KYY3TG"
	s3Uploader := uploader.NewS3Uploader(endpoint, accessKeyID, secretAccessKey)

	log.Info("Uploading snapshot")
	size, err := s3Uploader.Upload(ctx, localPath)
	if err != nil {
		panic(loggedError(log, err, "failed to upload backup"))
	}
	log.WithValues("upload-size", size).Info("Backup complete")
}
