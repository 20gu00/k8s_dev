# Images
Each container in a pod has its own image.  Currently, the only type of image supported is a [Docker Image](https://docs.docker.com/userguide/dockerimages/).

You create your Docker image and push it to a registry before referring to it in a kubernetes pod.

The `image` property of a container supports the same syntax as the `docker` command does, including private registries and tags.

## Using a Private Registry
Keys for private registries are stored in a `.dockercfg` file.  Create a config file by running `docker login <registry>.<domain>` and then copying the resulting `.dockercfg` file to the kubelet working dir.
The kubelet working dir varies by cloud provider.  It is `/` on GCE and `/home/core` on CoreOS.  You can determine the working dir by running this command:
`sudo ls -ld /proc/$(pidof kubelet)/cwd` on a kNode.

All users of the cluster will have access to any private registry in the `.dockercfg`.

## Preloading Images

Be default, the kubelet will try to pull each image from the specified registry.
However, if the `imagePullPolicy` property of the container is set to `IfNotPresent` or `Never`,
then a local image is used (preferentially or exclusively, respectively).

This can be used to preload certain images for speed or as an alternative to authenticating to a private registry.

Pull Policy is per-container, but any user of the cluster will have access to all local images.
