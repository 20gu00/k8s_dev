package mycsi

import (
	"fmt"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/golang/glog"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Common allocation units
const (
	// 国际单位制 1024
	KiB int64 = 1024
	MiB int64 = 1024 * KiB
	GiB int64 = 1024 * MiB
	TiB int64 = 1024 * GiB
)

// 定义一个 csi-controller 的结构体
type ControllerServer struct {
	// csi driver
	Driver *MyCSIDriver
}

// 容量扩展
func (cs *ControllerServer) ControllerExpandVolume(ctx context.Context, req *csi.ControllerExpandVolumeRequest) (*csi.ControllerExpandVolumeResponse, error) {

	volumeID := req.GetVolumeId()
	if volumeID == "" {
		return nil, status.Error(codes.InvalidArgument, "VolumeID is not present")
	}

	// newSize := int(req.GetCapacityRange().GetRequiredBytes() / GiB)

	// Perform Actual Volume Expansion using your API

	return &csi.ControllerExpandVolumeResponse{
		CapacityBytes:         req.GetCapacityRange().GetRequiredBytes(),
		NodeExpansionRequired: false,
	}, nil
}

// ControllerGetCapabilities implements the default GRPC callout.
// ControllerGetCapabilities csi-controller 的 capbility(例如. 插件可能未实现 GetCapacity、Snapshotting)
func (cs *ControllerServer) ControllerGetCapabilities(ctx context.Context, req *csi.ControllerGetCapabilitiesRequest) (*csi.ControllerGetCapabilitiesResponse, error) {
	glog.V(4).Infof("ControllerGetCapabilities called with req: %#v", req)
	return &csi.ControllerGetCapabilitiesResponse{
		Capabilities: cs.Driver.cscap,
	}, nil
}

// ControllerPublishVolume 挂载 发布 volume 到节点
func (cs *ControllerServer) ControllerPublishVolume(ctx context.Context, req *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {

	glog.V(3).Infof("controllerserver ControllerPublishVolume")
	glog.V(4).Infof("ControllerPublishVolume : req %#v", req)

	if err := cs.Driver.ValidateControllerServiceRequest(csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME); err != nil {
		glog.V(3).Infof("invalid Publish volume request: %v", req)
		return nil, status.Error(codes.Internal, fmt.Sprintf("ControllerPublishVolume: ValidateControllerServiceRequest failed: %v", err))
	}

	nodeID := req.GetNodeId()
	if nodeID == "" {
		return nil, status.Error(codes.InvalidArgument, "NodeID not present")
	}

	volumeID := req.GetVolumeId()
	if volumeID == "" {
		return nil, status.Error(codes.InvalidArgument, "VolumeID is not present")
	}

	// Perform attaching the volume to the worker node using nodeID (worker node host name) and volumeID (volume being asked to be attached to worker node)

	return &csi.ControllerPublishVolumeResponse{}, nil
}

// ControllerUnpublishVolume 从节点卸载 volume
func (cs *ControllerServer) ControllerUnpublishVolume(ctx context.Context, req *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {

	glog.V(3).Infof("controllerserver ControllerUnpublishVolume")
	glog.V(4).Infof("ControllerUnpublishVolume : req %#v", req)

	if err := cs.Driver.ValidateControllerServiceRequest(csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME); err != nil {
		glog.V(3).Infof("invalid Unpublish volume request: %v", req)
		return nil, status.Error(codes.Internal, fmt.Sprintf("ControllerUnpublishVolume: ValidateControllerServiceRequest failed: %v", err))
	}

	nodeID := req.GetNodeId()
	if nodeID == "" {
		return nil, status.Error(codes.InvalidArgument, "NodeID not present")
	}

	volumeID := req.GetVolumeId()
	if volumeID == "" {
		return nil, status.Error(codes.InvalidArgument, "VolumeID is not present")
	}

	// Perform detaching the volume from the worker node using nodeID and volumeID

	return &csi.ControllerUnpublishVolumeResponse{}, nil
}

// CreateSnapshot 快照
func (cs *ControllerServer) CreateSnapshot(ctx context.Context, req *csi.CreateSnapshotRequest) (*csi.CreateSnapshotResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// DeleteSnapshot 删除快照
func (cs *ControllerServer) DeleteSnapshot(ctx context.Context, req *csi.DeleteSnapshotRequest) (*csi.DeleteSnapshotResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// ListSnapshots list 快照
func (cs *ControllerServer) ListSnapshots(ctx context.Context, req *csi.ListSnapshotsRequest) (*csi.ListSnapshotsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// CreateVolume 创建volume
func (cs *ControllerServer) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {

	glog.V(3).Infof("create volume req: %v", req)

	// 首先判断 grpc 调用 csi-controller 的请求是否由创建和删除 volume 的 capability
	if err := cs.Driver.ValidateControllerServiceRequest(csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME); err != nil {
		glog.V(3).Infof("create volume 无效的请求: %v", req)
		return nil, status.Error(codes.Internal, fmt.Sprintf("创建 volume 失败: %v", err))
	}

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "create volume CreateVolumeRequest请求不能为空")
	}

	// volume 的名称
	volName := req.GetName()
	if volName == "" {
		return nil, status.Error(codes.InvalidArgument, "volume 名称必填,为空失败")
	}

	// 获取 volume 的 byte大小的
	volSize, err := cs.GetVolumeSizeInBytes(req)
	if err != nil {
		return nil, err
	}
	// 换算成 GiB
	volSize = (volSize) / GiB

	// volume 的 capability
	reqCapabilities := req.GetVolumeCapabilities()
	if reqCapabilities == nil {
		return nil, status.Error(codes.InvalidArgument, "volume capability 必填")
	}

	// 获取 create volume 的参数
	parameters := req.GetParameters()

	// name := volName
	// sizeize := int(volSize)
	// volumeType := parameters["type"]
	// availabilityZone := parameters["availability"]
	// fsType := parameters["fstype"]
	// multiAttach := false

	/*
	   for _, reqCap := range reqCapabilities {
	           if reqCap.GetAccessMode().GetMode() == csi.VolumeCapability_AccessMode_MULTI_NODE_READER_ONLY {
	                   multiAttach = true
	           }
	           if reqCap.GetAccessMode().GetMode() == csi.VolumeCapability_AccessMode_MULTI_NODE_SINGLE_WRITER {
	                   multiAttach = true
	           }
	           if reqCap.GetAccessMode().GetMode() == csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER {
	                   multiAttach = true
	           }
	   }

	*/

	// Perform Volume create using the characteristics defined above. Ensure you collect a unique ID of the volume.
	ID := "volume id collected"

	return &csi.CreateVolumeResponse{
		// 返回 volume
		Volume: &csi.Volume{
			VolumeId:      ID,
			CapacityBytes: 0,
			VolumeContext: parameters,
		},
	}, nil
}

// DeleteVolume 删除volume
func (cs *ControllerServer) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {

	glog.V(3).Infof("delete volume req: %v", req)

	volumeID := req.GetVolumeId()

	if volumeID == "" {
		return nil, status.Error(codes.InvalidArgument, "VolumeID not present")
	}

	// Perform volume deletion using the volume ID

	return &csi.DeleteVolumeResponse{}, nil
}

// GetCapacity volume 的 capability
func (cs *ControllerServer) GetCapacity(ctx context.Context, req *csi.GetCapacityRequest) (*csi.GetCapacityResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// ListVolumes list volume
func (cs *ControllerServer) ListVolumes(ctx context.Context, req *csi.ListVolumesRequest) (*csi.ListVolumesResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// 次要 ControllerGetVolume csi-controller 获取 volume
func (cs *ControllerServer) ControllerGetVolume(ctx context.Context, req *csi.ControllerGetVolumeRequest) (*csi.ControllerGetVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// ValidateVolumeCapabilities 校验 volume 的 capability(例如：是否可以同时用于多个节点的读/写)
func (cs *ControllerServer) ValidateVolumeCapabilities(ctx context.Context, req *csi.ValidateVolumeCapabilitiesRequest) (*csi.ValidateVolumeCapabilitiesResponse, error) {

	volumeID := req.GetVolumeId()

	if volumeID == "" {
		return nil, status.Error(codes.InvalidArgument, "VolumeID not present")
	}

	if len(req.VolumeCapabilities) == 0 {
		return nil, status.Error(codes.InvalidArgument, "No volume capability specified")
	}

	for _, cap := range req.VolumeCapabilities {
		if cap.GetAccessMode().GetMode() != csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER {
			return &csi.ValidateVolumeCapabilitiesResponse{Message: ""}, nil
		}
	}
	return &csi.ValidateVolumeCapabilitiesResponse{
		Confirmed: &csi.ValidateVolumeCapabilitiesResponse_Confirmed{
			VolumeCapabilities: req.VolumeCapabilities,
		},
	}, nil
}

// 次要 GetVolumeSizeInBytes volume size
func (cs *ControllerServer) GetVolumeSizeInBytes(req *csi.CreateVolumeRequest) (int64, error) {

	cap := req.GetCapacityRange()
	return cap.GetRequiredBytes(), nil
}
