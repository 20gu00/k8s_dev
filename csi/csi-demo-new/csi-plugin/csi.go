package mycsi

import (
	"fmt"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/golang/glog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	// 插件的文件夹
	PluginFolder = "/var/lib/kubelet/plugins/my-csi-driver"
)

// 定义 csi driver
type MyCSIDriver struct {
	/*
			名称 版本 节点id
			csi-identity 暴露csi插件本身信息,确保插件的健康状态
			csi-controller volume管理流程中的provison和attach(deattch)
			provison 创建和删除volume  attach 将存储卷附着在某个节点或者脱离某个节点,只有块存储需要attch
			csi-node 管理节点volume,分为NodeStorageVolume和NodePublishVolume两个阶段
			(NodeStageVolume针对块设备只能挂在一次,不满足volume可以同时挂在进多个pod,多个容器,于是这一阶段是讲volume格式化成文件系统,然后挂在到某个临时目录中)
		    (NodePulishVolume是将临时目录挂载进pod的对应的目录中,所以总的来看k8s是直接操作pv,pvc是面向用户,品比了复杂的存储实现逻辑,实现技术和关注点的解耦)
	*/
	name          string
	vendorVersion string
	nodeID        string

	ids *IdentityServer
	ns  *NodeServer
	cs  *ControllerServer

	// volume的capability accessmode访问模式校验
	// csi-controller csi-node的capability
	vcap  []*csi.VolumeCapability_AccessMode
	cscap []*csi.ControllerServiceCapability
	nscap []*csi.NodeServiceCapability
}

func init() {
	glog.Infof("csi init...")
}

// 获取 实例化 csi driver
func GetCSIDriver() *MyCSIDriver {
	glog.Infof("mycsi: GetCSIDriver")
	return &MyCSIDriver{}
}

func NewIdentityServer(d *MyCSIDriver) *IdentityServer {
	glog.V(3).Infof("mycsi: NewIdentityServer")
	return &IdentityServer{
		Driver: d,
	}
}

func NewControllerServer(d *MyCSIDriver) *ControllerServer {
	glog.V(3).Infof("mycsi: NewControllerServer")
	return &ControllerServer{
		Driver: d,
	}
}

func NewNodeServer(d *MyCSIDriver) *NodeServer {
	glog.V(3).Infof("mycsi: NewNodeServer")
	return &NodeServer{
		Driver: d,
	}
}

// 初始化 csi driver
func (driver *MyCSIDriver) InitializeDriver(name, vendorVersion, nodeID string) error {
	glog.V(3).Infof("mycsi: InitializeDriver. name: %s, version: %v, nodeID: %s", name, vendorVersion, nodeID)
	if name == "" {
		return fmt.Errorf("Driver name missing")
	}

	err := driver.PluginInitialize()
	if err != nil {
		glog.Errorf("插件初始化失败: %s", err)
		return err
	}

	// csi 驱动的信息
	driver.name = name
	driver.vendorVersion = vendorVersion
	driver.nodeID = nodeID

	// 设置 csi 的 capability(volume) accessmode
	vcam := []csi.VolumeCapability_AccessMode_Mode{
		// rwx 多个节点读写,后续volume校验访问模式有用
		csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER,
	}
	// add volume capability
	driver.AddVolumeCapabilityAccessModes(vcam)

	// csi-controller 的 capability
	csc := []csi.ControllerServiceCapability_RPC_Type{
		// 对 volume 的创建删除 容量修改,挂载和卸载(挂载到节点)
		csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
		csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME,
		csi.ControllerServiceCapability_RPC_EXPAND_VOLUME,
	}
	// add csi-controller capability
	driver.AddControllerServiceCapabilities(csc)

	ns := []csi.NodeServiceCapability_RPC_Type{
		// 容量修改 stage/unstage 针对块存储是否格式化挂载进某个临时全局目录
		csi.NodeServiceCapability_RPC_STAGE_UNSTAGE_VOLUME,
		csi.NodeServiceCapability_RPC_EXPAND_VOLUME,
	}
	// add csi-node capability
	driver.AddNodeServiceCapabilities(ns)

	driver.ids = NewIdentityServer(driver)
	driver.ns = NewNodeServer(driver)
	driver.cs = NewControllerServer(driver)
	return nil
}

// csi 插件初始化
func (driver *MyCSIDriver) PluginInitialize() error {
	glog.V(3).Infof("mycsi: PluginInitialize")
	return nil
}

// csi driver run
func (driver *MyCSIDriver) Run(endpoint string) {
	glog.Infof("Driver: %v version: %v", driver.name, driver.vendorVersion)
	s := NewNonBlockingGRPCServer()
	s.Start(endpoint, driver.ids, driver.cs, driver.ns)
	s.Wait()
}

func (driver *MyCSIDriver) ValidateControllerServiceRequest(c csi.ControllerServiceCapability_RPC_Type) error {
	glog.V(3).Infof("mycsi: ValidateControllerServiceRequest")
	if c == csi.ControllerServiceCapability_RPC_UNKNOWN {
		return nil
	}
	for _, cap := range driver.cscap {
		if c == cap.GetRpc().Type {
			return nil
		}
	}
	return status.Error(codes.InvalidArgument, "Invalid controller service request")
}

func (driver *MyCSIDriver) AddVolumeCapabilityAccessModes(vc []csi.VolumeCapability_AccessMode_Mode) error {
	glog.V(3).Infof("mycsi: AddVolumeCapabilityAccessModes")
	var vca []*csi.VolumeCapability_AccessMode
	for _, c := range vc {
		glog.V(3).Infof("Enabling volume access mode: %v", c.String())
		vca = append(vca, NewVolumeCapabilityAccessMode(c))
	}
	driver.vcap = vca
	return nil
}

func (driver *MyCSIDriver) AddControllerServiceCapabilities(cl []csi.ControllerServiceCapability_RPC_Type) error {
	glog.V(3).Infof("mycsi: AddControllerServiceCapabilities")
	var csc []*csi.ControllerServiceCapability
	for _, c := range cl {
		glog.V(3).Infof("Enabling controller service capability: %v", c.String())
		csc = append(csc, NewControllerServiceCapability(c))
	}
	driver.cscap = csc
	return nil
}

func (driver *MyCSIDriver) AddNodeServiceCapabilities(nl []csi.NodeServiceCapability_RPC_Type) error {
	glog.V(3).Infof("mycsi: AddNodeServiceCapabilities")
	var nsc []*csi.NodeServiceCapability
	for _, n := range nl {
		glog.V(3).Infof("Enabling node service capability: %v", n.String())
		nsc = append(nsc, NewNodeServiceCapability(n))
	}
	driver.nscap = nsc
	return nil
}
