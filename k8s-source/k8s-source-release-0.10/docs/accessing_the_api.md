# Reaching the API

## Ports and IPs Served On
The Kubernetes API is served by the Kubernetes APIServer process.  Typically,
there is one of these running on a single kubernetes-master node.

By default the Kubernetes APIserver serves
HTTP on 3 ports:
  1. Localhost Port
    - serves HTTP
    - default is port 8080, change with `-port` flag.
    - defaults IP is localhost, change with `-address` flag.
    - no authentication or authorization checks in HTTP
    - protected by need to have host access
  2. ReadOnly Port
    - default is port 7080, change with `-read_only_port`
    - default IP is first non-localhost network interface, change with `-public_address_override`
    - serves HTTP
    - no authentication checks in HTTP
    - only GET requests are allowed.
    - requests are rate limited
  3. Secure Port
    - default is port 6443, change with `-secure_port`
    - default IP is first non-localhost network interface, change with `-public_address_override`
    - serves HTTPS.  Set cert with `-tls_cert_file` and key with `-tls_private_key_file`.
    - uses token-file based [authentication](./authentication.md).
    - uses policy-based [authorization](./authorization.md).

## Proxies and Firewall rules

Additionally, in typical configurations (i.e. GCE), there is a proxy (nginx) running
on the same machine as the apiserver process.  The proxy serves HTTPS protected
by Basic Auth on port 443, and proxies to the apiserver on localhost:8080.
Typically, firewall rules will allow HTTPS access to port 443.

The above are defaults and reflect how Kubernetes is deployed to GCE using
kube-up.sh.  Other cloud providers may vary.

## Use Cases vs IP:Ports

There are three differently configured serving ports because there are a
variety of uses cases:
   1. Clients outside of a Kubernetes cluster, such as human running `kubectl`
      on desktop machine.  Currently, accesses the Localhost Port via a proxy (nginx)
      running on the `kubernetes-master` machine.  Proxy uses Basic Auth.
   2. Processes running in Containers on Kubernetes that need to do read from
      the apiserver.  Currently, these can use Readonly Port.
   3. Scheduler and Controller-manager processes, which need to do read-write
      API operations.  Currently, these have to run on the 
      operations on the apiserver.  Currently, these have to run on the same
      host as the apiserver and use the Localhost Port.
   4. Kubelets, which need to do read-write API operations and are necessarily 
      on different machines than the apiserver.  Currently, kubelets do not
      use the API.

## Expected Changes.
The following changes to what is decribed above are planned:
   - Kubelets will soon begin using the Secure Port to get their pods and
     report events.  Credentials will be distributed to kubelets at cluster
     setup time initially.  Policy will limit the actions kubelets can do.
   - Scheduler and Controller-manager will use the Secure Port too.  They
     will then be able to run on different machines than the apiserver.
   - A general mechanism will be provided for [giving credentials to
     pods](
     https://github.com/GoogleCloudPlatform/kubernetes/issues/1907).
   - The Readonly Port will no longer be needed and will be removed.
   - Clients, like kubectl, will all support token-based auth, and the
     Localhost will no longer be needed, and will not be the default.
     However, the localhost port may continue to be an option for
     installations that want to do their own auth proxy.
