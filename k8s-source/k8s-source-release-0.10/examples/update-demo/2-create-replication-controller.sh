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

set -o errexit
set -o nounset
set -o pipefail

if [[ "${DOCKER_HUB_USER+set}" != "set" ]] ; then
  echo "Please set DOCKER_HUB_USER to your Docker hub account"
  exit 1
fi

export KUBE_REPO_ROOT=${KUBE_REPO_ROOT-$(dirname $0)/../..}
export KUBECTL=${KUBE_REPO_ROOT}/cluster/kubectl.sh

set -x

SCHEMA=${KUBE_REPO_ROOT}/examples/update-demo/nautilus-rc.yaml

cat ${SCHEMA} | sed "s/DOCKER_HUB_USER/${DOCKER_HUB_USER}/" | ${KUBECTL} create -f -
