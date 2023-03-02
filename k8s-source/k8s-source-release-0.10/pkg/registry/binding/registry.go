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

package binding

import (
	"github.com/GoogleCloudPlatform/kubernetes/pkg/api"
)

// Registry contains the functions needed to support a BindingStorage.
type Registry interface {
	// ApplyBinding should apply the binding. That is, it should actually
	// assign or place pod binding.PodID on machine binding.Host.
	ApplyBinding(ctx api.Context, binding *api.Binding) error
}
