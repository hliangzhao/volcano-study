/*
Copyright 2021 hliangzhao.

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

package cache

import (
	"github.com/hliangzhao/volcano/pkg/scheduler/apis"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/pkg/scheduler/framework/plugins/volumebinding"
)

// Cache collects pods/nodes/queues information and provides their snapshots.
type Cache interface {
	// Run start informer
	Run(stopCh <-chan struct{})

	// Snapshot deepcopy overall cache information into snapshot
	Snapshot() *apis.ClusterInfo

	// WaitForCacheSync waits for all cache synced
	WaitForCacheSync(stopCh <-chan struct{})

	// AddBindTask binds Task to the target host
	AddBindTask(task *apis.TaskInfo) error

	// BindPodGroup binds job to cluster
	BindPodGroup(job *apis.JobInfo, cluster string) error

	// Evict evicts the task to release resources
	Evict(task *apis.TaskInfo, reason string) error

	// RecordJobStatusEvent records related events according to job status.
	// Deprecated: remove it after removed PDB support.
	RecordJobStatusEvent(job *apis.JobInfo)

	// UpdateJobStatus puts job in backlog for a while
	UpdateJobStatus(job *apis.JobInfo, updatePG bool) (*apis.JobInfo, error)

	// GetPodVolumes returns the volumes of task
	GetPodVolumes(task *apis.TaskInfo, node *corev1.Node) (*volumebinding.PodVolumes, error)

	// AllocateVolumes allocates volume on the host to the task
	AllocateVolumes(task *apis.TaskInfo, hostname string, podVolumes *volumebinding.PodVolumes) error

	// BindVolumes binds volumes to the task
	BindVolumes(task *apis.TaskInfo, volumes *volumebinding.PodVolumes) error

	// Client returns the kubernetes clientSet, which can be used by plugins
	Client() kubernetes.Interface

	// UpdateSchedulerNumaInfo updates numa info
	UpdateSchedulerNumaInfo(sets map[string]apis.ResNumaSets) error

	// SharedInformerFactory return scheduler SharedInformerFactory
	SharedInformerFactory() informers.SharedInformerFactory
}

// Binder binds task and hostname
type Binder interface {
	Bind(kubeClient *kubernetes.Clientset, tasks []*apis.TaskInfo) (error, []*apis.TaskInfo)
}

// Evictor evicts pods
type Evictor interface {
	Evict(pod *corev1.Pod, reason string) error
}

// StatusUpdater updates pod with given PodCondition
type StatusUpdater interface {
	UpdatePodCondition(pod *corev1.Pod, podCondition *corev1.PodCondition) (*corev1.Pod, error)
	UpdatePodGroup(pg *apis.PodGroup) (*apis.PodGroup, error)
}

// VolumeBinder allocates and binds volumes
type VolumeBinder interface {
	GetPodVolumes(task *apis.TaskInfo, node *corev1.Node) (*volumebinding.PodVolumes, error)
	AllocateVolumes(task *apis.TaskInfo, hostname string, podVolumes *volumebinding.PodVolumes) error
	BindVolumes(task *apis.TaskInfo, podVolumes *volumebinding.PodVolumes) error
}

// BatchBinder updates job information
type BatchBinder interface {
	Bind(job *apis.JobInfo, cluster string) (*apis.JobInfo, error)
}
