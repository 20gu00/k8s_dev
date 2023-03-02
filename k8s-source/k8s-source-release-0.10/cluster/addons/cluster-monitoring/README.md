# Heapster

Heapster enables monitoring of Kubernetes Clusters using [cAdvisor](https://github.com/google/cadvisor). The kubelet will communicate with an instance of cAdvisor running on localhost and proxy container stats to Heapster. Kubelet will attempt to connect to cAdvisor on port 4194 by default but this port can be configured with kubelet's `-cadvisor_port` run flag. Detailed information about heapster can be found [here](https://github.com/GoogleCloudPlatform/heapster).
