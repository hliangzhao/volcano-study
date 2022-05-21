//go:build !ignore_autogenerated
// +build !ignore_autogenerated

/*
Copyright 2021-2022 The Volcano Authors.

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
// Code generated by deepcopy-gen. DO NOT EDIT.

package v1alpha1

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CPUInfo) DeepCopyInto(out *CPUInfo) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CPUInfo.
func (in *CPUInfo) DeepCopy() *CPUInfo {
	if in == nil {
		return nil
	}
	out := new(CPUInfo)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Numatopology) DeepCopyInto(out *Numatopology) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Numatopology.
func (in *Numatopology) DeepCopy() *Numatopology {
	if in == nil {
		return nil
	}
	out := new(Numatopology)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Numatopology) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NumatopologyList) DeepCopyInto(out *NumatopologyList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Numatopology, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NumatopologyList.
func (in *NumatopologyList) DeepCopy() *NumatopologyList {
	if in == nil {
		return nil
	}
	out := new(NumatopologyList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *NumatopologyList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NumatopologySpec) DeepCopyInto(out *NumatopologySpec) {
	*out = *in
	if in.Policies != nil {
		in, out := &in.Policies, &out.Policies
		*out = make(map[PolicyName]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.ResReserved != nil {
		in, out := &in.ResReserved, &out.ResReserved
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.NumaResMap != nil {
		in, out := &in.NumaResMap, &out.NumaResMap
		*out = make(map[string]ResourceInfo, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.CPUDetail != nil {
		in, out := &in.CPUDetail, &out.CPUDetail
		*out = make(map[string]CPUInfo, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NumatopologySpec.
func (in *NumatopologySpec) DeepCopy() *NumatopologySpec {
	if in == nil {
		return nil
	}
	out := new(NumatopologySpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ResourceInfo) DeepCopyInto(out *ResourceInfo) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ResourceInfo.
func (in *ResourceInfo) DeepCopy() *ResourceInfo {
	if in == nil {
		return nil
	}
	out := new(ResourceInfo)
	in.DeepCopyInto(out)
	return out
}
