# Labels

_Labels_ are key/value pairs that are attached to objects, such as pods.
Labels can be used to organize and to select subsets of objects.  They are
created by users at the same time as an object.  Each object can have a set of
key/value labels set on it, with at most one label with a particular key. 
```
"labels": {
  "key1" : "value1",
  "key2" : "value2"
}
```

Unlike [names and UIDs](identifiers.md), labels do not provide uniqueness. In general, we expect many objects to carry the same label(s). 

Via a _label selector_, the client/user can identify a set of objects. The label selector is the core grouping primitive in Kubernetes. 

We also [plan](https://github.com/GoogleCloudPlatform/kubernetes/issues/560) to make labels available inside pods and [lifecycle hooks](container-environment.md).

Labels let you categorize objects in a complex service deployment or batch processing pipelines along multiple
dimensions, such as:
   - `release=stable`, `release=canary`, ...
   - `environment=dev`, `environment=qa`, `environment=production`
   - `tier=frontend`, `tier=backend`, ...
   - `partition=customerA`, `partition=customerB`, ...
   - `track=daily`, `track=weekly`

These are just examples; you are free to develop your own conventions.

Label selectors permit very simple filtering by label keys and values.   Currently, label selectors only support these forms:
```
key1
key1 = value11
key1 != value11
key1 in (value11, value12, ...)
key1 not in (value11, value12, ...)
```

LIST and WATCH operations may specify label selectors to filter the sets of objects returned using a query parameter: `?labels=key1%3Dvalue1,key2%3Dvalue2,...`. 

The `service` and `replicationController` kinds of objects use selectors to match sets of pods that they operate on.

See the [Labels Design Document](./design/labels.md) for more about how we expect labels and selectors to be used, and planned features.
