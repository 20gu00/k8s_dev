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

package openstack

import (
	"errors"
	"fmt"
	"io"
	"net"
	"regexp"
	"time"

	"code.google.com/p/gcfg"
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack"
	"github.com/rackspace/gophercloud/openstack/compute/v2/flavors"
	"github.com/rackspace/gophercloud/openstack/compute/v2/servers"
	"github.com/rackspace/gophercloud/openstack/networking/v2/extensions/lbaas/members"
	"github.com/rackspace/gophercloud/openstack/networking/v2/extensions/lbaas/monitors"
	"github.com/rackspace/gophercloud/openstack/networking/v2/extensions/lbaas/pools"
	"github.com/rackspace/gophercloud/openstack/networking/v2/extensions/lbaas/vips"
	"github.com/rackspace/gophercloud/pagination"

	"github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/api/resource"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/cloudprovider"
	"github.com/golang/glog"
)

var ErrNotFound = errors.New("Failed to find object")
var ErrMultipleResults = errors.New("Multiple results where only one expected")
var ErrNoAddressFound = errors.New("No address found for host")
var ErrAttrNotFound = errors.New("Expected attribute not found")

// encoding.TextUnmarshaler interface for time.Duration
type MyDuration struct {
	time.Duration
}

func (d *MyDuration) UnmarshalText(text []byte) error {
	res, err := time.ParseDuration(string(text))
	if err != nil {
		return err
	}
	d.Duration = res
	return nil
}

type LoadBalancerOpts struct {
	SubnetId          string     `gcfg:"subnet-id"` // required
	CreateMonitor     bool       `gcfg:"create-monitor"`
	MonitorDelay      MyDuration `gcfg:"monitor-delay"`
	MonitorTimeout    MyDuration `gcfg:"monitor-timeout"`
	MonitorMaxRetries uint       `gcfg:"monitor-max-retries"`
}

// OpenStack is an implementation of cloud provider Interface for OpenStack.
type OpenStack struct {
	provider *gophercloud.ProviderClient
	region   string
	lbOpts   LoadBalancerOpts
}

type Config struct {
	Global struct {
		AuthUrl    string `gcfg:"auth-url"`
		Username   string
		UserId     string `gcfg:"user-id"`
		Password   string
		ApiKey     string `gcfg:"api-key"`
		TenantId   string `gcfg:"tenant-id"`
		TenantName string `gcfg:"tenant-name"`
		DomainId   string `gcfg:"domain-id"`
		DomainName string `gcfg:"domain-name"`
		Region     string
	}
	LoadBalancer LoadBalancerOpts
}

func init() {
	cloudprovider.RegisterCloudProvider("openstack", func(config io.Reader) (cloudprovider.Interface, error) {
		cfg, err := readConfig(config)
		if err != nil {
			return nil, err
		}
		return newOpenStack(cfg)
	})
}

func (cfg Config) toAuthOptions() gophercloud.AuthOptions {
	return gophercloud.AuthOptions{
		IdentityEndpoint: cfg.Global.AuthUrl,
		Username:         cfg.Global.Username,
		UserID:           cfg.Global.UserId,
		Password:         cfg.Global.Password,
		APIKey:           cfg.Global.ApiKey,
		TenantID:         cfg.Global.TenantId,
		TenantName:       cfg.Global.TenantName,

		// Persistent service, so we need to be able to renew tokens
		AllowReauth: true,
	}
}

func readConfig(config io.Reader) (Config, error) {
	if config == nil {
		err := fmt.Errorf("no OpenStack cloud provider config file given")
		return Config{}, err
	}

	var cfg Config
	err := gcfg.ReadInto(&cfg, config)
	return cfg, err
}

func newOpenStack(cfg Config) (*OpenStack, error) {
	provider, err := openstack.AuthenticatedClient(cfg.toAuthOptions())
	if err != nil {
		return nil, err
	}

	os := OpenStack{
		provider: provider,
		region:   cfg.Global.Region,
		lbOpts:   cfg.LoadBalancer,
	}
	return &os, nil
}

type Instances struct {
	compute            *gophercloud.ServiceClient
	flavor_to_resource map[string]*api.NodeResources // keyed by flavor id
}

