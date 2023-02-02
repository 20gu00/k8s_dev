package mycsi

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/golang/glog"
	"golang.org/x/net/context"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/nightlyone/lockfile"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type NodeServer struct {
	Driver *MyCSIDriver
	// TODO: Only lock mutually exclusive calls and make locking more fine grained
	mux sync.Mutex
}

func (ns *NodeServer) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	glog.V(3).Infof("nodeserver NodePublishVolume")
	glog.V(4).Infof("NodePublishVolume called with req: %#v", req)

	// Validate Arguments
	volumeID := req.GetVolumeId()
	targetPath := req.GetTargetPath()
	stagingTargetPath := req.GetStagingTargetPath()
	volumeCapability := req.GetVolumeCapability()
	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "NodeStageVolume Volume ID must be provided")
	}
	if len(targetPath) == 0 {
		return nil, status.Error(codes.InvalidArgument, "NodeStageVolume Target Path must be provided")
	}
	if len(stagingTargetPath) == 0 {
		return nil, status.Error(codes.InvalidArgument, "NodeStageVolume Staging Target Path must be provided")
	}
	if volumeCapability == nil {
		return nil, status.Error(codes.InvalidArgument, "NodeStageVolume Volume Capability must be provided")
	}

	// Perform the following:
	// Create directory targetPath
	// mount --bind -r stagingTargetPath targetPath

	return &csi.NodePublishVolumeResponse{}, nil
}

func (ns *NodeServer) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	glog.V(3).Infof("nodeserver NodeUnpublishVolume")
	glog.V(4).Infof("NodeUnpublishVolume called with args: %v", req)

	// Validate Arguments
	volumeID := req.GetVolumeId()
	targetPath := req.GetTargetPath()
	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "NodeStageVolume Volume ID must be provided")
	}
	if len(targetPath) == 0 {
		return nil, status.Error(codes.InvalidArgument, "NodeStageVolume Staging Target Path must be provided")
	}

	// Perform the following:
	// unmount targetPath

	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (ns *NodeServer) NodeStageVolume(ctx context.Context, req *csi.NodeStageVolumeRequest) (
	*csi.NodeStageVolumeResponse, error) {
	glog.V(3).Infof("nodeserver NodeStageVolume %#v", req)

	// Validate Arguments
	volumeID := req.GetVolumeId()
	stagingTargetPath := req.GetStagingTargetPath()
	volumeCapability := req.GetVolumeCapability()
	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "NodeStageVolume Volume ID must be provided")
	}
	if len(stagingTargetPath) == 0 {
		return nil, status.Error(codes.InvalidArgument, "NodeStageVolume Staging Target Path must be provided")
	}
	if volumeCapability == nil {
		return nil, status.Error(codes.InvalidArgument, "NodeStageVolume Volume Capability must be provided")
	}

	fsType := volumeCapability.GetMount().FsType
	glog.V(3).Infof("nodeserver NodeStageVolume Required Filesystem Type : %s", fsType)

	var pID = os.Getpid()
	lock, err := lockfile.New(filepath.Join(os.TempDir(), "my-scsiscan.lck"))
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, (fmt.Sprintf("%d : Cannot init lock. Reason. %v", pID, err)))
	}

	for i := 0; i < 30; i++ {
		err = lock.TryLock()
		if err == nil {
			break
		}
		glog.V(3).Infof("%d : Could not get lock, error is %v. Sleeping for 5 seconds", pID, err)
		time.Sleep(5 * time.Second)
	}
	glog.V(3).Infof("%d : Got hold of Scsiscan lock", pID)

	defer lock.Unlock()

	// Perform the following:
	// Find where the attached volume exist on the worker node. Call it volDevicePath.
	// See if volDevicePath has any file system created on it. If not, Do mkfsX on volDevicePath with fsType
	// Create a directory stagingTargetPath
	// mount volDevicePath stagingTargetPath

	return &csi.NodeStageVolumeResponse{}, nil
}

func (ns *NodeServer) NodeUnstageVolume(ctx context.Context, req *csi.NodeUnstageVolumeRequest) (
	*csi.NodeUnstageVolumeResponse, error) {
	glog.V(3).Infof("nodeserver NodeUnstageVolume")

	// Validate Arguments
	volumeID := req.GetVolumeId()
	stagingTargetPath := req.GetStagingTargetPath()
	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "NodeStageVolume Volume ID must be provided")
	}
	if len(stagingTargetPath) == 0 {
		return nil, status.Error(codes.InvalidArgument, "NodeStageVolume Staging Target Path must be provided")
	}

	// Perform the following:
	// Get the 'device mounted path' for stagingTargetPath:
	//    1. run mount -w and grep for stagingTargetPath
	//    2. first field in the output is 'device mounted path'
	// Get all blockdevices of the 'device mounted path':
	//    1. if device mounted path is /dev/mapper/mpath then:
	//        a. multipath -l 'device mounted path'
	//        b. get parent device from output
	//        c. get slave device for parent device:
	//           1. split parent device into sub strings with '/' as separator.
	//           2. if number of fields is not 3 or second field has no 'dev' as prefix, return empty list as blockdevices.
	//           3. else get third field from split, look for /sys/block/<third field>/slaves/, read it as directory and return files under the direcotry as blockdevices
	//    2. if 'device mounted path' is not a multipath, return 'device mounted path' as block devices list of size 1.
	// unmount stagingTargetPath
	// Delete all blockdevices
	// If applicable, remove all multiple path devices for 'device mounted path'

	return &csi.NodeUnstageVolumeResponse{}, nil
}

func (ns *NodeServer) NodeGetCapabilities(ctx context.Context, req *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	glog.V(4).Infof("NodeGetCapabilities called with req: %#v", req)
	return &csi.NodeGetCapabilitiesResponse{
		Capabilities: ns.Driver.nscap,
	}, nil
}

func (ns *NodeServer) NodeGetInfo(ctx context.Context, req *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	glog.V(4).Infof("NodeGetInfo called with req: %#v", req)
	return &csi.NodeGetInfoResponse{
		NodeId: ns.Driver.nodeID,
	}, nil
}

func (ns *NodeServer) NodeExpandVolume(ctx context.Context, req *csi.NodeExpandVolumeRequest) (*csi.NodeExpandVolumeResponse, error) {

	volumeID := req.GetVolumeId()
	if volumeID == "" {
		return nil, status.Error(codes.InvalidArgument, "VolumeID is not present")
	}

	// newSize := int(req.GetCapacityRange().GetRequiredBytes() / GiB)

	return &csi.NodeExpandVolumeResponse{}, nil

}

func (ns *NodeServer) NodeGetVolumeStats(ctx context.Context, req *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}
