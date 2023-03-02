## Getting started with Vagrant

### Prerequisites
1. Install latest version >= 1.6.2 of vagrant from http://www.vagrantup.com/downloads.html
2. Install latest version of Virtual Box from https://www.virtualbox.org/wiki/Downloads
3. Install the `net-tools` package for your distribution for VirtualBox's private networks.
4. Get or build a [binary release](binary_release.md)

### Setup

By default, the Vagrant setup will create a single kubernetes-master and 1 kubernetes-minion. Each VM will take 512 MB, so make sure you have at least 2 GB of free memory. To start your local cluster, open a shell and run:

```
cd kubernetes

export KUBERNETES_PROVIDER=vagrant
cluster/kube-up.sh
```

The `KUBERNETES_PROVIDER` environment variable tells all of the various cluster management scripts which variant to use.  If you forget to set this, the assumption is you are running on Google Compute Engine.

Vagrant will provision each machine in the cluster with all the necessary components to run Kubernetes.  The initial setup can take a few minutes to complete on each machine.

By default, each VM in the cluster is running Fedora, and all of the Kubernetes services are installed into systemd.

To access the master or any minion:

```
vagrant ssh master
vagrant ssh minion-1
```

If you are running more than one minion, you can access the others by:

```
vagrant ssh minion-2
vagrant ssh minion-3
```

To view the service status and/or logs on the kubernetes-master:
```
vagrant ssh master
[vagrant@kubernetes-master ~] $ sudo systemctl status kube-apiserver
[vagrant@kubernetes-master ~] $ sudo journalctl -r -u kube-apiserver

[vagrant@kubernetes-master ~] $ sudo systemctl status kube-controller-manager
[vagrant@kubernetes-master ~] $ sudo journalctl -r -u kube-controller-manager

[vagrant@kubernetes-master ~] $ sudo systemctl status etcd
[vagrant@kubernetes-master ~] $ sudo systemctl status nginx
```

To view the services on any of the kubernetes-minion(s):
```
vagrant ssh minion-1
[vagrant@kubernetes-minion-1] $ sudo systemctl status docker
[vagrant@kubernetes-minion-1] $ sudo journalctl -r -u docker
[vagrant@kubernetes-minion-1] $ sudo systemctl status kubelet
[vagrant@kubernetes-minion-1] $ sudo journalctl -r -u kubelet
```

### Interacting with your Kubernetes cluster with Vagrant.

With your Kubernetes cluster up, you can manage the nodes in your cluster with the regular Vagrant commands.

To push updates to new Kubernetes code after making source changes:
```
vagrant provision
```

To stop and then restart the cluster:
```
vagrant halt
vagrant up
```

To destroy the cluster:
```
vagrant destroy
```

Once your Vagrant machines are up and provisioned, the first thing to do is to check that you can use the `kubectl.sh` script.

You may need to build the binaries first, you can do this with ```make```

```
$ ./cluster/kubectl.sh get minions

NAME                LABELS
10.245.1.4          <none>
10.245.1.5          <none>
10.245.1.3          <none>

```

### Interacting with your Kubernetes cluster with the `kube-*` scripts.

Alternatively to using the vagrant commands, you can also use the `cluster/kube-*.sh` scripts to interact with the vagrant based provider just like any other hosting platform for kubernetes.

All of these commands assume you have set `KUBERNETES_PROVIDER` appropriately:

```
export KUBERNETES_PROVIDER=vagrant
```

Bring up a vagrant cluster

```
cluster/kube-up.sh
```

Destroy the vagrant cluster

```
cluster/kube-down.sh
```

Update the vagrant cluster after you make changes (only works when building your own releases locally):

```
cluster/kube-push.sh
```

Interact with the cluster

```
cluster/kubectl.sh
```

### Authenticating with your master

When using the vagrant provider in Kubernetes, the `cluster/kubectl.sh` script will cache your credentials in a `~/.kubernetes_vagrant_auth` file so you will not be prompted for them in the future.

