
# Kubernetes Container Environment

## Overview
This document describes the environment for Kubelet managed containers on a Kubernetes node (kNode).  In contrast to the Kubernetes cluster API, which provides an API for creating and managing containers, the Kubernetes container environment provides the container access to information about what else is going on in the cluster. 

This cluster information makes it possible to build applications that are *cluster aware*.  
Additionally, the Kubernetes container environment defines a series of hooks that are surfaced to optional hook handlers defined as part of individual containers.  Container hooks are somewhat analagous to operating system signals in a traditional process model.   However these hooks are designed to make it easier to build reliable, scalable cloud applications in the Kubernetes cluster.  Containers that participate in this cluster lifecycle become *cluster native*. 

Another important part of the container environment is the file system that is available to the container.  In Kubernetes, the filesystem is a combination of an [image](./images.md) and one or more [volumes](./volumes.md).


The following sections describe both the cluster information provided to containers, as well as the hooks and life-cycle that allows containers to interact with the management system.

## Cluster Information
There are two types of information that are available within the container environment.  There is information about the container itself, and there is information about other objects in the system.

### Container Information
Currently, the only information about the container that is available to the container is the Pod name for the pod in which the container is running.  This ID is set as the hostname of the container, and is accessible through all calls to access the hostname within the container (e.g. the hostname command, or the [gethostname][1] function call in libc).  Additionally, user-defined environment variables from the pod definition, are also available to the container, as are any environment variables specified statically in the Docker image.

In the future, we anticipate expanding this information with richer information about the container.  Examples include available memory, number of restarts, and in general any state that you could get from the call to GET /pods on the API server.

### Cluster Information
Currently the list of all services that are running at the time when the container was created via the Kubernetes Cluster API are available to the container as environment variables.  The set of environment variables matches the syntax of Docker links.

For a service named **foo** that maps to a container port named **bar**, the following variables are defined:

```sh
FOO_SERVICE_HOST=<the host the service is running on>
FOO_SERVICE_PORT=<the port the service is running on>
```

Going forward, we expect that Services will have a dedicated IP address.  In that context, we will also surface services to the container via DNS.  Of course DNS is still not an enumerable protocol, so we will continue to provide environment variables so that containers can do discovery.

## Container Hooks
*NB*: Container hooks are under active development, we anticipate adding additional hooks as the Kubernetes container management system evolves.*

Container hooks provide information to the container about events in its management lifecycle.  For example, immediately after a container is started, it receives a *PostStart* hook.  These hooks are broadcast *into* the container with information about the life-cycle of the container.  They are different from the events provided by Docker and other systems which are *output* from the container.  Output events provide a log of what has already happened.  Input hooks provide real-time notification about things that are happening, but no historical log.  

### Hook Details
There are currently two container hooks that are surfaced to containers, and two proposed hooks:

*PreStart - ****Proposed***

This hook is sent immediately before a container is created.  It notifies that the container will be created immediately after the call completes.  No parameters are passed. *Note - *Some event handlers (namely ‘exec’ are incompatible with this event)

*PostStart*

This hook is sent immediately after a container is created.  It notifies the container that it has been created.  No parameters are passed to the handler.

*PostRestart - ****Proposed***

This hook is called before the PostStart handler, when a container has been restarted, rather than started for the first time.  No parameters are passed to the handler.

*PreStop*

This hook is called immediately before a container is terminated.  This event handler is blocking, and must complete before the call to delete the container is sent to the Docker daemon.  The SIGTERM notification sent by Docker is also still sent.

A single parameter named reason is passed to the handler which contains the reason for termination.  Currently the valid values for reason are:
* ●	```Delete``` - indicating an API call to delete the pod containing this container.
* ●	```Health``` - indicating that a health check of the container failed.
* ●	```Dependency``` - indicating that a dependency for the container or the pod is missing, and thus, the container needs to be restarted.  Examples include, the pod infra container crashing, or persistent disk failing for a container that mounts PD.

Eventually, user specified reasons may be [added to the API](https://github.com/GoogleCloudPlatform/kubernetes/issues/137).


### Hook Handler Execution
When a management hook occurs, the management system calls into any registered hook handlers in the container for that hook.  These hook handler calls are synchronous in the context of the pod containing the container. Note:this means that hook handler execution blocks any further management of the pod.  If your hook handler blocks, no other management (including health checks) will occur until the hook handler completes.  Blocking hook handlers do *not* affect management of other Pods.  Typically we expect that users will make their hook handlers as lightweight as possible, but there are cases where long running commands make sense (e.g. saving state prior to container stop)

For hooks which have parameters, these parameters are passed to the event handler as a set of key/value pairs.  The details of this parameter passing is handler implementation dependent (see below)

### Hook Handler Implementations
Hook handlers are the way that hooks are surfaced to containers.  Containers can select the type of hook handler they would like to implement.  Kubernetes currently supports two different hook handler types:

   * Exec - Executes a specific command (e.g. pre-stop.sh) inside the cgroup and namespaces of the container.  Resources consumed by the command are counted against the container.  Commands which print "ok" to standard out (stdout) are treated as healthy, any other output is treated as container failures (and will cause kubelet to forcibly restart the container).  Parameters are passed to the command as traditional linux command line flags (e.g. pre-stop.sh --reason=HEALTH)

   * HTTP - Executes an HTTP request against a specific endpoint on the container.  HTTP error codes (5xx) and non-response/failure to connect are treated as container failures. Parameters are passed to the http endpoint as query args (e.g. http://some.server.com/some/path?reason=HEALTH)

[1]: http://man7.org/linux/man-pages/man2/gethostname.2.html