// Instances returns an implementation of Instances for OpenStack.
func (os *OpenStack) Instances() (cloudprovider.Instances, bool) {
	glog.V(2).Info("openstack.Instances() called")

	compute, err := openstack.NewComputeV2(os.provider, gophercloud.EndpointOpts{
		Region: os.region,
	})
	if err != nil {
		glog.Warningf("Failed to find compute endpoint: %v", err)
		return nil, false
	}

	pager := flavors.ListDetail(compute, nil)

	flavor_to_resource := make(map[string]*api.NodeResources)
	err = pager.EachPage(func(page pagination.Page) (bool, error) {
		flavorList, err := flavors.ExtractFlavors(page)
		if err != nil {
			return false, err
		}
		for _, flavor := range flavorList {
			rsrc := api.NodeResources{
				Capacity: api.ResourceList{
					api.ResourceCPU:            *resource.NewMilliQuantity(int64(flavor.VCPUs*1000), resource.DecimalSI),
					api.ResourceMemory:         resource.MustParse(fmt.Sprintf("%dMi", flavor.RAM)),
					"openstack.org/disk":       resource.MustParse(fmt.Sprintf("%dG", flavor.Disk)),
					"openstack.org/rxTxFactor": *resource.NewQuantity(int64(flavor.RxTxFactor*1000), resource.DecimalSI),
					"openstack.org/swap":       resource.MustParse(fmt.Sprintf("%dMi", flavor.Swap)),
				},
			}
			flavor_to_resource[flavor.ID] = &rsrc
		}
		return true, nil
	})
	if err != nil {
		glog.Warningf("Failed to find compute flavors: %v", err)
		return nil, false
	}

	glog.V(2).Infof("Found %v compute flavors", len(flavor_to_resource))
	glog.V(1).Info("Claiming to support Instances")

	return &Instances{compute, flavor_to_resource}, true
}

func (i *Instances) List(name_filter string) ([]string, error) {
	glog.V(2).Infof("openstack List(%v) called", name_filter)

	opts := servers.ListOpts{
		Name:   name_filter,
		Status: "ACTIVE",
	}
	pager := servers.List(i.compute, opts)

	ret := make([]string, 0)
	err := pager.EachPage(func(page pagination.Page) (bool, error) {
		sList, err := servers.ExtractServers(page)
		if err != nil {
			return false, err
		}
		for _, server := range sList {
			ret = append(ret, server.Name)
		}
		return true, nil
	})
	if err != nil {
		return nil, err
	}

	glog.V(2).Infof("Found %v entries: %v", len(ret), ret)

	return ret, nil
}

func getServerByName(client *gophercloud.ServiceClient, name string) (*servers.Server, error) {
	opts := servers.ListOpts{
		Name:   fmt.Sprintf("^%s$", regexp.QuoteMeta(name)),
		Status: "ACTIVE",
	}
	pager := servers.List(client, opts)

	serverList := make([]servers.Server, 0, 1)

	err := pager.EachPage(func(page pagination.Page) (bool, error) {
		s, err := servers.ExtractServers(page)
		if err != nil {
			return false, err
		}
		serverList = append(serverList, s...)
		if len(serverList) > 1 {
			return false, ErrMultipleResults
		}
		return true, nil
	})
	if err != nil {
		return nil, err
	}

	if len(serverList) == 0 {
		return nil, ErrNotFound
	} else if len(serverList) > 1 {
		return nil, ErrMultipleResults
	}

	return &serverList[0], nil
}

func firstAddr(netblob interface{}) string {
	// Run-time types for the win :(
	list, ok := netblob.([]interface{})
	if !ok || len(list) < 1 {
		return ""
	}
	props, ok := list[0].(map[string]interface{})
	if !ok {
		return ""
	}
	tmp, ok := props["addr"]
	if !ok {
		return ""
	}
	addr, ok := tmp.(string)
	if !ok {
		return ""
	}
	return addr
}

func getAddressByName(api *gophercloud.ServiceClient, name string) (string, error) {
	srv, err := getServerByName(api, name)
	if err != nil {
		return "", err
	}

	var s string
	if s == "" {
		s = firstAddr(srv.Addresses["private"])
	}
	if s == "" {
		s = firstAddr(srv.Addresses["public"])
	}
	if s == "" {
		s = srv.AccessIPv4
	}
	if s == "" {
		s = srv.AccessIPv6
	}
	if s == "" {
		return "", ErrNoAddressFound
	}
	return s, nil
}

