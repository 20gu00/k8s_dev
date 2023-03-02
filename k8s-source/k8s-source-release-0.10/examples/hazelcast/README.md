## Cloud Native Deployments of Hazelcast using Kubernetes

The following document describes the development of a _cloud native_ [Hazelcast](http://hazelcast.org/) deployment on Kubernetes.  When we say _cloud native_ we mean an application which understands that it is running within a cluster manager, and uses this cluster management infrastructure to help implement the application. In particular, in this instance, a custom Hazelcast ```bootstrapper``` is used to enable Hazelcast to dynamically discover Hazelcast nodes that have already joined the cluster.

Any topology changes are communicated and handled by Hazelcast nodes themselves.

This document also attempts to describe the core components of Kubernetes, _Pods_, _Services_ and _Replication Controllers_.

### Prerequisites
This example assumes that you have a Kubernetes cluster installed and running, and that you have installed the ```kubectl``` command line tool somewhere in your path.  Please see the [getting started](https://github.com/GoogleCloudPlatform/kubernetes/tree/master/docs/getting-started-guides) for installation instructions for your platform.

### A note for the impatient
This is a somewhat long tutorial.  If you want to jump straight to the "do it now" commands, please see the [tl; dr](#tl-dr) at the end.

### Sources

Source is freely available at:
* Docker image - https://github.com/pires/hazelcast-kubernetes
* Hazelcast Discovery - https://github.com/pires/hazelcast-kubernetes-bootstrapper
* Docker Trusted Build - https://registry.hub.docker.com/u/pires/hazelcast-k8s

### Simple Single Pod Hazelcast Node
In Kubernetes, the atomic unit of an application is a [_Pod_](https://github.com/GoogleCloudPlatform/kubernetes/blob/master/docs/pods.md).  A Pod is one or more containers that _must_ be scheduled onto the same host.  All containers in a pod share a network namespace, and may optionally share mounted volumes.  In this simple case, we define a single container running Hazelcast for our pod:

```yaml
id: hazelcast
kind: Pod
apiVersion: v1beta1
desiredState:
  manifest:
    version: v1beta1
    id: hazelcast
    containers:
      - name: hazelcast
        image: pires/hazelcast-k8s
        cpu: 1000
        ports:
          - name: hazelcast
            containerPort: 5701
labels:
  name: hazelcast
```

There are a few things to note in this description.  First is that we are running the ```pires/hazelcast-k8s``` image.  This is a standard Ubuntu 14.04 installation with Java 8.  However it also adds a custom [```application ```](https://github.com/pires/hazelcast-kubernetes-bootstrapper) that finds any Hazelcast nodes in the cluster and bootstraps an Hazelcast instance.  The ```HazelcastDiscoveryController``` discovers the Kubernetes API Server using the built in Kubernetes discovery service, and then uses the Kubernetes API to find new nodes (more on this later).

You may also note that we tell Kubernetes that the container exposes the ```Hazelcast``` port.  Finally, we tell the cluster manager that we need 1000 milli-cpus (1 core).

Given this configuration, we can create the pod as follows:

```sh
$ kubectl create -f hazelcast.yaml
```

After a few moments, you should be able to see the pod running:

```sh
$ kubectl get pods hazelcast
POD                 CONTAINER(S)        IMAGE(S)               HOST                                                          LABELS              STATUS
hazelcast           hazelcast           pires/hazelcast-k8s    kubernetes-minion-1/1.2.3.4                                    name=hazelcast      Running
```


### Adding a Hazelcast Service
In Kubernetes a _Service_ describes a set of Pods that perform the same task.  For example, the set of nodes in a Hazelcast cluster, or even the single node we created above.  An important use for a Service is to create a load balancer which distributes traffic across members of the set.  But a _Service_ can also be used as a standing query which makes a dynamically changing set of Pods (or the single Pod we've already created) available via the Kubernetes API.  This is the way that we use initially use Services with Hazelcast.

Here is the service description:
```yaml
id: hazelcast
kind: Service
apiVersion: v1beta1
port: 5701
containerPort: 5701
selector:
  name: hazelcast
```

The important thing to note here is the ```selector```. It is a query over labels, that identifies the set of _Pods_ contained by the _Service_.  In this case the selector is ```name=hazelcast```.  If you look back at the Pod specification above, you'll see that the pod has the corresponding label, so it will be selected for membership in this Service.

Create this service as follows:
```sh
$ kubectl create -f hazelcast-service.yaml
```

Once the service is created, you can query it's endpoints:
```sh
$ kubectl get endpoints hazelcast -o yaml
apiVersion: v1beta1
creationTimestamp: 2015-01-05T05:51:50Z
endpoints:
- xxx.xxx.xxx.xxx:5701
id: hazelcast
kind: Endpoints
namespace: default
resourceVersion: 69130
selfLink: /api/v1beta1/endpoints/hazelcast?namespace=default
uid: f1937b47-949e-11e4-8a8b-42010af0e23e
```

You can see that the _Service_ has found the pod we created in step one.

### Adding replicated nodes
Of course, a single node cluster isn't particularly interesting.  The real power of Kubernetes and Hazelcast lies in easily building a replicated, resizable Hazelcast cluster.

In Kubernetes a _Replication Controller_ is responsible for replicating sets of identical pods.  Like a _Service_ it has a selector query which identifies the members of it's set.  Unlike a _Service_ it also has a desired number of replicas, and it will create or delete _Pods_ to ensure that the number of _Pods_ matches up with it's desired state.

Replication Controllers will "adopt" existing pods that match their selector query, so let's create a Replication Controller with a single replica to adopt our existing Hazelcast Pod.

```yaml
id: hazelcast
kind: ReplicationController
apiVersion: v1beta1
desiredState:
  replicas: 1
  replicaSelector:
    name: hazelcast
  # This is identical to the pod config above
  podTemplate:
    desiredState:
      manifest:
        version: v1beta1
        id: hazelcast
        containers:
          - name: hazelcast
            image: pires/hazelcast-k8s
            cpu: 1000
            ports:
              - name: hazelcast
                containerPort: 5701
    labels:
      name: hazelcast
```

The bulk of the replication controller config is actually identical to the Hazelcast pod declaration above, it simply gives the controller a recipe to use when creating new pods.  The other parts are the ```replicaSelector``` which contains the controller's selector query, and the ```replicas``` parameter which specifies the desired number of replicas, in this case 1.

Create this controller:

```sh
$ kubectl create -f hazelcast-controller.yaml
```

Now this is actually not that interesting, since we haven't actually done anything new.  Now it will get interesting.

Let's resize our cluster to 2:
```sh
$ kubectl resize rc hazelcast --replicas=2
```

Now if you list the pods in your cluster, you should see two hazelcast pods:

```sh
$ kubectl get pods
POD                                    CONTAINER(S)        IMAGE(S)                       HOST                                                          LABELS              STATUS
hazelcast                              hazelcast           pires/hazelcast-k8s            kubernetes-minion-1.c.my-cloud-code.internal/1.2.3.4          name=hazelcast      Running
16b2beab-94a1-11e4-8a8b-42010af0e23e   hazelcast           pires/hazelcast-k8s            kubernetes-minion-3.c.my-cloud-code.internal/2.3.4.5          name=hazelcast      Running
```

Notice that one of the pods has the human readable name ```hazelcast``` that you specified in your config before, and one has a random string, since it was named by the replication controller.

To prove that this all works, you can use the ```log``` command to examine the logs of one pod, for example:

```sh
$ kubectl log 16b2beab-94a1-11e4-8a8b-42010af0e23e hazelcast
2014-12-24T01:21:09.731468790Z 2014-12-24 01:21:09.701  INFO 10 --- [           main] c.g.p.h.HazelcastDiscoveryController     : Asking k8s registry at http://10.160.211.80:80..
2014-12-24T01:21:13.686978543Z 2014-12-24 01:21:13.686  INFO 10 --- [           main] c.g.p.h.HazelcastDiscoveryController     : Found 3 pods running Hazelcast.
2014-12-24T01:21:13.772599736Z 2014-12-24 01:21:13.772  INFO 10 --- [           main] c.g.p.h.HazelcastDiscoveryController     : Added member 10.160.2.3
2014-12-24T01:21:13.783689690Z 2014-12-24 01:21:13.783  INFO 10 --- [           main] c.g.p.h.HazelcastDiscoveryController     : Added member 10.160.2.4

(...)

2014-12-24T01:21:16.007729519Z 2014-12-24 01:21:16.000  INFO 10 --- [cached.thread-3] c.h.nio.tcp.TcpIpConnectionManager       : [10.160.2.4]:5701 [someGroup] [3.3.3] Established socket connection between /10.160.2.4:54931 and /10.160.2.3:5701
2014-12-24T01:21:16.427289059Z 2014-12-24 01:21:16.427  INFO 10 --- [thread-Acceptor] com.hazelcast.nio.tcp.SocketAcceptor     : [10.160.2.4]:5701 [someGroup] [3.3.3] Accepting socket connection from /10.160.2.3:50660
2014-12-24T01:21:16.433763738Z 2014-12-24 01:21:16.433  INFO 10 --- [cached.thread-3] c.h.nio.tcp.TcpIpConnectionManager       : [10.160.2.4]:5701 [someGroup] [3.3.3] Established socket connection between /10.160.2.4:5701 and /10.160.2.3:50660
2014-12-24T01:21:23.036227250Z 2014-12-24 01:21:23.035  INFO 10 --- [ration.thread-1] com.hazelcast.cluster.ClusterService     : [10.160.2.4]:5701 [someGroup] [3.3.3]
2014-12-24T01:21:23.036227250Z
2014-12-24T01:21:23.036227250Z Members [3] {
2014-12-24T01:21:23.036227250Z  Member [10.160.2.4]:5701 this
2014-12-24T01:21:23.036227250Z  Member [10.160.2.3]:5701
2014-12-24T01:21:23.036227250Z }
```

Now let's resize our cluster to 4 nodes:
```sh
$ kubectl resize rc hazelcast --replicas=4
```

Examine the status again by checking a node’s log and you should see the 4 members connected.

### tl; dr;
For those of you who are impatient, here is the summary of the commands we ran in this tutorial.

```sh
# create a single hazelcast node
kubectl create -f hazelcast.yaml

# create a service to track all hazelcast nodes
kubectl create -f hazelcast-service.yaml

# create a replication controller to replicate hazelcast nodes
kubectl create -f hazelcast-controller.yaml

# scale up to 2 nodes
kubectl resize rc hazelcast --replicas=2

# validate the cluster
docker exec <container-id> nodetool status

# scale up to 4 nodes
kubectl resize rc hazelcast --replicas=4
```

### Hazelcast Discovery Source

```java
import com.hazelcast.config.Config;
import com.hazelcast.config.GroupConfig;
import com.hazelcast.config.JoinConfig;
import com.hazelcast.config.MulticastConfig;
import com.hazelcast.config.NetworkConfig;
import com.hazelcast.config.SSLConfig;
import com.hazelcast.config.TcpIpConfig;
import com.hazelcast.core.Hazelcast;
import io.fabric8.kubernetes.api.KubernetesClient;
import io.fabric8.kubernetes.api.KubernetesFactory;
import io.fabric8.kubernetes.api.model.Pod;
import java.util.List;
import java.util.UUID;
import java.util.concurrent.CopyOnWriteArrayList;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.boot.CommandLineRunner;
import org.springframework.stereotype.Controller;

/**
 * Read from Kubernetes API all labeled Hazelcast pods, get their IP and connect to them.
 */
@Controller
public class HazelcastDiscoveryController implements CommandLineRunner {

  private static final Logger log = LoggerFactory.getLogger(
      HazelcastDiscoveryController.class);

  private static final String HAZELCAST_LABEL_NAME = "name";
  private static final String HAZELCAST_LABEL_VALUE = "hazelcast";

  private static String getEnvOrDefault(String var, String def) {
    final String val = System.getenv(var);
    return (val == null || val.isEmpty())
        ? def
        : val;
  }

  @Override
  public void run(String... args) {
    final String kubeApiHost = getEnvOrDefault("KUBERNETES_RO_SERVICE_HOST",
        "localhost");
    final String kubeApiPort = getEnvOrDefault("KUBERNETES_RO_SERVICE_PORT",
        "8080");
    final String kubeUrl = "http://" + kubeApiHost + ":" + kubeApiPort;
    log.info("Asking k8s registry at {}..", kubeUrl);
    final KubernetesClient kube = new KubernetesClient(new KubernetesFactory(
        kubeUrl));
    final List<Pod> hazelcastPods = new CopyOnWriteArrayList<>();
    kube.getPods().getItems().parallelStream().filter(pod
        -> pod.getLabels().get(HAZELCAST_LABEL_NAME).equals(
            HAZELCAST_LABEL_VALUE)).forEach(hazelcastPods::add);
    log.info("Found {} pods running Hazelcast.", hazelcastPods.size());
    if (!hazelcastPods.isEmpty()) {
      runHazelcast(hazelcastPods);
    }
  }

  private void runHazelcast(final List<Pod> hazelcastPods) {
    // configure Hazelcast instance
    final Config cfg = new Config();
    cfg.setInstanceName(UUID.randomUUID().toString());
    // group configuration
    final String HC_GROUP_NAME = getEnvOrDefault("HC_GROUP_NAME", "someGroup");
    final String HC_GROUP_PASSWORD = getEnvOrDefault("HC_GROUP_PASSWORD",
        "someSecret");
    final int HC_PORT = Integer.parseInt(getEnvOrDefault("HC_PORT", "5701"));
    cfg.setGroupConfig(new GroupConfig(HC_GROUP_NAME, HC_GROUP_PASSWORD));
    // network configuration initialization
    final NetworkConfig netCfg = new NetworkConfig();
    netCfg.setPortAutoIncrement(false);
    netCfg.setPort(HC_PORT);
    // multicast
    final MulticastConfig mcCfg = new MulticastConfig();
    mcCfg.setEnabled(false);
    // tcp
    final TcpIpConfig tcpCfg = new TcpIpConfig();
    hazelcastPods.parallelStream().forEach(pod -> {
      tcpCfg.addMember(pod.getCurrentState().getPodIP());
    });
    tcpCfg.setEnabled(true);
    // network join configuration
    final JoinConfig joinCfg = new JoinConfig();
    joinCfg.setMulticastConfig(mcCfg);
    joinCfg.setTcpIpConfig(tcpCfg);
    netCfg.setJoin(joinCfg);
    // ssl
    netCfg.setSSLConfig(new SSLConfig().setEnabled(false));
    // set it all
    cfg.setNetworkConfig(netCfg);
    // run
    Hazelcast.newHazelcastInstance(cfg);
  }

}
```
