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

KUBE_ROOT=$(dirname "${BASH_SOURCE}")/..
source "${KUBE_ROOT}/cluster/kube-env.sh"
source "${KUBE_ROOT}/cluster/${KUBERNETES_PROVIDER}/util.sh"

# Detect the OS name/arch so that we can find our binary
case "$(uname -s)" in
  Darwin)
    host_os=darwin
    ;;
  Linux)
    host_os=linux
    ;;
  *)
    echo "Unsupported host OS.  Must be Linux or Mac OS X." >&2
    exit 1
    ;;
esac

case "$(uname -m)" in
  x86_64*)
    host_arch=amd64
    ;;
  i?86_64*)
    host_arch=amd64
    ;;
  amd64*)
    host_arch=amd64
    ;;
  arm*)
    host_arch=arm
    ;;
  i?86*)
    host_arch=x86
    ;;
  *)
    echo "Unsupported host arch. Must be x86_64, 386 or arm." >&2
    exit 1
    ;;
esac

# Gather up the list of likely places and use ls to find the latest one.
locations=(
  "${KUBE_ROOT}/_output/dockerized/bin/${host_os}/${host_arch}/kubecfg"
  "${KUBE_ROOT}/_output/local/bin/${host_os}/${host_arch}/kubecfg"
  "${KUBE_ROOT}/platforms/${host_os}/${host_arch}/kubecfg"
)
kubecfg=$( (ls -t "${locations[@]}" 2>/dev/null || true) | head -1 )

if [[ ! -x "$kubecfg" ]]; then
  {
    echo "It looks as if you don't have a compiled kubecfg binary."
    echo
    echo "If you are running from a clone of the git repo, please run"
    echo "'./build/run.sh hack/build-cross.sh'. Note that this requires having"
    echo "Docker installed."
    echo
    echo "If you are running from a binary release tarball, something is wrong. "
    echo "Look at http://kubernetes.io/ for information on how to contact the "
    echo "development team for help."
  } >&2
  exit 1
fi

# When we are using vagrant it has hard coded auth.  We repeat that here so that
# we don't clobber auth that might be used for a publicly facing cluster.
if [[ "$KUBERNETES_PROVIDER" == "vagrant" ]]; then
  auth_config=(
    "-auth" "$HOME/.kubernetes_vagrant_auth"
  )
elif [[ "${KUBERNETES_PROVIDER}" == "gke" ]]; then
  # While KUBECFG calls remain, we manually pass the auth and cert info.
  detect-project &> /dev/null
  cluster_config_dir="${GCLOUD_CONFIG_DIR}/${PROJECT}.${ZONE}.${CLUSTER_NAME}"
  auth_config=(
    "-certificate_authority=${cluster_config_dir}/ca.crt"
    "-client_certificate=${cluster_config_dir}/kubecfg.crt"
    "-client_key=${cluster_config_dir}/kubecfg.key"
    "-auth=${cluster_config_dir}/kubernetes_auth"
  )
else
  auth_config=()
fi

detect-master > /dev/null
if [[ -n "${KUBE_MASTER_IP-}" && -z "${KUBERNETES_MASTER-}" ]]; then
  export KUBERNETES_MASTER=https://${KUBE_MASTER_IP}
fi

"${kubecfg}" "${auth_config[@]:+${auth_config[@]}}" "${@}"
