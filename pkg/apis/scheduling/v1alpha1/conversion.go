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

package v1alpha1

import (
	"github.com/hliangzhao/volcano/pkg/apis/scheduling"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/conversion"
	"unsafe"
)

func Convert_scheduling_QueueSpec_To_v1alpha1_QueueSpec(in *scheduling.QueueSpec, out *QueueSpec, s conversion.Scope) error {
	out.Weight = in.Weight
	out.Capability = *(*corev1.ResourceList)(unsafe.Pointer(&in.Capability))
	out.Reclaimable = (*bool)(unsafe.Pointer(in.Reclaimable))

	return nil
}
