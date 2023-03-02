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

package proxy

import (
	"net"
	"testing"

	"github.com/GoogleCloudPlatform/kubernetes/pkg/api"
)

func TestValidateWorks(t *testing.T) {
	if isValidEndpoint("") {
		t.Errorf("Didn't fail for empty string")
	}
	if isValidEndpoint("foobar") {
		t.Errorf("Didn't fail with no port")
	}
	if isValidEndpoint("foobar:-1") {
		t.Errorf("Didn't fail with a negative port")
	}
	if !isValidEndpoint("foobar:8080") {
		t.Errorf("Failed a valid config.")
	}
}

func TestFilterWorks(t *testing.T) {
	endpoints := []string{"foobar:1", "foobar:2", "foobar:-1", "foobar:3", "foobar:-2"}
	filtered := filterValidEndpoints(endpoints)

	if len(filtered) != 3 {
		t.Errorf("Failed to filter to the correct size")
	}
	if filtered[0] != "foobar:1" {
		t.Errorf("Index zero is not foobar:1")
	}
	if filtered[1] != "foobar:2" {
		t.Errorf("Index one is not foobar:2")
	}
	if filtered[2] != "foobar:3" {
		t.Errorf("Index two is not foobar:3")
	}
}

func TestLoadBalanceFailsWithNoEndpoints(t *testing.T) {
	loadBalancer := NewLoadBalancerRR()
	var endpoints []api.Endpoints
	loadBalancer.OnUpdate(endpoints)
	endpoint, err := loadBalancer.NextEndpoint("foo", nil)
	if err == nil {
		t.Errorf("Didn't fail with non-existent service")
	}
	if len(endpoint) != 0 {
		t.Errorf("Got an endpoint")
	}
}

func expectEndpoint(t *testing.T, loadBalancer *LoadBalancerRR, service string, expected string, netaddr net.Addr) {
	endpoint, err := loadBalancer.NextEndpoint(service, netaddr)
	if err != nil {
		t.Errorf("Didn't find a service for %s, expected %s, failed with: %v", service, expected, err)
	}
	if endpoint != expected {
		t.Errorf("Didn't get expected endpoint for service %s client %v, expected %s, got: %s", service, netaddr, expected, endpoint)
	}
}

func TestLoadBalanceWorksWithSingleEndpoint(t *testing.T) {
	loadBalancer := NewLoadBalancerRR()
	endpoint, err := loadBalancer.NextEndpoint("foo", nil)
	if err == nil || len(endpoint) != 0 {
		t.Errorf("Didn't fail with non-existent service")
	}
	endpoints := make([]api.Endpoints, 1)
	endpoints[0] = api.Endpoints{
		ObjectMeta: api.ObjectMeta{Name: "foo"},
		Endpoints:  []string{"endpoint1:40"},
	}
	loadBalancer.OnUpdate(endpoints)
	expectEndpoint(t, loadBalancer, "foo", "endpoint1:40", nil)
	expectEndpoint(t, loadBalancer, "foo", "endpoint1:40", nil)
	expectEndpoint(t, loadBalancer, "foo", "endpoint1:40", nil)
	expectEndpoint(t, loadBalancer, "foo", "endpoint1:40", nil)
}

func TestLoadBalanceWorksWithMultipleEndpoints(t *testing.T) {
	loadBalancer := NewLoadBalancerRR()
	endpoint, err := loadBalancer.NextEndpoint("foo", nil)
	if err == nil || len(endpoint) != 0 {
		t.Errorf("Didn't fail with non-existent service")
	}
	endpoints := make([]api.Endpoints, 1)
	endpoints[0] = api.Endpoints{
		ObjectMeta: api.ObjectMeta{Name: "foo"},
		Endpoints:  []string{"endpoint:1", "endpoint:2", "endpoint:3"},
	}
	loadBalancer.OnUpdate(endpoints)
	shuffledEndpoints := loadBalancer.endpointsMap["foo"]
	expectEndpoint(t, loadBalancer, "foo", shuffledEndpoints[0], nil)
	expectEndpoint(t, loadBalancer, "foo", shuffledEndpoints[1], nil)
	expectEndpoint(t, loadBalancer, "foo", shuffledEndpoints[2], nil)
	expectEndpoint(t, loadBalancer, "foo", shuffledEndpoints[0], nil)
}

