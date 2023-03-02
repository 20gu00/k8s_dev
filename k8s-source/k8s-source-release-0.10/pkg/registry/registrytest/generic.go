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
	"github.com/GoogleCloudPlatform/kubernetes/pkg/registry/generic"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/runtime"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/watch"
)

// GenericRegistry knows how to store & list any runtime.Object. Events don't require
// any non-generic features from the storage layer.
type GenericRegistry struct {
	Err        error
	Object     runtime.Object
	ObjectList runtime.Object
	sync.Mutex

	Broadcaster *watch.Broadcaster
}

func NewGeneric(list runtime.Object) *GenericRegistry {
	return &GenericRegistry{
		ObjectList:  list,
		Broadcaster: watch.NewBroadcaster(0, watch.WaitIfChannelFull),
	}
}

func (r *GenericRegistry) List(ctx api.Context, m generic.Matcher) (runtime.Object, error) {
	r.Lock()
	defer r.Unlock()
	if r.Err != nil {
		return nil, r.Err
	}
	return generic.FilterList(r.ObjectList, m)
}

func (r *GenericRegistry) Watch(ctx api.Context, m generic.Matcher, resourceVersion string) (watch.Interface, error) {
	// TODO: wire filter down into the mux; it needs access to current and previous state :(
	return r.Broadcaster.Watch(), nil
}

func (r *GenericRegistry) Get(ctx api.Context, id string) (runtime.Object, error) {
	r.Lock()
	defer r.Unlock()
	return r.Object, r.Err
}

func (r *GenericRegistry) Create(ctx api.Context, id string, obj runtime.Object) error {
	r.Lock()
	defer r.Unlock()
	r.Object = obj
	r.Broadcaster.Action(watch.Added, obj)
	return r.Err
}

func (r *GenericRegistry) Update(ctx api.Context, id string, obj runtime.Object) error {
	r.Lock()
	defer r.Unlock()
	r.Object = obj
	r.Broadcaster.Action(watch.Modified, obj)
	return r.Err
}

func (r *GenericRegistry) Delete(ctx api.Context, id string) error {
	r.Lock()
	defer r.Unlock()
	r.Broadcaster.Action(watch.Deleted, r.Object)
	return r.Err
}
