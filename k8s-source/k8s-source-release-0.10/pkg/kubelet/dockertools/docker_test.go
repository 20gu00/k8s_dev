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

package dockertools

import (
	"fmt"
	"hash/adler32"
	"reflect"
	"testing"

	"github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/credentialprovider"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/types"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/util"
	docker "github.com/fsouza/go-dockerclient"
)

func verifyCalls(t *testing.T, fakeDocker *FakeDockerClient, calls []string) {
	fakeDocker.Lock()
	defer fakeDocker.Unlock()
	verifyStringArrayEquals(t, fakeDocker.called, calls)
}

func verifyStringArrayEquals(t *testing.T, actual, expected []string) {
	invalid := len(actual) != len(expected)
	if !invalid {
		for ix, value := range actual {
			if expected[ix] != value {
				invalid = true
			}
		}
	}
	if invalid {
		t.Errorf("Expected: %#v, Actual: %#v", expected, actual)
	}
}

func TestGetContainerID(t *testing.T) {
	fakeDocker := &FakeDockerClient{}
	fakeDocker.ContainerList = []docker.APIContainers{
		{
			ID:    "foobar",
			Names: []string{"/k8s_foo_qux_1234_42"},
		},
		{
			ID:    "barbar",
			Names: []string{"/k8s_bar_qux_2565_42"},
		},
	}
	fakeDocker.Container = &docker.Container{
		ID: "foobar",
	}

	dockerContainers, err := GetKubeletDockerContainers(fakeDocker, false)
	if err != nil {
		t.Errorf("Expected no error, Got %#v", err)
	}
	if len(dockerContainers) != 2 {
		t.Errorf("Expected %#v, Got %#v", fakeDocker.ContainerList, dockerContainers)
	}
	verifyCalls(t, fakeDocker, []string{"list"})
	dockerContainer, found, _ := dockerContainers.FindPodContainer("qux", "", "foo")
	if dockerContainer == nil || !found {
		t.Errorf("Failed to find container %#v", dockerContainer)
	}

	fakeDocker.ClearCalls()
	dockerContainer, found, _ = dockerContainers.FindPodContainer("foobar", "", "foo")
	verifyCalls(t, fakeDocker, []string{})
	if dockerContainer != nil || found {
		t.Errorf("Should not have found container %#v", dockerContainer)
	}
}

func verifyPackUnpack(t *testing.T, podNamespace, podUID, podName, containerName string) {
	container := &api.Container{Name: containerName}
	hasher := adler32.New()
	util.DeepHashObject(hasher, *container)
	computedHash := uint64(hasher.Sum32())
	podFullName := fmt.Sprintf("%s.%s", podName, podNamespace)
	name := BuildDockerName(types.UID(podUID), podFullName, container)
	returnedPodFullName, returnedUID, returnedContainerName, hash := ParseDockerName(name)
	if podFullName != returnedPodFullName || podUID != string(returnedUID) || containerName != returnedContainerName || computedHash != hash {
		t.Errorf("For (%s, %s, %s, %d), unpacked (%s, %s, %s, %d)", podFullName, podUID, containerName, computedHash, returnedPodFullName, returnedUID, returnedContainerName, hash)
	}
}

func TestContainerManifestNaming(t *testing.T) {
	podUID := "12345678"
	verifyPackUnpack(t, "file", podUID, "name", "container")
	verifyPackUnpack(t, "file", podUID, "name-with-dashes", "container")
	// UID is same as pod name
	verifyPackUnpack(t, "file", podUID, podUID, "container")
	// No Container name
	verifyPackUnpack(t, "other", podUID, "name", "")

	container := &api.Container{Name: "container"}
	podName := "foo"
	podNamespace := "test"
	name := fmt.Sprintf("k8s_%s_%s.%s_%s_42", container.Name, podName, podNamespace, podUID)
	podFullName := fmt.Sprintf("%s.%s", podName, podNamespace)

	returnedPodFullName, returnedPodUID, returnedContainerName, hash := ParseDockerName(name)
	if returnedPodFullName != podFullName || string(returnedPodUID) != podUID || returnedContainerName != container.Name || hash != 0 {
		t.Errorf("unexpected parse: %s %s %s %d", returnedPodFullName, returnedPodUID, returnedContainerName, hash)
	}
}