func (i *Instances) IPAddress(name string) (net.IP, error) {
	glog.V(2).Infof("IPAddress(%v) called", name)

	ip, err := getAddressByName(i.compute, name)
	if err != nil {
		return nil, err
	}

	glog.V(2).Infof("IPAddress(%v) => %v", name, ip)

	return net.ParseIP(ip), err
}

func (i *Instances) GetNodeResources(name string) (*api.NodeResources, error) {
	glog.V(2).Infof("GetNodeResources(%v) called", name)

	srv, err := getServerByName(i.compute, name)
	if err != nil {
		return nil, err
	}

	s, ok := srv.Flavor["id"]
	if !ok {
		return nil, ErrAttrNotFound
	}
	flavId, ok := s.(string)
	if !ok {
		return nil, ErrAttrNotFound
	}
	rsrc, ok := i.flavor_to_resource[flavId]
	if !ok {
		return nil, ErrNotFound
	}

	glog.V(2).Infof("GetNodeResources(%v) => %v", name, rsrc)

	return rsrc, nil
}

func (os *OpenStack) Clusters() (cloudprovider.Clusters, bool) {
	return nil, false
}

type LoadBalancer struct {
	network *gophercloud.ServiceClient
	compute *gophercloud.ServiceClient
	opts    LoadBalancerOpts
}

func (os *OpenStack) TCPLoadBalancer() (cloudprovider.TCPLoadBalancer, bool) {
	// TODO: Search for and support Rackspace loadbalancer API, and others.
	network, err := openstack.NewNetworkV2(os.provider, gophercloud.EndpointOpts{
		Region: os.region,
	})
	if err != nil {
		glog.Warningf("Failed to find neutron endpoint: %v", err)
		return nil, false
	}

	compute, err := openstack.NewComputeV2(os.provider, gophercloud.EndpointOpts{
		Region: os.region,
	})
	if err != nil {
		glog.Warningf("Failed to find compute endpoint: %v", err)
		return nil, false
	}

	glog.V(1).Info("Claiming to support TCPLoadBalancer")

	return &LoadBalancer{network, compute, os.lbOpts}, true
}

func getVipByName(client *gophercloud.ServiceClient, name string) (*vips.VirtualIP, error) {
	opts := vips.ListOpts{
		Name: name,
	}
	pager := vips.List(client, opts)

	vipList := make([]vips.VirtualIP, 0, 1)

	err := pager.EachPage(func(page pagination.Page) (bool, error) {
		v, err := vips.ExtractVIPs(page)
		if err != nil {
			return false, err
		}
		vipList = append(vipList, v...)
		if len(vipList) > 1 {
			return false, ErrMultipleResults
		}
		return true, nil
	})
	if err != nil {
		return nil, err
	}

	if len(vipList) == 0 {
		return nil, ErrNotFound
	} else if len(vipList) > 1 {
		return nil, ErrMultipleResults
	}

	return &vipList[0], nil
}

func (lb *LoadBalancer) TCPLoadBalancerExists(name, region string) (bool, error) {
	vip, err := getVipByName(lb.network, name)
	if err == ErrNotFound {
		return false, nil
	}
	return vip != nil, err
}

// TODO: This code currently ignores 'region' and always creates a
// loadbalancer in only the current OpenStack region.  We should take
// a list of regions (from config) and query/create loadbalancers in
// each region.