func TestLoadBalanceWorksWithMultipleEndpointsAndUpdates(t *testing.T) {
	loadBalancer := NewLoadBalancerRR()
	endpoint, err := loadBalancer.NextEndpoint("foo", nil)
	if err == nil || len(endpoint) != 0 {
		t.Errorf("Didn't fail with non-existent service")
	}
	endpoints := make([]api.Endpoints, 1)
	endpoints[0] = api.Endpoints{
		ObjectMeta: api.ObjectMeta{Name: "foo"},
		Endpoints:  []string{"endpoint:1", "endpoint:2", "endpoint:3"},
	}
	loadBalancer.OnUpdate(endpoints)
	shuffledEndpoints := loadBalancer.endpointsMap["foo"]
	expectEndpoint(t, loadBalancer, "foo", shuffledEndpoints[0], nil)
	expectEndpoint(t, loadBalancer, "foo", shuffledEndpoints[1], nil)
	expectEndpoint(t, loadBalancer, "foo", shuffledEndpoints[2], nil)
	expectEndpoint(t, loadBalancer, "foo", shuffledEndpoints[0], nil)
	expectEndpoint(t, loadBalancer, "foo", shuffledEndpoints[1], nil)
	// Then update the configuration with one fewer endpoints, make sure
	// we start in the beginning again
	endpoints[0] = api.Endpoints{ObjectMeta: api.ObjectMeta{Name: "foo"},
		Endpoints: []string{"endpoint:8", "endpoint:9"},
	}
	loadBalancer.OnUpdate(endpoints)
	shuffledEndpoints = loadBalancer.endpointsMap["foo"]
	expectEndpoint(t, loadBalancer, "foo", shuffledEndpoints[0], nil)
	expectEndpoint(t, loadBalancer, "foo", shuffledEndpoints[1], nil)
	expectEndpoint(t, loadBalancer, "foo", shuffledEndpoints[0], nil)
	expectEndpoint(t, loadBalancer, "foo", shuffledEndpoints[1], nil)
	// Clear endpoints
	endpoints[0] = api.Endpoints{ObjectMeta: api.ObjectMeta{Name: "foo"}, Endpoints: []string{}}
	loadBalancer.OnUpdate(endpoints)

	endpoint, err = loadBalancer.NextEndpoint("foo", nil)
	if err == nil || len(endpoint) != 0 {
		t.Errorf("Didn't fail with non-existent service")
	}
}

func TestLoadBalanceWorksWithServiceRemoval(t *testing.T) {
	loadBalancer := NewLoadBalancerRR()
	endpoint, err := loadBalancer.NextEndpoint("foo", nil)
	if err == nil || len(endpoint) != 0 {
		t.Errorf("Didn't fail with non-existent service")
	}
	endpoints := make([]api.Endpoints, 2)
	endpoints[0] = api.Endpoints{
		ObjectMeta: api.ObjectMeta{Name: "foo"},
		Endpoints:  []string{"endpoint:1", "endpoint:2", "endpoint:3"},
	}
	endpoints[1] = api.Endpoints{
		ObjectMeta: api.ObjectMeta{Name: "bar"},
		Endpoints:  []string{"endpoint:4", "endpoint:5"},
	}
	loadBalancer.OnUpdate(endpoints)
	shuffledFooEndpoints := loadBalancer.endpointsMap["foo"]
	expectEndpoint(t, loadBalancer, "foo", shuffledFooEndpoints[0], nil)
	expectEndpoint(t, loadBalancer, "foo", shuffledFooEndpoints[1], nil)
	expectEndpoint(t, loadBalancer, "foo", shuffledFooEndpoints[2], nil)
	expectEndpoint(t, loadBalancer, "foo", shuffledFooEndpoints[0], nil)
	expectEndpoint(t, loadBalancer, "foo", shuffledFooEndpoints[1], nil)

	shuffledBarEndpoints := loadBalancer.endpointsMap["bar"]
	expectEndpoint(t, loadBalancer, "bar", shuffledBarEndpoints[0], nil)
	expectEndpoint(t, loadBalancer, "bar", shuffledBarEndpoints[1], nil)
	expectEndpoint(t, loadBalancer, "bar", shuffledBarEndpoints[0], nil)
	expectEndpoint(t, loadBalancer, "bar", shuffledBarEndpoints[1], nil)
	expectEndpoint(t, loadBalancer, "bar", shuffledBarEndpoints[0], nil)

	// Then update the configuration by removing foo
	loadBalancer.OnUpdate(endpoints[1:])
	endpoint, err = loadBalancer.NextEndpoint("foo", nil)
	if err == nil || len(endpoint) != 0 {
		t.Errorf("Didn't fail with non-existent service")
	}

	// but bar is still there, and we continue RR from where we left off.
	expectEndpoint(t, loadBalancer, "bar", shuffledBarEndpoints[1], nil)
	expectEndpoint(t, loadBalancer, "bar", shuffledBarEndpoints[0], nil)
	expectEndpoint(t, loadBalancer, "bar", shuffledBarEndpoints[1], nil)
	expectEndpoint(t, loadBalancer, "bar", shuffledBarEndpoints[0], nil)
}