func TestGetDockerServerVersion(t *testing.T) {
	fakeDocker := &FakeDockerClient{VersionInfo: docker.Env{"Client version=1.2", "Server version=1.1.3", "Server API version=1.15"}}
	runner := dockerContainerCommandRunner{fakeDocker}
	version, err := runner.getDockerServerVersion()
	if err != nil {
		t.Errorf("got error while getting docker server version - %s", err)
	}
	expectedVersion := []uint{1, 15}
	if len(expectedVersion) != len(version) {
		t.Errorf("invalid docker server version. expected: %v, got: %v", expectedVersion, version)
	} else {
		for idx, val := range expectedVersion {
			if version[idx] != val {
				t.Errorf("invalid docker server version. expected: %v, got: %v", expectedVersion, version)
			}
		}
	}
}

func TestExecSupportExists(t *testing.T) {
	fakeDocker := &FakeDockerClient{VersionInfo: docker.Env{"Client version=1.2", "Server version=1.3.0", "Server API version=1.15"}}
	runner := dockerContainerCommandRunner{fakeDocker}
	useNativeExec, err := runner.nativeExecSupportExists()
	if err != nil {
		t.Errorf("got error while checking for exec support - %s", err)
	}
	if !useNativeExec {
		t.Errorf("invalid exec support check output. Expected true")
	}
}

func TestExecSupportNotExists(t *testing.T) {
	fakeDocker := &FakeDockerClient{VersionInfo: docker.Env{"Client version=1.2", "Server version=1.1.2", "Server API version=1.14"}}
	runner := dockerContainerCommandRunner{fakeDocker}
	useNativeExec, _ := runner.nativeExecSupportExists()
	if useNativeExec {
		t.Errorf("invalid exec support check output.")
	}
}

func TestDockerContainerCommand(t *testing.T) {
	runner := dockerContainerCommandRunner{}
	containerID := "1234"
	command := []string{"ls"}
	cmd, _ := runner.getRunInContainerCommand(containerID, command)
	if cmd.Dir != "/var/lib/docker/execdriver/native/"+containerID {
		t.Errorf("unexpected command CWD: %s", cmd.Dir)
	}
	if !reflect.DeepEqual(cmd.Args, []string{"/usr/sbin/nsinit", "exec", "ls"}) {
		t.Errorf("unexpected command args: %s", cmd.Args)
	}
}

var parseImageNameTests = []struct {
	imageName string
	name      string
	tag       string
}{
	{"ubuntu", "ubuntu", ""},
	{"ubuntu:2342", "ubuntu", "2342"},
	{"ubuntu:latest", "ubuntu", "latest"},
	{"foo/bar:445566", "foo/bar", "445566"},
	{"registry.example.com:5000/foobar", "registry.example.com:5000/foobar", ""},
	{"registry.example.com:5000/foobar:5342", "registry.example.com:5000/foobar", "5342"},
	{"registry.example.com:5000/foobar:latest", "registry.example.com:5000/foobar", "latest"},
}

func TestParseImageName(t *testing.T) {
	for _, tt := range parseImageNameTests {
		name, tag := parseImageName(tt.imageName)
		if name != tt.name || tag != tt.tag {
			t.Errorf("Expected name/tag: %s/%s, got %s/%s", tt.name, tt.tag, name, tag)
		}
	}
}

func TestDockerKeyringLookup(t *testing.T) {
	empty := docker.AuthConfiguration{}

	ada := docker.AuthConfiguration{
		Username: "ada",
		Password: "smash",
		Email:    "ada@example.com",
	}

	grace := docker.AuthConfiguration{
		Username: "grace",
		Password: "squash",
		Email:    "grace@example.com",
	}

	dk := &credentialprovider.BasicDockerKeyring{}
	dk.Add(credentialprovider.DockerConfig{
		"bar.example.com/pong": credentialprovider.DockerConfigEntry{
			Username: grace.Username,
			Password: grace.Password,
			Email:    grace.Email,
		},
		"bar.example.com": credentialprovider.DockerConfigEntry{
			Username: ada.Username,
			Password: ada.Password,
			Email:    ada.Email,
		},
	})

	tests := []struct {
		image string
		match docker.AuthConfiguration
		ok    bool
	}{
		// direct match
		{"bar.example.com", ada, true},

		// direct match deeper than other possible matches
		{"bar.example.com/pong", grace, true},

		// no direct match, deeper path ignored
		{"bar.example.com/ping", ada, true},

		// match first part of path token
		{"bar.example.com/pongz", grace, true},

		// match regardless of sub-path
		{"bar.example.com/pong/pang", grace, true},

		// no host match
		{"example.com", empty, false},
		{"foo.example.com", empty, false},
	}

	for i, tt := range tests {
		match, ok := dk.Lookup(tt.image)
		if tt.ok != ok {
			t.Errorf("case %d: expected ok=%t, got %t", i, tt.ok, ok)
		}

		if !reflect.DeepEqual(tt.match, match) {
			t.Errorf("case %d: expected match=%#v, got %#v", i, tt.match, match)
		}
	}
}

