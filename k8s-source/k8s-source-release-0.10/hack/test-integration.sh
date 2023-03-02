#!/bin/bash

# Copyright 2014 Google Inc. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# any command line arguments will be passed to hack/build_go.sh to build the
# cmd/integration binary.  --use_go_build is a legitimate argument, as are
# any other build time arguments.

set -o errexit
set -o nounset
set -o pipefail

KUBE_ROOT=$(dirname "${BASH_SOURCE}")/..
source "${KUBE_ROOT}/hack/lib/init.sh"

cleanup() {
  kube::etcd::cleanup
  kube::log::status "Integration test cleanup complete"
}

"${KUBE_ROOT}/hack/build-go.sh" "$@" cmd/integration

# Run cleanup to stop etcd on interrupt or other kill signal.
trap cleanup EXIT

kube::etcd::start

kube::log::status "Running integration test cases"
KUBE_GOFLAGS="-tags 'integration no-docker' " \
  KUBE_RACE="-race" \
  "${KUBE_ROOT}/hack/test-go.sh" test/integration

kube::log::status "Running integration test scenario"

"${KUBE_OUTPUT_HOSTBIN}/integration" --v=2

cleanup