func TestStickyLoadBalanceWorksWithSingleEndpoint(t *testing.T) {
	client1 := &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0}
	client2 := &net.TCPAddr{IP: net.IPv4(127, 0, 0, 2), Port: 0}
	loadBalancer := NewLoadBalancerRR()
	endpoint, err := loadBalancer.NextEndpoint("foo", nil)
	if err == nil || len(endpoint) != 0 {
		t.Errorf("Didn't fail with non-existent service")
	}
	loadBalancer.NewService("foo", api.AffinityTypeClientIP, 0)
	endpoints := make([]api.Endpoints, 1)
	endpoints[0] = api.Endpoints{
		ObjectMeta: api.ObjectMeta{Name: "foo"},
		Endpoints:  []string{"endpoint:1"},
	}
	loadBalancer.OnUpdate(endpoints)
	expectEndpoint(t, loadBalancer, "foo", "endpoint:1", client1)
	expectEndpoint(t, loadBalancer, "foo", "endpoint:1", client1)
	expectEndpoint(t, loadBalancer, "foo", "endpoint:1", client2)
	expectEndpoint(t, loadBalancer, "foo", "endpoint:1", client2)
}

func TestStickyLoadBalanaceWorksWithMultipleEndpoints(t *testing.T) {
	client1 := &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0}
	client2 := &net.TCPAddr{IP: net.IPv4(127, 0, 0, 2), Port: 0}
	client3 := &net.TCPAddr{IP: net.IPv4(127, 0, 0, 3), Port: 0}
	loadBalancer := NewLoadBalancerRR()
	endpoint, err := loadBalancer.NextEndpoint("foo", nil)
	if err == nil || len(endpoint) != 0 {
		t.Errorf("Didn't fail with non-existent service")
	}

	loadBalancer.NewService("foo", api.AffinityTypeClientIP, 0)
	endpoints := make([]api.Endpoints, 1)
	endpoints[0] = api.Endpoints{
		ObjectMeta: api.ObjectMeta{Name: "foo"},
		Endpoints:  []string{"endpoint:1", "endpoint:2", "endpoint:3"},
	}
	loadBalancer.OnUpdate(endpoints)
	shuffledEndpoints := loadBalancer.endpointsMap["foo"]
	expectEndpoint(t, loadBalancer, "foo", shuffledEndpoints[0], client1)
	expectEndpoint(t, loadBalancer, "foo", shuffledEndpoints[0], client1)
	expectEndpoint(t, loadBalancer, "foo", shuffledEndpoints[1], client2)
	expectEndpoint(t, loadBalancer, "foo", shuffledEndpoints[1], client2)
	expectEndpoint(t, loadBalancer, "foo", shuffledEndpoints[2], client3)
	expectEndpoint(t, loadBalancer, "foo", shuffledEndpoints[2], client3)
	expectEndpoint(t, loadBalancer, "foo", shuffledEndpoints[0], client1)
	expectEndpoint(t, loadBalancer, "foo", shuffledEndpoints[0], client1)
}

