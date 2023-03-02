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

# TODO(jbeda): Provide a way to override project
# gcloud multiplexing for shared GCE/GKE tests.
GCLOUD=gcloud
ZONE=${KUBE_GCE_ZONE:-us-central1-b}
MASTER_SIZE=n1-standard-1
MINION_SIZE=n1-standard-1
NUM_MINIONS=${NUM_MINIONS:-4}
MINION_DISK_TYPE=pd-standard
MINION_DISK_SIZE=10GB
# TODO(dchen1107): Filed an internal issue to create an alias
# for containervm image, so that gcloud will expand this
# to the latest supported image.
IMAGE=container-vm-v20150112
IMAGE_PROJECT=google-containers
NETWORK=${KUBE_GCE_NETWORK:-default}
INSTANCE_PREFIX="${KUBE_GCE_INSTANCE_PREFIX:-kubernetes}"
MASTER_NAME="${INSTANCE_PREFIX}-master"
MASTER_TAG="${INSTANCE_PREFIX}-master"
MINION_TAG="${INSTANCE_PREFIX}-minion"
MINION_NAMES=($(eval echo ${INSTANCE_PREFIX}-minion-{1..${NUM_MINIONS}}))

# Compute IP addresses for nodes.
function increment_ipv4 {
  local ip_base=$1
  local incr_amount=$2
  local -a ip_components
  local ip_regex="([0-9]+).([0-9]+).([0-9]+).([0-9]+)"
  [[ $ip_base =~ $ip_regex ]]
  ip_components=("${BASH_REMATCH[1]}" "${BASH_REMATCH[2]}" "${BASH_REMATCH[3]}" "${BASH_REMATCH[4]}")
  ip_dec=0
  local comp
  for comp in "${ip_components[@]}"; do
    ip_dec=$((ip_dec<<8))
    ip_dec=$((ip_dec + $comp))
  done
 
  ip_dec=$((ip_dec + $incr_amount))
 
  ip_components=()
  local i
  for ((i=0; i < 4; i++)); do
    comp=$((ip_dec & 0xFF))
    ip_components+=($comp)
    ip_dec=$((ip_dec>>8))
  done
  echo "${ip_components[3]}.${ip_components[2]}.${ip_components[1]}.${ip_components[0]}"
}
 
node_count="${NUM_MINIONS}"
next_node="10.244.0.0"
node_subnet_size=24
node_subnet_count=$((2 ** (32-$node_subnet_size)))
subnets=()
 
for ((node_num=0; node_num<node_count; node_num++)); do
  subnets+=("$next_node"/"${node_subnet_size}")
  next_node=$(increment_ipv4 $next_node $node_subnet_count)
done

CLUSTER_IP_RANGE="10.244.0.0/16"
MINION_IP_RANGES=($(eval echo "${subnets[@]}"))

MINION_SCOPES=("storage-ro" "compute-rw")
# Increase the sleep interval value if concerned about API rate limits. 3, in seconds, is the default.
POLL_SLEEP_INTERVAL=3
PORTAL_NET="10.0.0.0/16"

# Optional: Install node monitoring.
ENABLE_NODE_MONITORING=true

# Optional: When set to true, heapster will be setup as part of the cluster bring up.
ENABLE_CLUSTER_MONITORING=true

# When set to true, Docker Cache is enabled by default as part of the cluster bring up.
ENABLE_DOCKER_REGISTRY_CACHE=true

# Optional: Enable node logging.
ENABLE_NODE_LOGGING=true
LOGGING_DESTINATION=elasticsearch # options: elasticsearch, gcp

# Optional: When set to true, Elasticsearch and Kibana will be setup as part of the cluster bring up.
ENABLE_CLUSTER_LOGGING=true
ELASTICSEARCH_LOGGING_REPLICAS=1

# Don't require https for registries in our local RFC1918 network
EXTRA_DOCKER_OPTS="--insecure-registry 10.0.0.0/8"

# Optional: Install cluster DNS.
ENABLE_CLUSTER_DNS=true
DNS_SERVER_IP="10.0.0.10"
DNS_DOMAIN="kubernetes.local"
DNS_REPLICAS=1
