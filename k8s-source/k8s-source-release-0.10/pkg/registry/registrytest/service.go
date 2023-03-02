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

package registrytest

import (
	"sync"

	"github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/labels"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/watch"
)

func NewServiceRegistry() *ServiceRegistry {
	return &ServiceRegistry{}
}

type ServiceRegistry struct {
	mu            sync.Mutex
	List          api.ServiceList
	Service       *api.Service
	Err           error
	Endpoints     api.Endpoints
	EndpointsList api.EndpointsList

	DeletedID string
	GottenID  string
	UpdatedID string
}

func (r *ServiceRegistry) SetError(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Err = err
}

func (r *ServiceRegistry) ListServices(ctx api.Context) (*api.ServiceList, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	ns, _ := api.NamespaceFrom(ctx)

	// Copy metadata from internal list into result
	res := new(api.ServiceList)
	res.TypeMeta = r.List.TypeMeta
	res.ListMeta = r.List.ListMeta

	if ns != api.NamespaceAll {
		for _, service := range r.List.Items {
			if ns == service.Namespace {
				res.Items = append(res.Items, service)
			}
		}
	} else {
		res.Items = append([]api.Service{}, r.List.Items...)
	}

	return res, r.Err
}

func (r *ServiceRegistry) CreateService(ctx api.Context, svc *api.Service) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.Service = new(api.Service)
	*r.Service = *svc
	r.List.Items = append(r.List.Items, *svc)
	return r.Err
}

func (r *ServiceRegistry) GetService(ctx api.Context, id string) (*api.Service, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.GottenID = id
	return r.Service, r.Err
}

func (r *ServiceRegistry) DeleteService(ctx api.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.DeletedID = id
	r.Service = nil
	return r.Err
}

func (r *ServiceRegistry) UpdateService(ctx api.Context, svc *api.Service) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.UpdatedID = svc.Name
	*r.Service = *svc
	return r.Err
}

func (r *ServiceRegistry) WatchServices(ctx api.Context, label labels.Selector, field labels.Selector, resourceVersion string) (watch.Interface, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	return nil, r.Err
}

func (r *ServiceRegistry) ListEndpoints(ctx api.Context) (*api.EndpointsList, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	return &r.EndpointsList, r.Err
}

func (r *ServiceRegistry) GetEndpoints(ctx api.Context, id string) (*api.Endpoints, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.GottenID = id
	return &r.Endpoints, r.Err
}

func (r *ServiceRegistry) UpdateEndpoints(ctx api.Context, e *api.Endpoints) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.Endpoints = *e
	return r.Err
}

func (r *ServiceRegistry) WatchEndpoints(ctx api.Context, label, field labels.Selector, resourceVersion string) (watch.Interface, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	return nil, r.Err
}