func TestStickyLoadBalanaceWorksWithMultipleEndpointsStickyNone(t *testing.T) {
	client1 := &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0}
	client2 := &net.TCPAddr{IP: net.IPv4(127, 0, 0, 2), Port: 0}
	client3 := &net.TCPAddr{IP: net.IPv4(127, 0, 0, 3), Port: 0}
	loadBalancer := NewLoadBalancerRR()
	endpoint, err := loadBalancer.NextEndpoint("foo", nil)
	if err == nil || len(endpoint) != 0 {
		t.Errorf("Didn't fail with non-existent service")
	}

	loadBalancer.NewService("foo", api.AffinityTypeNone, 0)
	endpoints := make([]api.Endpoints, 1)
	endpoints[0] = api.Endpoints{
		ObjectMeta: api.ObjectMeta{Name: "foo"},
		Endpoints:  []string{"endpoint:1", "endpoint:2", "endpoint:3"},
	}
	loadBalancer.OnUpdate(endpoints)
	shuffledEndpoints := loadBalancer.endpointsMap["foo"]
	expectEndpoint(t, loadBalancer, "foo", shuffledEndpoints[0], client1)
	expectEndpoint(t, loadBalancer, "foo", shuffledEndpoints[1], client1)
	expectEndpoint(t, loadBalancer, "foo", shuffledEndpoints[2], client2)
	expectEndpoint(t, loadBalancer, "foo", shuffledEndpoints[0], client2)
	expectEndpoint(t, loadBalancer, "foo", shuffledEndpoints[1], client3)
	expectEndpoint(t, loadBalancer, "foo", shuffledEndpoints[2], client3)
	expectEndpoint(t, loadBalancer, "foo", shuffledEndpoints[0], client1)
	expectEndpoint(t, loadBalancer, "foo", shuffledEndpoints[1], client1)
}

func TestStickyLoadBalanaceWorksWithMultipleEndpointsRemoveOne(t *testing.T) {
	client1 := &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0}
	client2 := &net.TCPAddr{IP: net.IPv4(127, 0, 0, 2), Port: 0}
	client3 := &net.TCPAddr{IP: net.IPv4(127, 0, 0, 3), Port: 0}
	client4 := &net.TCPAddr{IP: net.IPv4(127, 0, 0, 4), Port: 0}
	client5 := &net.TCPAddr{IP: net.IPv4(127, 0, 0, 5), Port: 0}
	client6 := &net.TCPAddr{IP: net.IPv4(127, 0, 0, 6), Port: 0}
	loadBalancer := NewLoadBalancerRR()
	endpoint, err := loadBalancer.NextEndpoint("foo", nil)
	if err == nil || len(endpoint) != 0 {
		t.Errorf("Didn't fail with non-existent service")
	}

	loadBalancer.NewService("foo", api.AffinityTypeClientIP, 0)
	endpoints := make([]api.Endpoints, 1)
	endpoints[0] = api.Endpoints{
		ObjectMeta: api.ObjectMeta{Name: "foo"},
		Endpoints:  []string{"endpoint:1", "endpoint:2", "endpoint:3"},
	}
	loadBalancer.OnUpdate(endpoints)
	shuffledEndpoints := loadBalancer.endpointsMap["foo"]
	expectEndpoint(t, loadBalancer, "foo", shuffledEndpoints[0], client1)
	client1Endpoint := shuffledEndpoints[0]
	expectEndpoint(t, loadBalancer, "foo", shuffledEndpoints[0], client1)
	expectEndpoint(t, loadBalancer, "foo", shuffledEndpoints[1], client2)
	client2Endpoint := shuffledEndpoints[1]
	expectEndpoint(t, loadBalancer, "foo", shuffledEndpoints[1], client2)
	expectEndpoint(t, loadBalancer, "foo", shuffledEndpoints[2], client3)
	client3Endpoint := shuffledEndpoints[2]

	endpoints[0] = api.Endpoints{
		ObjectMeta: api.ObjectMeta{Name: "foo"},
		Endpoints:  []string{"endpoint:1", "endpoint:2"},
	}
	loadBalancer.OnUpdate(endpoints)
	shuffledEndpoints = loadBalancer.endpointsMap["foo"]
	if client1Endpoint == "endpoint:3" {
		client1Endpoint = shuffledEndpoints[0]
	} else if client2Endpoint == "endpoint:3" {
		client2Endpoint = shuffledEndpoints[0]
	} else if client3Endpoint == "endpoint:3" {
		client3Endpoint = shuffledEndpoints[0]
	}
	expectEndpoint(t, loadBalancer, "foo", client1Endpoint, client1)
	expectEndpoint(t, loadBalancer, "foo", client2Endpoint, client2)
	expectEndpoint(t, loadBalancer, "foo", client3Endpoint, client3)

	endpoints[0] = api.Endpoints{
		ObjectMeta: api.ObjectMeta{Name: "foo"},
		Endpoints:  []string{"endpoint:1", "endpoint:2", "endpoint:4"},
	}
	loadBalancer.OnUpdate(endpoints)
	shuffledEndpoints = loadBalancer.endpointsMap["foo"]
	expectEndpoint(t, loadBalancer, "foo", client1Endpoint, client1)
	expectEndpoint(t, loadBalancer, "foo", client2Endpoint, client2)
	expectEndpoint(t, loadBalancer, "foo", client3Endpoint, client3)
	expectEndpoint(t, loadBalancer, "foo", shuffledEndpoints[0], client4)
	expectEndpoint(t, loadBalancer, "foo", shuffledEndpoints[1], client5)
	expectEndpoint(t, loadBalancer, "foo", shuffledEndpoints[2], client6)
}

