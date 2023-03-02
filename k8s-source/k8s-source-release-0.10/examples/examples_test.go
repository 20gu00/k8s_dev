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

package examples_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/api/latest"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/api/validation"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/runtime"
	"github.com/golang/glog"
)

func validateObject(obj runtime.Object) (errors []error) {
	ctx := api.NewDefaultContext()
	switch t := obj.(type) {
	case *api.ReplicationController:
		if t.Namespace == "" {
			t.Namespace = api.NamespaceDefault
		}
		errors = validation.ValidateReplicationController(t)
	case *api.ReplicationControllerList:
		for i := range t.Items {
			errors = append(errors, validateObject(&t.Items[i])...)
		}
	case *api.Service:
		if t.Namespace == "" {
			t.Namespace = api.NamespaceDefault
		}
		api.ValidNamespace(ctx, &t.ObjectMeta)
		errors = validation.ValidateService(t)
	case *api.ServiceList:
		for i := range t.Items {
			errors = append(errors, validateObject(&t.Items[i])...)
		}
	case *api.Pod:
		if t.Namespace == "" {
			t.Namespace = api.NamespaceDefault
		}
		api.ValidNamespace(ctx, &t.ObjectMeta)
		errors = validation.ValidatePod(t)
	case *api.PodList:
		for i := range t.Items {
			errors = append(errors, validateObject(&t.Items[i])...)
		}
	default:
		return []error{fmt.Errorf("no validation defined for %#v", obj)}
	}
	return errors
}

func walkJSONFiles(inDir string, fn func(name, path string, data []byte)) error {
	err := filepath.Walk(inDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && path != inDir {
			return filepath.SkipDir
		}
		name := filepath.Base(path)
		ext := filepath.Ext(name)
		if ext != "" {
			name = name[:len(name)-len(ext)]
		}
		if !(ext == ".json" || ext == ".yaml") {
			return nil
		}
		glog.Infof("Testing %s", path)
		data, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		fn(name, path, data)
		return nil
	})
	return err
}

func TestExampleObjectSchemas(t *testing.T) {
	cases := map[string]map[string]runtime.Object{
		"../api/examples": {
			"controller":       &api.ReplicationController{},
			"controller-list":  &api.ReplicationControllerList{},
			"pod":              &api.Pod{},
			"pod-list":         &api.PodList{},
			"service":          &api.Service{},
			"external-service": &api.Service{},
			"service-list":     &api.ServiceList{},
		},
		"../examples/guestbook": {
			"frontend-controller":    &api.ReplicationController{},
			"redis-slave-controller": &api.ReplicationController{},
			"redis-master":           &api.Pod{},
			"frontend-service":       &api.Service{},
			"redis-master-service":   &api.Service{},
			"redis-slave-service":    &api.Service{},
		},
		"../examples/walkthrough": {
			"pod1": &api.Pod{},
			"pod2": &api.Pod{},
			"pod-with-http-healthcheck": &api.Pod{},
			"service":                   &api.Service{},
			"replication-controller":    &api.ReplicationController{},
		},
		"../examples/update-demo": {
			"kitten-rc":   &api.ReplicationController{},
			"nautilus-rc": &api.ReplicationController{},
		},
	}

	for path, expected := range cases {
		tested := 0
		err := walkJSONFiles(path, func(name, path string, data []byte) {
			expectedType, found := expected[name]
			if !found {
				t.Errorf("%s does not have a test case defined", path)
				return
			}
			tested += 1
			if err := latest.Codec.DecodeInto(data, expectedType); err != nil {
				t.Errorf("%s did not decode correctly: %v\n%s", path, err, string(data))
				return
			}
			if errors := validateObject(expectedType); len(errors) > 0 {
				t.Errorf("%s did not validate correctly: %v", path, errors)
			}
		})
		if err != nil {
			t.Errorf("Expected no error, Got %v", err)
		}
		if tested != len(expected) {
			t.Errorf("Expected %d examples, Got %d", len(expected), tested)
		}
	}
}

var sampleRegexp = regexp.MustCompile("(?ms)^```(?:(?P<type>yaml)\\w*\\n(?P<content>.+?)|\\w*\\n(?P<content>\\{.+?\\}))\\w*\\n^```")
var subsetRegexp = regexp.MustCompile("(?ms)\\.{3}")

func TestReadme(t *testing.T) {
	paths := []string{
		"../README.md",
		"../examples/walkthrough/README.md",
	}

	for _, path := range paths {
		data, err := ioutil.ReadFile(path)
		if err != nil {
			t.Errorf("Unable to read file %s: %v", path, err)
			continue
		}

		matches := sampleRegexp.FindAllStringSubmatch(string(data), -1)
		if matches == nil {
			continue
		}
		for _, match := range matches {
			var content, subtype string
			for i, name := range sampleRegexp.SubexpNames() {
				if name == "type" {
					subtype = match[i]
				}
				if name == "content" && match[i] != "" {
					content = match[i]
				}
			}
			if subtype == "yaml" && subsetRegexp.FindString(content) != "" {
				t.Logf("skipping (%s): \n%s", subtype, content)
				continue
			}

			//t.Logf("testing (%s): \n%s", subtype, content)
			expectedType := &api.Pod{}
			if err := latest.Codec.DecodeInto([]byte(content), expectedType); err != nil {
				t.Errorf("%s did not decode correctly: %v\n%s", path, err, string(content))
				continue
			}
			if errors := validateObject(expectedType); len(errors) > 0 {
				t.Errorf("%s did not validate correctly: %v", path, errors)
			}
			_, err := latest.Codec.Encode(expectedType)
			if err != nil {
				t.Errorf("Could not encode object: %v", err)
				continue
			}
		}
	}
}
