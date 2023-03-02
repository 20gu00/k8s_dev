/*
Copyright 2014 Google Inc. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package gce_pd

import (
	"os"
	"testing"

	"github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/kubelet/volume"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/types"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/util/mount"
)

func TestCanSupport(t *testing.T) {
	plugMgr := volume.PluginMgr{}
	plugMgr.InitPlugins(ProbeVolumePlugins(), &volume.FakeHost{"/tmp/fake"})

	plug, err := plugMgr.FindPluginByName("kubernetes.io/gce-pd")
	if err != nil {
		t.Errorf("Can't find the plugin by name")
	}
	if plug.Name() != "kubernetes.io/gce-pd" {
		t.Errorf("Wrong name: %s", plug.Name())
	}
	if !plug.CanSupport(&api.Volume{Source: api.VolumeSource{GCEPersistentDisk: &api.GCEPersistentDisk{}}}) {
		t.Errorf("Expected true")
	}
}

type fakePDManager struct{}

// TODO(jonesdl) To fully test this, we could create a loopback device
// and mount that instead.
func (fake *fakePDManager) AttachDisk(pd *gcePersistentDisk) error {
	globalPath := makeGlobalPDName(pd.plugin.host, pd.pdName, pd.readOnly)
	err := os.MkdirAll(globalPath, 0750)
	if err != nil {
		return err
	}
	return nil
}

func (fake *fakePDManager) DetachDisk(pd *gcePersistentDisk, devicePath string) error {
	globalPath := makeGlobalPDName(pd.plugin.host, pd.pdName, pd.readOnly)
	err := os.RemoveAll(globalPath)
	if err != nil {
		return err
	}
	return nil
}

type fakeMounter struct{}

func (fake *fakeMounter) Mount(source string, target string, fstype string, flags uintptr, data string) error {
	return nil
}

func (fake *fakeMounter) Unmount(target string, flags int) error {
	return nil
}

func (fake *fakeMounter) List() ([]mount.MountPoint, error) {
	return []mount.MountPoint{}, nil
}

func TestPlugin(t *testing.T) {
	plugMgr := volume.PluginMgr{}
	plugMgr.InitPlugins(ProbeVolumePlugins(), &volume.FakeHost{"/tmp/fake"})

	plug, err := plugMgr.FindPluginByName("kubernetes.io/gce-pd")
	if err != nil {
		t.Errorf("Can't find the plugin by name")
	}
	spec := &api.Volume{
		Name: "vol1",
		Source: api.VolumeSource{
			GCEPersistentDisk: &api.GCEPersistentDisk{
				PDName: "pd",
				FSType: "ext4",
			},
		},
	}
	builder, err := plug.(*gcePersistentDiskPlugin).newBuilderInternal(spec, types.UID("poduid"), &fakePDManager{}, &fakeMounter{})
	if err != nil {
		t.Errorf("Failed to make a new Builder: %v", err)
	}
	if builder == nil {
		t.Errorf("Got a nil Builder: %v")
	}

	path := builder.GetPath()
	if path != "/tmp/fake/pods/poduid/volumes/kubernetes.io~gce-pd/vol1" {
		t.Errorf("Got unexpected path: %s", path)
	}

	if err := builder.SetUp(); err != nil {
		t.Errorf("Expected success, got: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			t.Errorf("SetUp() failed, volume path not created: %s", path)
		} else {
			t.Errorf("SetUp() failed: %v", err)
		}
	}
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			t.Errorf("SetUp() failed, volume path not created: %s", path)
		} else {
			t.Errorf("SetUp() failed: %v", err)
		}
	}

	cleaner, err := plug.(*gcePersistentDiskPlugin).newCleanerInternal("vol1", types.UID("poduid"), &fakePDManager{}, &fakeMounter{})
	if err != nil {
		t.Errorf("Failed to make a new Cleaner: %v", err)
	}
	if cleaner == nil {
		t.Errorf("Got a nil Cleaner: %v")
	}

	if err := cleaner.TearDown(); err != nil {
		t.Errorf("Expected success, got: %v", err)
	}
	if _, err := os.Stat(path); err == nil {
		t.Errorf("TearDown() failed, volume path still exists: %s", path)
	} else if !os.IsNotExist(err) {
		t.Errorf("SetUp() failed: %v", err)
	}
}

func TestPluginLegacy(t *testing.T) {
	plugMgr := volume.PluginMgr{}
	plugMgr.InitPlugins(ProbeVolumePlugins(), &volume.FakeHost{"/tmp/fake"})

	plug, err := plugMgr.FindPluginByName("gce-pd")
	if err != nil {
		t.Errorf("Can't find the plugin by name")
	}
	if plug.Name() != "gce-pd" {
		t.Errorf("Wrong name: %s", plug.Name())
	}
	if plug.CanSupport(&api.Volume{Source: api.VolumeSource{GCEPersistentDisk: &api.GCEPersistentDisk{}}}) {
		t.Errorf("Expected false")
	}

	if _, err := plug.NewBuilder(&api.Volume{Source: api.VolumeSource{GCEPersistentDisk: &api.GCEPersistentDisk{}}}, types.UID("poduid")); err == nil {
		t.Errorf("Expected failiure")
	}

	cleaner, err := plug.NewCleaner("vol1", types.UID("poduid"))
	if err != nil {
		t.Errorf("Failed to make a new Cleaner: %v", err)
	}
	if cleaner == nil {
		t.Errorf("Got a nil Cleaner: %v")
	}
}