func TestStickyLoadBalanceWorksWithMultipleEndpointsAndUpdates(t *testing.T) {
	client1 := &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0}
	client2 := &net.TCPAddr{IP: net.IPv4(127, 0, 0, 2), Port: 0}
	client3 := &net.TCPAddr{IP: net.IPv4(127, 0, 0, 3), Port: 0}
	loadBalancer := NewLoadBalancerRR()
	endpoint, err := loadBalancer.NextEndpoint("foo", nil)
	if err == nil || len(endpoint) != 0 {
		t.Errorf("Didn't fail with non-existent service")
	}

	loadBalancer.NewService("foo", api.AffinityTypeClientIP, 0)
	endpoints := make([]api.Endpoints, 1)
	endpoints[0] = api.Endpoints{
		ObjectMeta: api.ObjectMeta{Name: "foo"},
		Endpoints:  []string{"endpoint:1", "endpoint:2", "endpoint:3"},
	}
	loadBalancer.OnUpdate(endpoints)
	shuffledEndpoints := loadBalancer.endpointsMap["foo"]
	expectEndpoint(t, loadBalancer, "foo", shuffledEndpoints[0], client1)
	expectEndpoint(t, loadBalancer, "foo", shuffledEndpoints[0], client1)
	expectEndpoint(t, loadBalancer, "foo", shuffledEndpoints[1], client2)
	expectEndpoint(t, loadBalancer, "foo", shuffledEndpoints[1], client2)
	expectEndpoint(t, loadBalancer, "foo", shuffledEndpoints[2], client3)
	expectEndpoint(t, loadBalancer, "foo", shuffledEndpoints[0], client1)
	expectEndpoint(t, loadBalancer, "foo", shuffledEndpoints[1], client2)
	// Then update the configuration with one fewer endpoints, make sure
	// we start in the beginning again
	endpoints[0] = api.Endpoints{ObjectMeta: api.ObjectMeta{Name: "foo"},
		Endpoints: []string{"endpoint:4", "endpoint:5"},
	}
	loadBalancer.OnUpdate(endpoints)
	shuffledEndpoints = loadBalancer.endpointsMap["foo"]
	expectEndpoint(t, loadBalancer, "foo", shuffledEndpoints[0], client1)
	expectEndpoint(t, loadBalancer, "foo", shuffledEndpoints[1], client2)
	expectEndpoint(t, loadBalancer, "foo", shuffledEndpoints[0], client1)
	expectEndpoint(t, loadBalancer, "foo", shuffledEndpoints[1], client2)
	expectEndpoint(t, loadBalancer, "foo", shuffledEndpoints[1], client2)

	// Clear endpoints
	endpoints[0] = api.Endpoints{ObjectMeta: api.ObjectMeta{Name: "foo"}, Endpoints: []string{}}
	loadBalancer.OnUpdate(endpoints)

	endpoint, err = loadBalancer.NextEndpoint("foo", nil)
	if err == nil || len(endpoint) != 0 {
		t.Errorf("Didn't fail with non-existent service")
	}
}

