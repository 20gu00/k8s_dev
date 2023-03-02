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

kube::util::sortable_date() {
  date "+%Y%m%d-%H%M%S"
}

kube::util::wait_for_url() {
  local url=$1
  local prefix=${2:-}
  local wait=${3:-0.2}
  local times=${4:-10}

  local i
  for i in $(seq 1 $times); do
    local out
    if out=$(curl -fs $url 2>/dev/null); then
      kube::log::status ${prefix}${out}
      return 0
    fi
    sleep $wait
  done
  kube::log::error "Timed out waiting for ${url}"
  return 1
}

# This figures out the host platform without relying on golang.  We need this as
# we don't want a golang install to be a prerequisite to building yet we need
# this info to figure out where the final binaries are placed.
kube::util::host_platform() {
  local host_os
  local host_arch
  case "$(uname -s)" in
    Darwin)
      host_os=darwin
      ;;
    Linux)
      host_os=linux
      ;;
    *)
      kube::log::error "Unsupported host OS.  Must be Linux or Mac OS X."
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
      kube::log::error "Unsupported host arch. Must be x86_64, 386 or arm."
      exit 1
      ;;
  esac
  echo "${host_os}/${host_arch}"
}
