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

type MyCSIDriver struct {
	name          string
	vendorVersion string
	nodeID        string

	ids *IdentityServer
	ns  *NodeServer
	cs  *ControllerServer

	vcap  []*csi.VolumeCapability_AccessMode
	cscap []*csi.ControllerServiceCapability
	nscap []*csi.NodeServiceCapability
}

func init() {
	glog.Infof("My CSI init")
}

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

func (driver *MyCSIDriver) InitializeDriver(name, vendorVersion, nodeID string) error {
	glog.V(3).Infof("mycsi: InitializeDriver. name: %s, version: %v, nodeID: %s", name, vendorVersion, nodeID)
	if name == "" {
		return fmt.Errorf("Driver name missing")
	}

	err := driver.PluginInitialize()
	if err != nil {
		glog.Errorf("Error in plugin initialization: %s", err)
		return err
	}

	driver.name = name
	driver.vendorVersion = vendorVersion
	driver.nodeID = nodeID

	// Adding Capabilities
	vcam := []csi.VolumeCapability_AccessMode_Mode{
		csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER,
	}
	driver.AddVolumeCapabilityAccessModes(vcam)

	csc := []csi.ControllerServiceCapability_RPC_Type{
		csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
		csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME,
		csi.ControllerServiceCapability_RPC_EXPAND_VOLUME,
	}
	driver.AddControllerServiceCapabilities(csc)

	ns := []csi.NodeServiceCapability_RPC_Type{
		csi.NodeServiceCapability_RPC_STAGE_UNSTAGE_VOLUME,
		csi.NodeServiceCapability_RPC_EXPAND_VOLUME,
	}
	driver.AddNodeServiceCapabilities(ns)

	driver.ids = NewIdentityServer(driver)
	driver.ns = NewNodeServer(driver)
	driver.cs = NewControllerServer(driver)
	return nil
}

func (driver *MyCSIDriver) PluginInitialize() error {
	glog.V(3).Infof("mycsi: PluginInitialize")
	return nil
}

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