func TestStickyLoadBalanceWorksWithServiceRemoval(t *testing.T) {
	client1 := &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0}
	client2 := &net.TCPAddr{IP: net.IPv4(127, 0, 0, 2), Port: 0}
	client3 := &net.TCPAddr{IP: net.IPv4(127, 0, 0, 3), Port: 0}
	loadBalancer := NewLoadBalancerRR()
	endpoint, err := loadBalancer.NextEndpoint("foo", nil)
	if err == nil || len(endpoint) != 0 {
		t.Errorf("Didn't fail with non-existent service")
	}
	loadBalancer.NewService("foo", api.AffinityTypeClientIP, 0)
	endpoints := make([]api.Endpoints, 2)
	endpoints[0] = api.Endpoints{
		ObjectMeta: api.ObjectMeta{Name: "foo"},
		Endpoints:  []string{"endpoint:1", "endpoint:2", "endpoint:3"},
	}
	loadBalancer.NewService("bar", api.AffinityTypeClientIP, 0)
	endpoints[1] = api.Endpoints{
		ObjectMeta: api.ObjectMeta{Name: "bar"},
		Endpoints:  []string{"endpoint:4", "endpoint:5"},
	}
	loadBalancer.OnUpdate(endpoints)
	shuffledFooEndpoints := loadBalancer.endpointsMap["foo"]
	expectEndpoint(t, loadBalancer, "foo", shuffledFooEndpoints[0], client1)
	expectEndpoint(t, loadBalancer, "foo", shuffledFooEndpoints[1], client2)
	expectEndpoint(t, loadBalancer, "foo", shuffledFooEndpoints[2], client3)
	expectEndpoint(t, loadBalancer, "foo", shuffledFooEndpoints[2], client3)
	expectEndpoint(t, loadBalancer, "foo", shuffledFooEndpoints[0], client1)
	expectEndpoint(t, loadBalancer, "foo", shuffledFooEndpoints[1], client2)

	shuffledBarEndpoints := loadBalancer.endpointsMap["bar"]
	expectEndpoint(t, loadBalancer, "foo", shuffledFooEndpoints[0], client1)
	expectEndpoint(t, loadBalancer, "foo", shuffledFooEndpoints[1], client2)
	expectEndpoint(t, loadBalancer, "foo", shuffledFooEndpoints[0], client1)
	expectEndpoint(t, loadBalancer, "foo", shuffledFooEndpoints[1], client2)
	expectEndpoint(t, loadBalancer, "foo", shuffledFooEndpoints[0], client1)
	expectEndpoint(t, loadBalancer, "foo", shuffledFooEndpoints[0], client1)

	// Then update the configuration by removing foo
	loadBalancer.OnUpdate(endpoints[1:])
	endpoint, err = loadBalancer.NextEndpoint("foo", nil)
	if err == nil || len(endpoint) != 0 {
		t.Errorf("Didn't fail with non-existent service")
	}

	// but bar is still there, and we continue RR from where we left off.
	shuffledBarEndpoints = loadBalancer.endpointsMap["bar"]
	expectEndpoint(t, loadBalancer, "bar", shuffledBarEndpoints[0], client1)
	expectEndpoint(t, loadBalancer, "bar", shuffledBarEndpoints[1], client2)
	expectEndpoint(t, loadBalancer, "bar", shuffledBarEndpoints[0], client1)
	expectEndpoint(t, loadBalancer, "bar", shuffledBarEndpoints[1], client2)
	expectEndpoint(t, loadBalancer, "bar", shuffledBarEndpoints[0], client1)
	expectEndpoint(t, loadBalancer, "bar", shuffledBarEndpoints[0], client1)
}
