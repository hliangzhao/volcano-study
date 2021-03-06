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

package plugininterface

// fully checked and understood

import (
	batchv1alpha1 "github.com/hliangzhao/volcano/pkg/apis/batch/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

// PluginClient wraps a kube client for plugin execution
type PluginClient struct {
	KubeClient kubernetes.Interface
}

type PluginInterface interface {
	// Name returns the name of the plugin
	Name() string

	// OnPodCreate is called when creating job pod
	OnPodCreate(pod *corev1.Pod, job *batchv1alpha1.Job) error

	// OnJobAdd is called when do job initiation
	// Note: it can be called multi times, must be idempotent
	OnJobAdd(job *batchv1alpha1.Job) error

	// OnJobDelete is called when killing job
	// Note: it can be called multi times, must be idempotent
	OnJobDelete(job *batchv1alpha1.Job) error

	// OnJobUpdate is called when job updated
	// Note: it can be called multi times, must be idempotent
	OnJobUpdate(job *batchv1alpha1.Job) error
}
