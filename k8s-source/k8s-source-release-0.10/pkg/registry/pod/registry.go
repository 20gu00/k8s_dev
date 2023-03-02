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

package pod

import (
	"github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/labels"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/watch"
)

// Registry is an interface implemented by things that know how to store Pod objects.
type Registry interface {
	// ListPods obtains a list of pods having labels which match selector.
	ListPods(ctx api.Context, selector labels.Selector) (*api.PodList, error)
	// ListPodsPredicate obtains a list of pods for which filter returns true.
	ListPodsPredicate(ctx api.Context, filter func(*api.Pod) bool) (*api.PodList, error)
	// Watch for new/changed/deleted pods
	WatchPods(ctx api.Context, label, field labels.Selector, resourceVersion string) (watch.Interface, error)
	// Get a specific pod
	GetPod(ctx api.Context, podID string) (*api.Pod, error)
	// Create a pod based on a specification.
	CreatePod(ctx api.Context, pod *api.Pod) error
	// Update an existing pod
	UpdatePod(ctx api.Context, pod *api.Pod) error
	// Delete an existing pod
	DeletePod(ctx api.Context, podID string) error
}
