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

// A binary that is capable of running a complete, standalone kubernetes cluster.
// Expects an etcd server is available, or on the path somewhere.
// Does *not* currently setup the Kubernetes network model, that must be done ahead of time.
// TODO: Setup the k8s network bridge as part of setup.
package main

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/api/resource"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/api/testapi"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/apiserver"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/client"
	nodeControllerPkg "github.com/GoogleCloudPlatform/kubernetes/pkg/cloudprovider/controller"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/controller"
	kubeletServer "github.com/GoogleCloudPlatform/kubernetes/pkg/kubelet/server"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/master"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/master/ports"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/service"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/tools"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/util"
	"github.com/GoogleCloudPlatform/kubernetes/plugin/pkg/scheduler"
	_ "github.com/GoogleCloudPlatform/kubernetes/plugin/pkg/scheduler/algorithmprovider"
	"github.com/GoogleCloudPlatform/kubernetes/plugin/pkg/scheduler/factory"

	"github.com/golang/glog"
	flag "github.com/spf13/pflag"
)

var (
	addr           = flag.String("addr", "127.0.0.1", "The address to use for the apiserver.")
	port           = flag.Int("port", 8080, "The port for the apiserver to use.")
	dockerEndpoint = flag.String("docker_endpoint", "", "If non-empty, use this for the docker endpoint to communicate with")
	etcdServer     = flag.String("etcd_server", "http://localhost:4001", "If non-empty, path to the set of etcd server to use")
	// TODO: Discover these by pinging the host machines, and rip out these flags.
	nodeMilliCPU           = flag.Int64("node_milli_cpu", 1000, "The amount of MilliCPU provisioned on each node")
	nodeMemory             = flag.Int64("node_memory", 3*1024*1024*1024, "The amount of memory (in bytes) provisioned on each node")
	masterServiceNamespace = flag.String("master_service_namespace", api.NamespaceDefault, "The namespace from which the kubernetes master services should be injected into pods")
)

type delegateHandler struct {
	delegate http.Handler
}

func (h *delegateHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if h.delegate != nil {
		h.delegate.ServeHTTP(w, req)
		return
	}
	w.WriteHeader(http.StatusNotFound)
}

// RunApiServer starts an API server in a go routine.
func runApiServer(cl *client.Client, etcdClient tools.EtcdClient, addr net.IP, port int, masterServiceNamespace string) {
	handler := delegateHandler{}

	helper, err := master.NewEtcdHelper(etcdClient, "")
	if err != nil {
		glog.Fatalf("Unable to get etcd helper: %v", err)
	}

	// Create a master and install handlers into mux.
	m := master.New(&master.Config{
		Client:     cl,
		EtcdHelper: helper,
		KubeletClient: &client.HTTPKubeletClient{
			Client: http.DefaultClient,
			Port:   10250,
		},
		EnableLogsSupport:    false,
		EnableSwaggerSupport: true,
		APIPrefix:            "/api",
		Authorizer:           apiserver.NewAlwaysAllowAuthorizer(),

		ReadWritePort:          port,
		ReadOnlyPort:           port,
		PublicAddress:          addr,
		MasterServiceNamespace: masterServiceNamespace,
	})
	handler.delegate = m.InsecureHandler

	go http.ListenAndServe(fmt.Sprintf("%s:%d", addr, port), &handler)
}

// RunScheduler starts up a scheduler in it's own goroutine
func runScheduler(cl *client.Client) {
	// Scheduler
	schedulerConfigFactory := factory.NewConfigFactory(cl)
	schedulerConfig, err := schedulerConfigFactory.Create()
	if err != nil {
		glog.Fatalf("Couldn't create scheduler config: %v", err)
	}
	scheduler.New(schedulerConfig).Run()
}

// RunControllerManager starts a controller
func runControllerManager(machineList []string, cl *client.Client, nodeMilliCPU, nodeMemory int64) {
	nodeResources := &api.NodeResources{
		Capacity: api.ResourceList{
			api.ResourceCPU:    *resource.NewMilliQuantity(nodeMilliCPU, resource.DecimalSI),
			api.ResourceMemory: *resource.NewQuantity(nodeMemory, resource.BinarySI),
		},
	}
	kubeClient := &client.HTTPKubeletClient{Client: http.DefaultClient, Port: ports.KubeletPort}
	nodeController := nodeControllerPkg.NewNodeController(nil, "", machineList, nodeResources, cl, kubeClient)
	nodeController.Run(10*time.Second, 10)

	endpoints := service.NewEndpointController(cl)
	go util.Forever(func() { endpoints.SyncServiceEndpoints() }, time.Second*10)

	controllerManager := controller.NewReplicationManager(cl)
	controllerManager.Run(10 * time.Second)
}

func startComponents(etcdClient tools.EtcdClient, cl *client.Client, addr net.IP, port int) {
	machineList := []string{"localhost"}

	runApiServer(cl, etcdClient, addr, port, *masterServiceNamespace)
	runScheduler(cl)
	runControllerManager(machineList, cl, *nodeMilliCPU, *nodeMemory)

	dockerClient := util.ConnectToDockerOrDie(*dockerEndpoint)
	kubeletServer.SimpleRunKubelet(cl, nil, dockerClient, machineList[0], "/tmp/kubernetes", "", "127.0.0.1", 10250, *masterServiceNamespace, kubeletServer.ProbeVolumePlugins())
}

func newApiClient(addr net.IP, port int) *client.Client {
	apiServerURL := fmt.Sprintf("http://%s:%d", addr, port)
	cl := client.NewOrDie(&client.Config{Host: apiServerURL, Version: testapi.Version()})
	return cl
}

func main() {
	util.InitFlags()
	util.InitLogs()
	defer util.FlushLogs()

	glog.Infof("Creating etcd client pointing to %v", *etcdServer)
	etcdClient, err := tools.NewEtcdClientStartServerIfNecessary(*etcdServer)
	if err != nil {
		glog.Fatalf("Failed to connect to etcd: %v", err)
	}
	address := net.ParseIP(*addr)
	startComponents(etcdClient, newApiClient(address, *port), address, *port)
	glog.Infof("Kubernetes API Server is up and running on http://%s:%d", *addr, *port)

	select {}
}