// This validates that dockercfg entries with a scheme and url path are properly matched
// by images that only match the hostname.
// NOTE: the above covers the case of a more specific match trumping just hostname.
func TestIssue3797(t *testing.T) {
	rex := docker.AuthConfiguration{
		Username: "rex",
		Password: "tiny arms",
		Email:    "rex@example.com",
	}

	dk := &credentialprovider.BasicDockerKeyring{}
	dk.Add(credentialprovider.DockerConfig{
		"https://quay.io/v1/": credentialprovider.DockerConfigEntry{
			Username: rex.Username,
			Password: rex.Password,
			Email:    rex.Email,
		},
	})

	tests := []struct {
		image string
		match docker.AuthConfiguration
		ok    bool
	}{
		// direct match
		{"quay.io", rex, true},

		// partial matches
		{"quay.io/foo", rex, true},
		{"quay.io/foo/bar", rex, true},
	}

	for i, tt := range tests {
		match, ok := dk.Lookup(tt.image)
		if tt.ok != ok {
			t.Errorf("case %d: expected ok=%t, got %t", i, tt.ok, ok)
		}

		if !reflect.DeepEqual(tt.match, match) {
			t.Errorf("case %d: expected match=%#v, got %#v", i, tt.match, match)
		}
	}
}

type imageTrackingDockerClient struct {
	*FakeDockerClient
	imageName string
}

func (f *imageTrackingDockerClient) InspectImage(name string) (image *docker.Image, err error) {
	image, err = f.FakeDockerClient.InspectImage(name)
	f.imageName = name
	return
}

func TestIsImagePresent(t *testing.T) {
	cl := &imageTrackingDockerClient{&FakeDockerClient{}, ""}
	puller := &dockerPuller{
		client: cl,
	}
	_, _ = puller.IsImagePresent("abc:123")
	if cl.imageName != "abc:123" {
		t.Errorf("expected inspection of image abc:123, instead inspected image %v", cl.imageName)
	}
}

func TestGetRunningContainers(t *testing.T) {
	fakeDocker := &FakeDockerClient{}
	tests := []struct {
		containers  map[string]*docker.Container
		inputIDs    []string
		expectedIDs []string
		err         error
	}{
		{
			containers: map[string]*docker.Container{
				"foobar": {
					ID: "foobar",
					State: docker.State{
						Running: false,
					},
				},
				"baz": {
					ID: "baz",
					State: docker.State{
						Running: true,
					},
				},
			},
			inputIDs:    []string{"foobar", "baz"},
			expectedIDs: []string{"baz"},
		},
		{
			containers: map[string]*docker.Container{
				"foobar": {
					ID: "foobar",
					State: docker.State{
						Running: true,
					},
				},
				"baz": {
					ID: "baz",
					State: docker.State{
						Running: true,
					},
				},
			},
			inputIDs:    []string{"foobar", "baz"},
			expectedIDs: []string{"foobar", "baz"},
		},
		{
			containers: map[string]*docker.Container{
				"foobar": {
					ID: "foobar",
					State: docker.State{
						Running: false,
					},
				},
				"baz": {
					ID: "baz",
					State: docker.State{
						Running: false,
					},
				},
			},
			inputIDs:    []string{"foobar", "baz"},
			expectedIDs: []string{},
		},
		{
			containers: map[string]*docker.Container{
				"foobar": {
					ID: "foobar",
					State: docker.State{
						Running: false,
					},
				},
				"baz": {
					ID: "baz",
					State: docker.State{
						Running: false,
					},
				},
			},
			inputIDs: []string{"foobar", "baz"},
			err:      fmt.Errorf("test error"),
		},
	}
	for _, test := range tests {
		fakeDocker.ContainerMap = test.containers
		fakeDocker.Err = test.err
		if results, err := GetRunningContainers(fakeDocker, test.inputIDs); err == nil {
			resultIDs := []string{}
			for _, result := range results {
				resultIDs = append(resultIDs, result.ID)
			}
			if !reflect.DeepEqual(resultIDs, test.expectedIDs) {
				t.Errorf("expected: %v, saw: %v", test.expectedIDs, resultIDs)
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		} else {
			if err != test.err {
				t.Errorf("unexpected error: %v", err)
			}
		}
	}
}