```
cat ~/.kubernetes_vagrant_auth
{ "User": "vagrant",
  "Password": "vagrant"
  "CAFile": "/home/k8s_user/.kubernetes.vagrant.ca.crt",
  "CertFile": "/home/k8s_user/.kubecfg.vagrant.crt",
  "KeyFile": "/home/k8s_user/.kubecfg.vagrant.key"
}
```

You should now be set to use the `cluster/kubectl.sh` script. For example try to list the minions that you have started with:

```
cluster/kubectl.sh get minions
```

### Running containers

Your cluster is running, you can list the minions in your cluster:

```
$ cluster/kubectl.sh get minions

NAME                 LABELS
10.245.2.4           <none>
10.245.2.3           <none>
10.245.2.2           <none>

```

Now start running some containers!

You can now use any of the cluster/kube-*.sh commands to interact with your VM machines.
Before starting a container there will be no pods, services and replication controllers.

```
$ cluster/kubectl.sh get pods
NAME   IMAGE(S)   HOST   LABELS   STATUS

$ cluster/kubectl.sh get services
NAME   LABELS   SELECTOR   IP   PORT

$ cluster/kubectl.sh get replicationControllers
NAME   IMAGE(S   SELECTOR   REPLICAS
```

Start a container running nginx with a replication controller and three replicas
(note that this step uses the `kubecfg.sh` command instead of `kubectl.sh`):

```
$ cluster/kubecfg.sh -p 8080:80 run dockerfile/nginx 3 myNginx
```

When listing the pods, you will see that three containers have been started and are in Waiting state:

```
$ cluster/kubectl.sh get pods
NAME                                   IMAGE(S)            HOST                    LABELS         STATUS
781191ff-3ffe-11e4-9036-0800279696e1   dockerfile/nginx    10.245.2.4/10.245.2.4   name=myNginx   Waiting
7813c8bd-3ffe-11e4-9036-0800279696e1   dockerfile/nginx    10.245.2.2/10.245.2.2   name=myNginx   Waiting
78140853-3ffe-11e4-9036-0800279696e1   dockerfile/nginx    10.245.2.3/10.245.2.3   name=myNginx   Waiting
```

You need to wait for the provisioning to complete, you can monitor the minions by doing:

```
$ sudo salt '*minion-1' cmd.run 'docker images'
kubernetes-minion-1:
    REPOSITORY          TAG                 IMAGE ID            CREATED             VIRTUAL SIZE
    <none>              <none>              96864a7d2df3        26 hours ago        204.4 MB
    google/cadvisor     latest              e0575e677c50        13 days ago         12.64 MB
    kubernetes/pause    latest              6c4579af347b        8 weeks ago         239.8 kB
```

Once the docker image for nginx has been downloaded, the container will start and you can list it:

```
$ sudo salt '*minion-1' cmd.run 'docker ps'
kubernetes-minion-1:
    CONTAINER ID        IMAGE                     COMMAND                CREATED             STATUS              PORTS                    NAMES
    dbe79bf6e25b        dockerfile/nginx:latest   "nginx"                21 seconds ago      Up 19 seconds                                k8s--mynginx.8c5b8a3a--7813c8bd_-_3ffe_-_11e4_-_9036_-_0800279696e1.etcd--7813c8bd_-_3ffe_-_11e4_-_9036_-_0800279696e1--fcfa837f
    fa0e29c94501        kubernetes/pause:latest   "/pause"               8 minutes ago       Up 8 minutes        0.0.0.0:8080->80/tcp     k8s--net.a90e7ce4--7813c8bd_-_3ffe_-_11e4_-_9036_-_0800279696e1.etcd--7813c8bd_-_3ffe_-_11e4_-_9036_-_0800279696e1--baf5b21b
    aa2ee3ed844a        google/cadvisor:latest    "/usr/bin/cadvisor -   38 minutes ago      Up 38 minutes                                k8s--cadvisor.9e90d182--cadvisor_-_agent.file--4626b3a2
    65a3a926f357        kubernetes/pause:latest   "/pause"               39 minutes ago      Up 39 minutes       0.0.0.0:4194->8080/tcp   k8s--net.c5ba7f0e--cadvisor_-_agent.file--342fd561
```