func (lb *LoadBalancer) CreateTCPLoadBalancer(name, region string, externalIP net.IP, port int, hosts []string, affinity api.AffinityType) (net.IP, error) {
	glog.V(2).Infof("CreateTCPLoadBalancer(%v, %v, %v, %v, %v)", name, region, externalIP, port, hosts)
	if affinity != api.AffinityTypeNone {
		return nil, fmt.Errorf("unsupported load balancer affinity: %v", affinity)
	}
	pool, err := pools.Create(lb.network, pools.CreateOpts{
		Name:     name,
		Protocol: pools.ProtocolTCP,
		SubnetID: lb.opts.SubnetId,
	}).Extract()
	if err != nil {
		return nil, err
	}

	for _, host := range hosts {
		addr, err := getAddressByName(lb.compute, host)
		if err != nil {
			return nil, err
		}

		_, err = members.Create(lb.network, members.CreateOpts{
			PoolID:       pool.ID,
			ProtocolPort: port,
			Address:      addr,
		}).Extract()
		if err != nil {
			pools.Delete(lb.network, pool.ID)
			return nil, err
		}
	}

	var mon *monitors.Monitor
	if lb.opts.CreateMonitor {
		mon, err = monitors.Create(lb.network, monitors.CreateOpts{
			Type:       monitors.TypeTCP,
			Delay:      int(lb.opts.MonitorDelay.Duration.Seconds()),
			Timeout:    int(lb.opts.MonitorTimeout.Duration.Seconds()),
			MaxRetries: int(lb.opts.MonitorMaxRetries),
		}).Extract()
		if err != nil {
			pools.Delete(lb.network, pool.ID)
			return nil, err
		}

		_, err = pools.AssociateMonitor(lb.network, pool.ID, mon.ID).Extract()
		if err != nil {
			monitors.Delete(lb.network, mon.ID)
			pools.Delete(lb.network, pool.ID)
			return nil, err
		}
	}

	vip, err := vips.Create(lb.network, vips.CreateOpts{
		Name:         name,
		Description:  fmt.Sprintf("Kubernetes external service %s", name),
		Address:      externalIP.String(),
		Protocol:     "TCP",
		ProtocolPort: port,
		PoolID:       pool.ID,
	}).Extract()
	if err != nil {
		if mon != nil {
			monitors.Delete(lb.network, mon.ID)
		}
		pools.Delete(lb.network, pool.ID)
		return nil, err
	}

	return net.ParseIP(vip.Address), nil
}

func (lb *LoadBalancer) UpdateTCPLoadBalancer(name, region string, hosts []string) error {
	glog.V(2).Infof("UpdateTCPLoadBalancer(%v, %v, %v)", name, region, hosts)

	vip, err := getVipByName(lb.network, name)
	if err != nil {
		return err
	}

	// Set of member (addresses) that _should_ exist
	addrs := map[string]bool{}
	for _, host := range hosts {
		addr, err := getAddressByName(lb.compute, host)
		if err != nil {
			return err
		}

		addrs[addr] = true
	}

	// Iterate over members that _do_ exist
	pager := members.List(lb.network, members.ListOpts{PoolID: vip.PoolID})
	err = pager.EachPage(func(page pagination.Page) (bool, error) {
		memList, err := members.ExtractMembers(page)
		if err != nil {
			return false, err
		}

		for _, member := range memList {
			if _, found := addrs[member.Address]; found {
				// Member already exists
				delete(addrs, member.Address)
			} else {
				// Member needs to be deleted
				err = members.Delete(lb.network, member.ID).ExtractErr()
				if err != nil {
					return false, err
				}
			}
		}

		return true, nil
	})
	if err != nil {
		return err
	}

	// Anything left in addrs is a new member that needs to be added
	for addr := range addrs {
		_, err := members.Create(lb.network, members.CreateOpts{
			PoolID:       vip.PoolID,
			Address:      addr,
			ProtocolPort: vip.ProtocolPort,
		}).Extract()
		if err != nil {
			return err
		}
	}

	return nil
}

func (lb *LoadBalancer) DeleteTCPLoadBalancer(name, region string) error {
	glog.V(2).Infof("DeleteTCPLoadBalancer(%v, %v)", name, region)

	vip, err := getVipByName(lb.network, name)
	if err != nil {
		return err
	}

	pool, err := pools.Get(lb.network, vip.PoolID).Extract()
	if err != nil {
		return err
	}

	// Have to delete VIP before pool can be deleted
	err = vips.Delete(lb.network, vip.ID).ExtractErr()
	if err != nil {
		return err
	}

	// Ignore errors for everything following here

	for _, monId := range pool.MonitorIDs {
		pools.DisassociateMonitor(lb.network, pool.ID, monId)
	}
	pools.Delete(lb.network, pool.ID)

	return nil
}

func (os *OpenStack) Zones() (cloudprovider.Zones, bool) {
	glog.V(1).Info("Claiming to support Zones")

	return os, true
}
func (os *OpenStack) GetZone() (cloudprovider.Zone, error) {
	glog.V(1).Infof("Current zone is %v", os.region)

	return cloudprovider.Zone{Region: os.region}, nil
}
