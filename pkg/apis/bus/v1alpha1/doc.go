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

// Package v1alpha1 contains API Schema definitions for the bus v1alpha1 API group
// Package bus is a common module that can be used by the left apis. In this package, we define the api Command.
// It describes a command that will act on these api objects.
// +kubebuilder:object:generate=true
// +groupName=bus.volcano.sh
// +k8s:deepcopy-gen=package
package v1alpha1