Going back to listing the pods, services and replicationControllers, you now have:

```
$ cluster/kubectl.sh get pods
NAME                                   IMAGE(S)            HOST                    LABELS         STATUS
781191ff-3ffe-11e4-9036-0800279696e1   dockerfile/nginx    10.245.2.4/10.245.2.4   name=myNginx   Running
7813c8bd-3ffe-11e4-9036-0800279696e1   dockerfile/nginx    10.245.2.2/10.245.2.2   name=myNginx   Running
78140853-3ffe-11e4-9036-0800279696e1   dockerfile/nginx    10.245.2.3/10.245.2.3   name=myNginx   Running

$ cluster/kubectl.sh get services
NAME   LABELS   SELECTOR   IP   PORT

$ cluster/kubectl.sh get replicationControllers
NAME      IMAGE(S            SELECTOR       REPLICAS
myNginx   dockerfile/nginx   name=myNginx   3
```

We did not start any services, hence there are none listed. But we see three replicas displayed properly.
Check the [guestbook](../../examples/guestbook/README.md) application to learn how to create a service.
You can already play with resizing the replicas with:

```
$ cluster/kubecfg.sh resize myNginx 2
$ cluster/kubectl.sh get pods
NAME                                   IMAGE(S)            HOST                    LABELS         STATUS
7813c8bd-3ffe-11e4-9036-0800279696e1   dockerfile/nginx    10.245.2.2/10.245.2.2   name=myNginx   Running
78140853-3ffe-11e4-9036-0800279696e1   dockerfile/nginx    10.245.2.3/10.245.2.3   name=myNginx   Running
```

Congratulations!

### Testing

The following will run all of the end-to-end testing scenarios assuming you set your environment in cluster/kube-env.sh

```
NUM_MINIONS=3 hack/e2e-test.sh
```

### Troubleshooting

#### I keep downloading the same (large) box all the time!

By default the Vagrantfile will download the box from S3.  You can change this (and cache the box locally) by providing an alternate URL when calling `kube-up.sh`

```bash
export KUBERNETES_BOX_URL=path_of_your_kuber_box
export KUBERNETES_PROVIDER=vagrant
cluster/kube-up.sh
```


#### I just created the cluster, but I am getting authorization errors!

You probably have an incorrect ~/.kubernetes_vagrant_auth file for the cluster you are attempting to contact.

```
rm ~/.kubernetes_vagrant_auth
```

After using kubecfg.sh make sure that the correct credentials are set:

```
cat ~/.kubernetes_vagrant_auth
{
  "User": "vagrant",
  "Password": "vagrant"
}
```

#### I just created the cluster, but I do not see my container running !

If this is your first time creating the cluster, the kubelet on each minion schedules a number of docker pull requests to fetch prerequisite images.  This can take some time and as a result may delay your initial pod getting provisioned.

#### I changed Kubernetes code, but it's not running !

Are you sure there was no build error?  After running `$ vagrant provision`, scroll up and ensure that each Salt state was completed successfully on each box in the cluster.
It's very likely you see a build error due to an error in your source files!

#### I have brought Vagrant up but the minions won't validate !

Are you sure you built a release first? Did you install `net-tools`? For more clues, login to one of the minions (`vagrant ssh minion-1`) and inspect the salt minion log (`sudo cat /var/log/salt/minion`).

#### I want to change the number of minions !

You can control the number of minions that are instantiated via the environment variable `NUM_MINIONS` on your host machine.  If you plan to work with replicas, we strongly encourage you to work with enough minions to satisfy your largest intended replica size.  If you do not plan to work with replicas, you can save some system resources by running with a single minion. You do this, by setting `NUM_MINIONS` to 1 like so:

```
export NUM_MINIONS=1
```

#### I ran vagrant suspend and nothing works!
```vagrant suspend``` seems to mess up the network.  It's not supported at this time.
