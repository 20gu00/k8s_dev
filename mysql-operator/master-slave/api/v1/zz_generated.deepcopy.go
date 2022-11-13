//go:build !ignore_autogenerated
// +build !ignore_autogenerated

/*
Copyright 2022 cjq.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by controller-gen. DO NOT EDIT.

package v1

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MasterSlave) DeepCopyInto(out *MasterSlave) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MasterSlave.
func (in *MasterSlave) DeepCopy() *MasterSlave {
	if in == nil {
		return nil
	}
	out := new(MasterSlave)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *MasterSlave) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MasterSlaveList) DeepCopyInto(out *MasterSlaveList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]MasterSlave, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MasterSlaveList.
func (in *MasterSlaveList) DeepCopy() *MasterSlaveList {
	if in == nil {
		return nil
	}
	out := new(MasterSlaveList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *MasterSlaveList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MasterSlaveSpec) DeepCopyInto(out *MasterSlaveSpec) {
	*out = *in
	if in.Replicas != nil {
		in, out := &in.Replicas, &out.Replicas
		*out = new(int32)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MasterSlaveSpec.
func (in *MasterSlaveSpec) DeepCopy() *MasterSlaveSpec {
	if in == nil {
		return nil
	}
	out := new(MasterSlaveSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MasterSlaveStatus) DeepCopyInto(out *MasterSlaveStatus) {
	*out = *in
	in.StatefulSetStatus.DeepCopyInto(&out.StatefulSetStatus)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MasterSlaveStatus.
func (in *MasterSlaveStatus) DeepCopy() *MasterSlaveStatus {
	if in == nil {
		return nil
	}
	out := new(MasterSlaveStatus)
	in.DeepCopyInto(out)
	return out
}
