
# Persistent Installation of MySQL and WordPress on Kubernetes

This example describes how to run a persistent installation of [Wordpress](https://wordpress.org/).

We'll use the [mysql](https://registry.hub.docker.com/_/mysql/) and [wordpress](https://registry.hub.docker.com/_/wordpress/) official [Docker](https://www.docker.com/) images for this installation. (The wordpress image includes an Apache server).

We'll create two Kubernetes [pods](https://github.com/GoogleCloudPlatform/kubernetes/blob/master/docs/pods.md) to run mysql and wordpress, both with associated [persistent disks](https://cloud.google.com/compute/docs/disks), then set up a Kubernetes [service](https://github.com/GoogleCloudPlatform/kubernetes/blob/master/docs/services.md) to front each pod.

This example demonstrates several useful things, including: how to set up and use persistent disks with Kubernetes pods; how to define Kubernetes services to leverage docker-links-compatible service environment variables; and use of an external load balancer to expose the wordpress service externally and make it transparent to the user if the wordpress pod moves to a different cluster node.

Some of the example's details, such as the Persistent Disk setup, require that Kubernetes is running on [Google Compute Engine](https://cloud.google.com/compute/).


## Install gcloud and start up a Kubernetes cluster

First, if you have not already done so, [create](https://cloud.google.com/compute/docs/quickstart) a [Google Cloud Platform](https://cloud.google.com/) project, and install the [gcloud SDK](https://cloud.google.com/sdk/).

Then, set the gcloud default project name to point to the project you want to use for your Kubernetes cluster:

```
gcloud config set project <project-name>
```

Next, grab the Kubernetes [release binary](https://github.com/GoogleCloudPlatform/kubernetes/releases). You can do this via:
```shell
wget -q -O - https://get.k8s.io | sh
```
or
```shell
curl -sS https://get.k8s.io | sh
```

Then, start up a Kubernetes cluster as [described here](https://github.com/GoogleCloudPlatform/kubernetes/blob/master/docs/getting-started-guides/gce.md).

```
$ <kubernetes>/cluster/kube-up.sh
```

where `<kubernetes>` is the path to your Kubernetes installation.

## Create and format two persistent disks

For this WordPress installation, we're going to configure our Kubernetes [pods](https://github.com/GoogleCloudPlatform/kubernetes/blob/master/docs/pods.md) to use [persistent disks](https://cloud.google.com/compute/docs/disks). This means that we can preserve installation state across pod shutdown and re-startup.

Before doing anything else, we'll create and format the persistent disks that we'll use for the installation: one for the mysql pod, and one for the wordpress pod.
The general series of steps required is as described [here](https://github.com/GoogleCloudPlatform/kubernetes/blob/master/docs/volumes.md), where $ZONE is the zone where your cluster is running, and $DISK_SIZE is specified as, e.g. '500GB'.  In future, this process will be more streamlined.

So for the two disks used in this example, do the following.
First create and format the mysql disk, setting the disk size to meet your needs:

```shell
gcloud compute disks create --size=$DISK_SIZE --zone=$ZONE mysql-disk
gcloud compute instances attach-disk --zone=$ZONE --disk=mysql-disk --device-name temp-data kubernetes-master
gcloud compute ssh --zone=$ZONE kubernetes-master \
  --command "sudo mkdir -p /mnt/tmp && sudo /usr/share/google/safe_format_and_mount /dev/disk/by-id/google-temp-data /mnt/tmp"
gcloud compute instances detach-disk --zone=$ZONE --disk mysql-disk kubernetes-master
```

Then create and format the wordpress disk.  Note that you may not want as large a disk size for the wordpress code as for the mysql disk.

```shell
gcloud compute disks create --size=$DISK_SIZE --zone=$ZONE wordpress-disk
gcloud compute instances attach-disk --zone=$ZONE --disk=$wordpress-disk --device-name temp-data kubernetes-master
gcloud compute ssh --zone=$ZONE kubernetes-master \
  --command "sudo mkdir -p /mnt/tmp && sudo /usr/share/google/safe_format_and_mount /dev/disk/by-id/google-temp-data /mnt/tmp"
gcloud compute instances detach-disk --zone=$ZONE --disk wordpress-disk kubernetes-master
```

## Start the Mysql Pod and Service

Now that the persistent disks are defined, the Kubernetes pods can be launched.  We'll start with the mysql pod.

### Start the Mysql pod

First, **edit `mysql.yaml`**, the mysql pod definition, to use a database password that you specify.
`mysql.yaml` looks like this:

```yaml
apiVersion: v1beta1
id: mysql
desiredState:
  manifest:
    version: v1beta1
    id: mysql
    containers:
      - name: mysql
        image: mysql
        env:
          - name: MYSQL_ROOT_PASSWORD
           # change this
            value: yourpassword
        cpu: 100
        ports:
          - containerPort: 3306
        volumeMounts:
            # name must match the volume name below
          - name: mysql-persistent-storage
            # mount path within the container
            mountPath: /var/lib/mysql
    volumes:
      - name: mysql-persistent-storage
        source:
          persistentDisk:
            # This GCE PD must already exist and be formatted ext4
            pdName: mysql-disk
            fsType: ext4
labels:
  name: mysql
kind: Pod
```

Note that we've defined a volume mount for `/var/lib/mysql`, and specified a volume that uses the persistent disk (`mysql-disk`) that you created.
Once you've edited the file to set your database password, create the pod as follows, where `<kubernetes>` is the path to your Kubernetes installation:

```shell
$ <kubernetes>/cluster/kubectl.sh create -f mysql.yaml
```

It may take a short period before the new pod reaches the `Running` state.
List all pods to see the status of this new pod and the cluster node that it is running on:

```shell
$ <kubernetes>/cluster/kubectl.sh get pods
```


#### Check the running pod on the Compute instance

You can take a look at the logs for a pod by using `kubectl.sh log`.  For example:

```shell
$ <kubernetes>/cluster/kubectl.sh log mysql
```

If you want to do deeper troubleshooting, e.g. if it seems a container is not staying up, you can also ssh in to the node that a pod is running on.  There, you can run `sudo -s`, then `docker ps -a` to see all the containers.  You can then inspect the logs of containers that have exited, via `docker logs <container_id>`.  (You can also find some relevant logs under `/var/log`, e.g. `docker.log` and `kubelet.log`).

### Start the Mysql service

We'll define and start a [service](https://github.com/GoogleCloudPlatform/kubernetes/blob/master/docs/services.md) that lets other pods access the mysql database on a known port and host.
We will specifically name the service `mysql`.  This will let us leverage the support for [Docker-links-compatible](https://github.com/GoogleCloudPlatform/kubernetes/blob/master/docs/services.md#how-do-they-work) service environment variables when we up the wordpress pod. The wordpress Docker image expects to be linked to a mysql container named `mysql`, as you can see in the "How to use this image" section on the wordpress docker hub [page](https://registry.hub.docker.com/_/wordpress/).

So if we label our Kubernetes mysql service `mysql`, the wordpress pod will be able to use the Docker-links-compatible environment variables, defined by Kubernetes, to connect to the database.

The `mysql-service.yaml` file looks like this:

```yaml
kind: Service
apiVersion: v1beta1
id: mysql
# the port that this service should serve on
port: 3306
# just like the selector in the replication controller,
# but this time it identifies the set of pods to load balance
# traffic to.
selector:
  name: mysql
# the container on each pod to connect to, can be a name
# (e.g. 'www') or a number (e.g. 80)
containerPort: 3306
labels:
  name: mysql
```

Start the service like this:

```shell
$ <kubernetes>/cluster/kubectl.sh create -f mysql-service.yaml
```

You can see what services are running via:

```shell
$ <kubernetes>/cluster/kubectl.sh get services
```


## Start the WordPress Pod and Service

Once the mysql service is up, start the wordpress pod, specified in
`wordpress.yaml`.  Before you start it, **edit `wordpress.yaml`** and **set the database password to be the same as you used in `mysql.yaml`**.
Note that this config file also defines a volume, this one using the `wordpress-disk` persistent disk that you created.

```yaml
apiVersion: v1beta1
id: wordpress
desiredState:
  manifest:
    version: v1beta1
    id: frontendController
    containers:
      - name: wordpress
        image: wordpress
        ports:
          - containerPort: 80
        volumeMounts:
            # name must match the volume name below
          - name: wordpress-persistent-storage
            # mount path within the container
            mountPath: /var/www/html
        env:
          - name: WORDPRESS_DB_PASSWORD
            # change this - must match mysql.yaml password
            value: yourpassword
    volumes:
      - name: wordpress-persistent-storage
        source:
          # emptyDir: {}
          persistentDisk:
            # This GCE PD must already exist and be formatted ext4
            pdName: wordpress-disk
            fsType: ext4
labels:
  name: wpfrontend
kind: Pod
```

Create the pod:

```shell
$ <kubernetes>/cluster/kubectl.sh create -f wordpress.yaml
```

And list the pods to check that the status of the new pod changes
to `Running`.  As above, this might take a minute.

```shell
$ <kubernetes>/cluster/kubectl.sh get pods
```

### Start the WordPress service

Once the wordpress pod is running, start its service, specified by `wordpress-service.yaml`.

The service config file looks like this:

```yaml
kind: Service
apiVersion: v1beta1
id: wpfrontend
# the port that this service should serve on.
port: 80
# just like the selector in the replication controller,
# but this time it identifies the set of pods to load balance
# traffic to.
selector:
  name: wpfrontend
# the container on each pod to connect to, can be a name
# (e.g. 'www') or a number (e.g. 80)
containerPort: 80
labels:
  name: wpfrontend
createExternalLoadBalancer: true
```

Note the `createExternalLoadBalancer` setting.  This will set up the wordpress service behind an external IP.
`createExternalLoadBalancer` only works on GCE.
Note also that we've set the service port to 80.  We'll return to that shortly.

Start the service:

```shell
$ <kubernetes>/cluster/kubectl.sh create -f wordpress-service.yaml
```

and see it in the list of services:

```shell
$ <kubernetes>/cluster/kubectl.sh get services
```

Then, find the external IP for your WordPress service by listing the forwarding rules for your project:

```shell
$ gcloud compute forwarding-rules list
```

Look for the rule called `wpfrontend`, which is what we named the wordpress service, and note its IP address.

## Visit your new WordPress blog

To access your new installation, you first may need to open up port 80 (the port specified in the wordpress service config) in the firewall for your cluster. You can do this, e.g. via:

```shell
$ gcloud compute firewall-rules create sample-http --allow tcp:80
```

This will define a firewall rule called `sample-http` that opens port 80 in the default network for your project.

Now, we can visit the running WordPress app.
Use the external IP that you obtained above, and visit it on port 80:

```
http://<external_ip>
```

You should see the familiar WordPress init page.

## Take down and restart your blog

Set up your WordPress blog and play around with it a bit.  Then, take down its pods and bring them back up again. Because you used persistent disks, your blog state will be preserved.

If you are just experimenting, you can take down and bring up only the pods:

```shell
$ <kubernetes>/cluster/kubectl.sh delete -f wordpress.yaml
$ <kubernetes>/cluster/kubectl.sh delete -f mysql.yaml
```

When you restart the pods again (using the `create` operation as described above), their services will pick up the new pods based on their labels.

If you want to shut down the entire app installation, you can delete the services as well.

If you are ready to turn down your Kubernetes cluster altogether, run:

```shell
$ <kubernetes>/cluster/kube-down.sh
```








