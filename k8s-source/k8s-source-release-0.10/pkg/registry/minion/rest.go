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

package minion

import (
	"errors"
	"fmt"
	"net"
	"strconv"

	"github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	kerrors "github.com/GoogleCloudPlatform/kubernetes/pkg/api/errors"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/api/rest"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/api/validation"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/apiserver"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/labels"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/master/ports"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/runtime"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/watch"
)

// REST adapts minion into apiserver's RESTStorage model.
type REST struct {
	registry Registry
}

// NewREST returns a new apiserver.RESTStorage implementation for minion.
func NewREST(m Registry) *REST {
	return &REST{
		registry: m,
	}
}

var ErrDoesNotExist = errors.New("The requested resource does not exist.")
var ErrNotHealty = errors.New("The requested minion is not healthy.")

// Create satisfies the RESTStorage interface.
func (rs *REST) Create(ctx api.Context, obj runtime.Object) (<-chan apiserver.RESTResult, error) {
	minion, ok := obj.(*api.Node)
	if !ok {
		return nil, fmt.Errorf("not a minion: %#v", obj)
	}

	if err := rest.BeforeCreate(rest.Nodes, ctx, obj); err != nil {
		return nil, err
	}

	return apiserver.MakeAsync(func() (runtime.Object, error) {
		if err := rs.registry.CreateMinion(ctx, minion); err != nil {
			err = rest.CheckGeneratedNameError(rest.Nodes, err, minion)
			return nil, err
		}
		return minion, nil
	}), nil
}

// Delete satisfies the RESTStorage interface.
func (rs *REST) Delete(ctx api.Context, id string) (<-chan apiserver.RESTResult, error) {
	minion, err := rs.registry.GetMinion(ctx, id)
	if minion == nil {
		return nil, ErrDoesNotExist
	}
	if err != nil {
		return nil, err
	}
	return apiserver.MakeAsync(func() (runtime.Object, error) {
		return &api.Status{Status: api.StatusSuccess}, rs.registry.DeleteMinion(ctx, id)
	}), nil
}

// Get satisfies the RESTStorage interface.
func (rs *REST) Get(ctx api.Context, id string) (runtime.Object, error) {
	minion, err := rs.registry.GetMinion(ctx, id)
	if err != nil {
		return minion, err
	}
	if minion == nil {
		return nil, ErrDoesNotExist
	}
	return minion, err
}

// List satisfies the RESTStorage interface.
func (rs *REST) List(ctx api.Context, label, field labels.Selector) (runtime.Object, error) {
	return rs.registry.ListMinions(ctx)
}

func (rs *REST) New() runtime.Object {
	return &api.Node{}
}

func (*REST) NewList() runtime.Object {
	return &api.NodeList{}
}

// Update satisfies the RESTStorage interface.
func (rs *REST) Update(ctx api.Context, obj runtime.Object) (<-chan apiserver.RESTResult, error) {
	minion, ok := obj.(*api.Node)
	if !ok {
		return nil, fmt.Errorf("not a minion: %#v", obj)
	}
	// This is hacky, but minions don't really have a namespace, but kubectl currently automatically
	// stuffs one in there.  Fix it here temporarily until we fix kubectl
	if minion.Namespace == api.NamespaceDefault {
		minion.Namespace = api.NamespaceNone
	}
	// Clear out the self link, if specified, since it's not in the registry either.
	minion.SelfLink = ""

	oldMinion, err := rs.registry.GetMinion(ctx, minion.Name)
	if err != nil {
		return nil, err
	}

	// This is hacky, but minion HostIP has been moved from spec to status since v1beta2. When updating
	// minion from older client, HostIP will be lost.  Fix it here temporarily until we strip out status
	// info from user input.
	if minion.Status.HostIP == "" {
		minion.Status.HostIP = oldMinion.Status.HostIP
	}

	if errs := validation.ValidateMinionUpdate(oldMinion, minion); len(errs) > 0 {
		return nil, kerrors.NewInvalid("minion", minion.Name, errs)
	}

	return apiserver.MakeAsync(func() (runtime.Object, error) {
		err := rs.registry.UpdateMinion(ctx, minion)
		if err != nil {
			return nil, err
		}
		return rs.registry.GetMinion(ctx, minion.Name)
	}), nil
}

// Watch returns Minions events via a watch.Interface.
// It implements apiserver.ResourceWatcher.
func (rs *REST) Watch(ctx api.Context, label, field labels.Selector, resourceVersion string) (watch.Interface, error) {
	return rs.registry.WatchMinions(ctx, label, field, resourceVersion)
}

// ResourceLocation returns a URL to which one can send traffic for the specified minion.
func (rs *REST) ResourceLocation(ctx api.Context, id string) (string, error) {
	minion, err := rs.registry.GetMinion(ctx, id)
	if err != nil {
		return "", err
	}
	host := minion.Name
	// TODO: Minion webservers should be secure!
	return "http://" + net.JoinHostPort(host, strconv.Itoa(ports.KubeletPort)), nil
}
