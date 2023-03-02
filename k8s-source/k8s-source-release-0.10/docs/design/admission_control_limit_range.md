# Admission control plugin: LimitRanger

## Background

This document proposes a system for enforcing min/max limits per resource as part of admission control.

## Model Changes

A new resource, **LimitRange**, is introduced to enumerate min/max limits for a resource type scoped to a
Kubernetes namespace.

```
const (
  // Limit that applies to all pods in a namespace
  LimitTypePod string = "Pod"
  // Limit that applies to all containers in a namespace
  LimitTypeContainer string = "Container"
)

// LimitRangeItem defines a min/max usage limit for any resource that matches on kind
type LimitRangeItem struct {
  // Type of resource that this limit applies to
  Type string `json:"type,omitempty"`
  // Max usage constraints on this kind by resource name
  Max ResourceList `json:"max,omitempty"`
  // Min usage constraints on this kind by resource name
  Min ResourceList `json:"min,omitempty"`
}

// LimitRangeSpec defines a min/max usage limit for resources that match on kind
type LimitRangeSpec struct {
  // Limits is the list of LimitRangeItem objects that are enforced
  Limits []LimitRangeItem `json:"limits"`
}

// LimitRange sets resource usage limits for each kind of resource in a Namespace
type LimitRange struct {
  TypeMeta   `json:",inline"`
  ObjectMeta `json:"metadata,omitempty"`

  // Spec defines the limits enforced
  Spec LimitRangeSpec `json:"spec,omitempty"`
}

// LimitRangeList is a list of LimitRange items.
type LimitRangeList struct {
  TypeMeta `json:",inline"`
  ListMeta `json:"metadata,omitempty"`

  // Items is a list of LimitRange objects
  Items []LimitRange `json:"items"`
}
```

## AdmissionControl plugin: LimitRanger

The **LimitRanger** plug-in introspects all incoming admission requests. 

It makes decisions by evaluating the incoming object against all defined **LimitRange** objects in the request context namespace.

The following min/max limits are imposed:

**Type: Container**

| ResourceName | Description |
| ------------ | ----------- |
| cpu | Min/Max amount of cpu per container |
| memory | Min/Max amount of memory per container |

**Type: Pod**

| ResourceName | Description |
| ------------ | ----------- |
| cpu | Min/Max amount of cpu per pod |
| memory | Min/Max amount of memory per pod |

If the incoming object would cause a violation of the enumerated constraints, the request is denied with a set of
messages explaining what constraints were the source of the denial.

If a constraint is not enumerated by a **LimitRange** it is not tracked.

## kube-apiserver

The server is updated to be aware of **LimitRange** objects.

The constraints are only enforced if the kube-apiserver is started as follows:

```
$ kube-apiserver -admission_control=LimitRanger
```

## kubectl

kubectl is modified to support the **LimitRange** resource.

```kubectl describe``` provides a human-readable output of limits.

For example,

```
$ kubectl namespace myspace
$ kubectl create -f examples/limitrange/limit-range.json
$ kubectl get limits
NAME
limits
$ kubectl describe limits limits
Name:   limits
Type    Resource  Min Max
----    --------  --- ---
Pod   memory    1Mi 1Gi
Pod   cpu   250m  2
Container cpu   250m  2
Container memory    1Mi 1Gi
```

## Future Enhancements: Define limits for a particular pod or container.

In the current proposal, the **LimitRangeItem** matches purely on **LimitRangeItem.Type**

It is expected we will want to define limits for particular pods or containers by name/uid and label/field selector.

To make a **LimitRangeItem** more restrictive, we will intend to add these additional restrictions at a future point in time.
