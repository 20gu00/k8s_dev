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

package ui

import (
	"net/http"

	assetfs "github.com/elazarl/go-bindata-assetfs"
)

type MuxInterface interface {
	Handle(pattern string, handler http.Handler)
}

func InstallSupport(mux MuxInterface, enableSwaggerSupport bool) {
	// Expose files in www/ on <host>/static/
	fileServer := http.FileServer(&assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, Prefix: "www"})
	prefix := "/static/"
	mux.Handle(prefix, http.StripPrefix(prefix, fileServer))

	if enableSwaggerSupport {
		// Expose files in third_party/swagger-ui/ on <host>/swagger-ui
		fileServer = http.FileServer(&assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, Prefix: "third_party/swagger-ui"})
		prefix = "/swagger-ui/"
		mux.Handle(prefix, http.StripPrefix(prefix, fileServer))
	}
}
