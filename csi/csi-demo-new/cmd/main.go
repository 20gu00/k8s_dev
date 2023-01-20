package main

import (
	"flag"
	"github.com/golang/glog"
	"math/rand"
	"os"
	"time"

	csiDriver "devcsi/csi-plugin"
)

func init() {
	// 设置标准错误输出
	// 相当于-logstderr=true,直接写好
	flag.Set("logtostderr", "true")
}

var (
	/*
		csi地址
		csi驱动名称
		节点id
		版本
	*/
	endpoint      = flag.String("csi-address", "unix://tmp/csi.sock", "CSI endpoint")
	driverName    = flag.String("drivername", "my-csi-driver", "name of the driver")
	nodeID        = flag.String("nodeid", "", "node id")
	vendorVersion = "1.0.0"
)

func main() {
	flag.Parse()
	// 根据时间设置种子
	rand.Seed(time.Now().UnixNano())

	handle()
	os.Exit(0)
}

// 处理函数
func handle() {
	// 实例化 csi driver
	driver := csiDriver.GetCSIDriver()
	// 初始化 csi driver
	// 驱动名称 版本 节点id
	err := driver.InitializeDriver(*driverName, vendorVersion, *nodeID)
	if err != nil {
		glog.Fatalf("初始化 csi driver 失败: %v", err)
	}
	// 运行 csi driver
	driver.Run(*endpoint)
}
