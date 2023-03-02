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

package credentialprovider

import (
	"net/url"
	"sort"
	"strings"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/golang/glog"
)

// DockerKeyring tracks a set of docker registry credentials, maintaining a
// reverse index across the registry endpoints. A registry endpoint is made
// up of a host (e.g. registry.example.com), but it may also contain a path
// (e.g. registry.example.com/foo) This index is important for two reasons:
// - registry endpoints may overlap, and when this happens we must find the
//   most specific match for a given image
// - iterating a map does not yield predictable results
type DockerKeyring interface {
	Lookup(image string) (docker.AuthConfiguration, bool)
}

// BasicDockerKeyring is a trivial map-backed implementation of DockerKeyring
type BasicDockerKeyring struct {
	index []string
	creds map[string]docker.AuthConfiguration
}

// lazyDockerKeyring is an implementation of DockerKeyring that lazily
// materializes its dockercfg based on a set of dockerConfigProviders.
type lazyDockerKeyring struct {
	Providers []DockerConfigProvider
}

func (dk *BasicDockerKeyring) Add(cfg DockerConfig) {
	if dk.index == nil {
		dk.index = make([]string, 0)
		dk.creds = make(map[string]docker.AuthConfiguration)
	}
	for loc, ident := range cfg {
		creds := docker.AuthConfiguration{
			Username: ident.Username,
			Password: ident.Password,
			Email:    ident.Email,
		}

		parsed, err := url.Parse(loc)
		if err != nil {
			glog.Errorf("Entry %q in dockercfg invalid (%v), ignoring", loc, err)
			continue
		}

		// The docker client allows exact matches:
		//    foo.bar.com/namespace
		// Or hostname matches:
		//    foo.bar.com
		// See ResolveAuthConfig in docker/registry/auth.go.
		if parsed.Host != "" {
			// NOTE: foo.bar.com comes through as Path.
			dk.creds[parsed.Host] = creds
			dk.index = append(dk.index, parsed.Host)
		}
		if parsed.Path != "/" {
			dk.creds[parsed.Host+parsed.Path] = creds
			dk.index = append(dk.index, parsed.Host+parsed.Path)
		}
	}

	// Update the index used to identify which credentials to use for a given
	// image. The index is reverse-sorted so more specific paths are matched
	// first. For example, if for the given image "quay.io/coreos/etcd",
	// credentials for "quay.io/coreos" should match before "quay.io".
	sort.Sort(sort.Reverse(sort.StringSlice(dk.index)))
}

const defaultRegistryHost = "index.docker.io/v1/"

// isDefaultRegistryMatch determines whether the given image will
// pull from the default registry (DockerHub) based on the
// characteristics of its name.
func isDefaultRegistryMatch(image string) bool {
	parts := strings.SplitN(image, "/", 2)

	if len(parts) == 1 {
		// e.g. library/ubuntu
		return true
	}

	// From: http://blog.docker.com/2013/07/how-to-use-your-own-registry/
	// Docker looks for either a “.” (domain separator) or “:” (port separator)
	// to learn that the first part of the repository name is a location and not
	// a user name.
	return !strings.ContainsAny(parts[0], ".:")
}

// Lookup implements the DockerKeyring method for fetching credentials
// based on image name.
func (dk *BasicDockerKeyring) Lookup(image string) (docker.AuthConfiguration, bool) {
	// range over the index as iterating over a map does not provide
	// a predictable ordering
	for _, k := range dk.index {
		// NOTE: prefix is a sufficient check because while scheme is allowed,
		// it is stripped as part of 'Add'
		if !strings.HasPrefix(image, k) {
			continue
		}

		return dk.creds[k], true
	}

	// Use credentials for the default registry if provided, and appropriate
	if auth, ok := dk.creds[defaultRegistryHost]; ok && isDefaultRegistryMatch(image) {
		return auth, true
	}

	return docker.AuthConfiguration{}, false
}

// Lookup implements the DockerKeyring method for fetching credentials
// based on image name.
func (dk *lazyDockerKeyring) Lookup(image string) (docker.AuthConfiguration, bool) {
	keyring := &BasicDockerKeyring{}

	for _, p := range dk.Providers {
		keyring.Add(p.Provide())
	}

	return keyring.Lookup(image)
}
