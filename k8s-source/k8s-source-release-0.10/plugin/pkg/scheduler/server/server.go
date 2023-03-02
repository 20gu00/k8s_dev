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

// Package server implements a Server object for running the scheduler.
package server

import (
	"net"
	"net/http"
	"strconv"

	"github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/client"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/client/record"
	_ "github.com/GoogleCloudPlatform/kubernetes/pkg/healthz"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/hyperkube"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/master/ports"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/util"
	"github.com/GoogleCloudPlatform/kubernetes/plugin/pkg/scheduler"
	_ "github.com/GoogleCloudPlatform/kubernetes/plugin/pkg/scheduler/algorithmprovider"
	"github.com/GoogleCloudPlatform/kubernetes/plugin/pkg/scheduler/factory"

	"github.com/golang/glog"
	"github.com/spf13/pflag"
)

// SchedulerServer has all the context and params needed to run a Scheduler
type SchedulerServer struct {
	Port              int
	Address           util.IP
	ClientConfig      client.Config
	AlgorithmProvider string
}

// NewSchedulerServer creates a new SchedulerServer with default parameters
func NewSchedulerServer() *SchedulerServer {
	s := SchedulerServer{
		Port:              ports.SchedulerPort,
		Address:           util.IP(net.ParseIP("127.0.0.1")),
		AlgorithmProvider: factory.DefaultProvider,
	}
	return &s
}

// NewHyperkubeServer creates a new hyperkube Server object that includes the
// description and flags.
func NewHyperkubeServer() *hyperkube.Server {
	s := NewSchedulerServer()

	hks := hyperkube.Server{
		SimpleUsage: "scheduler",
		Long:        "Implements a Kubernetes scheduler.  This will assign pods to kubelets based on capacity and constraints.",
		Run: func(_ *hyperkube.Server, args []string) error {
			return s.Run(args)
		},
	}
	s.AddFlags(hks.Flags())
	return &hks
}

// AddFlags adds flags for a specific SchedulerServer to the specified FlagSet
func (s *SchedulerServer) AddFlags(fs *pflag.FlagSet) {
	fs.IntVar(&s.Port, "port", s.Port, "The port that the scheduler's http service runs on")
	fs.Var(&s.Address, "address", "The IP address to serve on (set to 0.0.0.0 for all interfaces)")
	client.BindClientConfigFlags(fs, &s.ClientConfig)
	fs.StringVar(&s.AlgorithmProvider, "algorithm_provider", s.AlgorithmProvider, "The scheduling algorithm provider to use")
}

// Run runs the specified SchedulerServer.  This should never exit.
func (s *SchedulerServer) Run(_ []string) error {
	kubeClient, err := client.New(&s.ClientConfig)
	if err != nil {
		glog.Fatalf("Invalid API configuration: %v", err)
	}

	record.StartRecording(kubeClient.Events(""), api.EventSource{Component: "scheduler"})

	go http.ListenAndServe(net.JoinHostPort(s.Address.String(), strconv.Itoa(s.Port)), nil)

	configFactory := factory.NewConfigFactory(kubeClient)
	config, err := s.createConfig(configFactory)
	if err != nil {
		glog.Fatalf("Failed to create scheduler configuration: %v", err)
	}
	sched := scheduler.New(config)
	sched.Run()

	select {}
}

func (s *SchedulerServer) createConfig(configFactory *factory.ConfigFactory) (*scheduler.Config, error) {
	// check of algorithm provider is registered and fail fast
	_, err := factory.GetAlgorithmProvider(s.AlgorithmProvider)
	if err != nil {
		return nil, err
	}
	return configFactory.CreateFromProvider(s.AlgorithmProvider)
}
