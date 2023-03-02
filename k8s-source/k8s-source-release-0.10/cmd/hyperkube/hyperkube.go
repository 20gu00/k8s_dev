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

// A binary that can morph into all of the other kubernetes binaries. You can
// also soft-link to it busybox style.
package main

import (
	"os"

	"github.com/GoogleCloudPlatform/kubernetes/pkg/controllermanager"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/hyperkube"
	kubelet "github.com/GoogleCloudPlatform/kubernetes/pkg/kubelet/server"
	apiserver "github.com/GoogleCloudPlatform/kubernetes/pkg/master/server"
	proxy "github.com/GoogleCloudPlatform/kubernetes/pkg/proxy/server"
	sched "github.com/GoogleCloudPlatform/kubernetes/plugin/pkg/scheduler/server"
)

func main() {
	hk := hyperkube.HyperKube{
		Name: "hyperkube",
		Long: "This is an all-in-one binary that can run any of the various Kubernetes servers.",
	}

	hk.AddServer(apiserver.NewHyperkubeServer())
	hk.AddServer(controllermanager.NewHyperkubeServer())
	hk.AddServer(sched.NewHyperkubeServer())
	hk.AddServer(kubelet.NewHyperkubeServer())
	hk.AddServer(proxy.NewHyperkubeServer())

	hk.RunToExit(os.Args)
}
