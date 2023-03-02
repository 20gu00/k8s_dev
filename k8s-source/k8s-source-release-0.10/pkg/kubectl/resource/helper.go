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

package resource

import (
	"github.com/GoogleCloudPlatform/kubernetes/pkg/api/meta"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/labels"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/runtime"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/watch"
)

// Helper provides methods for retrieving or mutating a RESTful
// resource.
type Helper struct {
	// The name of this resource as the server would recognize it
	Resource string
	// A RESTClient capable of mutating this resource
	RESTClient RESTClient
	// A codec for decoding and encoding objects of this resource type.
	Codec runtime.Codec
	// An interface for reading or writing the resource version of this
	// type.
	Versioner runtime.ResourceVersioner
}

// NewHelper creates a Helper from a ResourceMapping
func NewHelper(client RESTClient, mapping *meta.RESTMapping) *Helper {
	return &Helper{
		RESTClient: client,
		Resource:   mapping.Resource,
		Codec:      mapping.Codec,
		Versioner:  mapping.MetadataAccessor,
	}
}

func (m *Helper) Get(namespace, name string) (runtime.Object, error) {
	return m.RESTClient.Get().
		Namespace(namespace).
		Resource(m.Resource).
		Name(name).
		Do().
		Get()
}

func (m *Helper) List(namespace string, selector labels.Selector) (runtime.Object, error) {
	return m.RESTClient.Get().
		Namespace(namespace).
		Resource(m.Resource).
		SelectorParam("labels", selector).
		Do().
		Get()
}

func (m *Helper) Watch(namespace, resourceVersion string, labelSelector, fieldSelector labels.Selector) (watch.Interface, error) {
	return m.RESTClient.Get().
		Prefix("watch").
		Namespace(namespace).
		Resource(m.Resource).
		Param("resourceVersion", resourceVersion).
		SelectorParam("labels", labelSelector).
		SelectorParam("fields", fieldSelector).
		Watch()
}

func (m *Helper) WatchSingle(namespace, name, resourceVersion string) (watch.Interface, error) {
	return m.RESTClient.Get().
		Prefix("watch").
		Namespace(namespace).
		Resource(m.Resource).
		Name(name).
		Param("resourceVersion", resourceVersion).
		Watch()
}

func (m *Helper) Delete(namespace, name string) error {
	return m.RESTClient.Delete().
		Namespace(namespace).
		Resource(m.Resource).
		Name(name).
		Do().
		Error()
}

func (m *Helper) Create(namespace string, modify bool, data []byte) error {
	if modify {
		obj, err := m.Codec.Decode(data)
		if err != nil {
			// We don't know how to check a version on this object, but create it anyway
			return createResource(m.RESTClient, m.Resource, namespace, data)
		}

		// Attempt to version the object based on client logic.
		version, err := m.Versioner.ResourceVersion(obj)
		if err != nil {
			// We don't know how to clear the version on this object, so send it to the server as is
			return createResource(m.RESTClient, m.Resource, namespace, data)
		}
		if version != "" {
			if err := m.Versioner.SetResourceVersion(obj, ""); err != nil {
				return err
			}
			newData, err := m.Codec.Encode(obj)
			if err != nil {
				return err
			}
			data = newData
		}
	}

	return createResource(m.RESTClient, m.Resource, namespace, data)
}

func createResource(c RESTClient, resource, namespace string, data []byte) error {
	return c.Post().Namespace(namespace).Resource(resource).Body(data).Do().Error()
}

func (m *Helper) Update(namespace, name string, overwrite bool, data []byte) error {
	c := m.RESTClient

	obj, err := m.Codec.Decode(data)
	if err != nil {
		// We don't know how to handle this object, but update it anyway
		return updateResource(c, m.Resource, namespace, name, data)
	}

	// Attempt to version the object based on client logic.
	version, err := m.Versioner.ResourceVersion(obj)
	if err != nil {
		// We don't know how to version this object, so send it to the server as is
		return updateResource(c, m.Resource, namespace, name, data)
	}
	if version == "" && overwrite {
		// Retrieve the current version of the object to overwrite the server object
		serverObj, err := c.Get().Namespace(namespace).Resource(m.Resource).Name(name).Do().Get()
		if err != nil {
			// The object does not exist, but we want it to be created
			return updateResource(c, m.Resource, namespace, name, data)
		}
		serverVersion, err := m.Versioner.ResourceVersion(serverObj)
		if err != nil {
			return err
		}
		if err := m.Versioner.SetResourceVersion(obj, serverVersion); err != nil {
			return err
		}
		newData, err := m.Codec.Encode(obj)
		if err != nil {
			return err
		}
		data = newData
	}

	return updateResource(c, m.Resource, namespace, name, data)
}

func updateResource(c RESTClient, resource, namespace, name string, data []byte) error {
	return c.Put().Namespace(namespace).Resource(resource).Name(name).Body(data).Do().Error()
}
