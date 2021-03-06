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

package cache

// fully checked and understood

import (
	"fmt"
	batchv1alpha1 "github.com/hliangzhao/volcano/pkg/apis/batch/v1alpha1"
	controllerapis "github.com/hliangzhao/volcano/pkg/controllers/apis"
	"golang.org/x/time/rate"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	"sync"
	"time"
)

/* JobInfo is reconstructed in cache to JobCache, the local store of volcano jobs.
The scheduling actions are made for jobs in JobCache. */

type jobCache struct {
	sync.Mutex
	jobInfos    map[string]*controllerapis.JobInfo // {jobName: jobInfo}
	deletedJobs workqueue.RateLimitingInterface
}

func keyFn(namespace, name string) string {
	return fmt.Sprintf("%s/%s", namespace, name)
}

func JobKeyByName(namespace string, name string) string {
	return keyFn(namespace, name)
}

func JobKeyByRequest(req *controllerapis.Request) string {
	return keyFn(req.Namespace, req.JobName)
}

func JobKey(job *batchv1alpha1.Job) string {
	return keyFn(job.Namespace, job.Name)
}

func jobTerminated(job *controllerapis.JobInfo) bool {
	return job.Job == nil || len(job.Pods) == 0
}

func jobKeyOfPod(pod *corev1.Pod) (string, error) {
	jobName, found := pod.Annotations[batchv1alpha1.JobNameKey]
	if !found {
		return "", fmt.Errorf("failed to find job name of pod <%s/%s>", pod.Namespace, pod.Name)
	}
	return keyFn(pod.Namespace, jobName), nil
}

func NewCache() Cache {
	return &jobCache{
		jobInfos: map[string]*controllerapis.JobInfo{},
		deletedJobs: workqueue.NewRateLimitingQueue(workqueue.NewMaxOfRateLimiter(
			workqueue.NewItemExponentialFailureRateLimiter(5*time.Microsecond, 180*time.Second),
			&workqueue.BucketRateLimiter{Limiter: rate.NewLimiter(rate.Limit(10), 100)},
		)),
	}
}

// Get returns the copy of the jobInfo from the cache by job key.
func (jc *jobCache) Get(key string) (*controllerapis.JobInfo, error) {
	jc.Lock()
	defer jc.Unlock()

	ji, found := jc.jobInfos[key]
	if !found {
		return nil, fmt.Errorf("failed to find job <%s>", key)
	}
	if ji.Job == nil {
		return nil, fmt.Errorf("job <%s> is not ready", key)
	}
	return ji.Clone(), nil
}

// GetStatus returns the status of the jobInfo stored in the cache by job key.
func (jc *jobCache) GetStatus(key string) (*batchv1alpha1.JobStatus, error) {
	jc.Lock()
	defer jc.Unlock()

	ji, found := jc.jobInfos[key]
	if !found {
		return nil, fmt.Errorf("failed to find job <%s>", key)
	}
	if ji.Job == nil {
		return nil, fmt.Errorf("job <%s> is not ready", key)
	}

	status := ji.Job.Status
	return &status, nil
}

// Add adds a new job to the cache.
func (jc *jobCache) Add(job *batchv1alpha1.Job) error {
	jc.Lock()
	defer jc.Unlock()

	key := JobKey(job)
	if ji, found := jc.jobInfos[key]; found {
		if ji.Job == nil {
			ji.SetJob(job)
			return nil
		}
		return fmt.Errorf("duplicated jobInfo <%v>", key)
	}

	jc.jobInfos[key] = &controllerapis.JobInfo{
		Name:      job.Name,
		Namespace: job.Namespace,
		Job:       job,
		Pods:      make(map[string]map[string]*corev1.Pod),
	}
	return nil
}

// Update updates the given job in the cache.
func (jc *jobCache) Update(job *batchv1alpha1.Job) error {
	jc.Lock()
	defer jc.Unlock()

	key := JobKey(job)
	ji, found := jc.jobInfos[key]
	if !found {
		return fmt.Errorf("failed to find job <%v>", key)
	}

	ji.Job = job
	return nil
}

// deleteJob deletes a job from cache. We only need to add the job to the `deletedJobs` work-queue.
func (jc *jobCache) deleteJob(ji *controllerapis.JobInfo) {
	klog.V(3).Infof("Try to delete Job <%v/%v>", ji.Namespace, ji.Name)
	jc.deletedJobs.AddRateLimited(ji)
}

// Delete adds the given job (it must be stored in cache) to the deletedJobs queue.
func (jc *jobCache) Delete(job *batchv1alpha1.Job) error {
	jc.Lock()
	defer jc.Unlock()

	key := JobKey(job)
	ji, found := jc.jobInfos[key]
	if !found {
		return fmt.Errorf("failed to find job <%v>", key)
	}

	ji.Job = nil
	jc.deleteJob(ji)
	return nil
}

// AddPod adds the given pod to the right jobInfo in the cache.
func (jc *jobCache) AddPod(pod *corev1.Pod) error {
	jc.Lock()
	defer jc.Unlock()

	key, err := jobKeyOfPod(pod)
	if err != nil {
		return err
	}
	ji, found := jc.jobInfos[key]
	if !found {
		jc.jobInfos[key] = &controllerapis.JobInfo{
			Pods: make(map[string]map[string]*corev1.Pod),
		}
	}

	return ji.AddPod(pod)
}

// UpdatePod updates the given pod to the right jobInfo in the cache.
func (jc *jobCache) UpdatePod(pod *corev1.Pod) error {
	jc.Lock()
	defer jc.Unlock()

	key, err := jobKeyOfPod(pod)
	if err != nil {
		return err
	}
	ji, found := jc.jobInfos[key]
	if !found {
		jc.jobInfos[key] = &controllerapis.JobInfo{
			Pods: make(map[string]map[string]*corev1.Pod),
		}
	}

	return ji.UpdatePod(pod)
}

// DeletePod deletes pod from the cache.
func (jc *jobCache) DeletePod(pod *corev1.Pod) error {
	jc.Lock()
	defer jc.Unlock()

	key, err := jobKeyOfPod(pod)
	if err != nil {
		return err
	}
	ji, found := jc.jobInfos[key]
	if !found {
		jc.jobInfos[key] = &controllerapis.JobInfo{
			Pods: make(map[string]map[string]*corev1.Pod),
		}
	}

	if err = ji.DeletePod(pod); err != nil {
		return err
	}
	if jc.jobInfos[key].Job == nil {
		jc.deleteJob(ji)
	}
	return nil
}

// TaskCompleted judges whether a specific task (located by jobKey and taskName) in cache is competed.
func (jc *jobCache) TaskCompleted(jobKey, taskName string) bool {
	jc.Lock()
	defer jc.Unlock()

	var taskReplicas, completed int32

	ji, found := jc.jobInfos[jobKey]
	if !found {
		return false
	}
	taskPods, found := ji.Pods[taskName]
	if !found || ji.Job == nil {
		return false
	}

	for _, task := range ji.Job.Spec.Tasks {
		if task.Name == taskName {
			taskReplicas = task.Replicas
			break
		}
	}
	if taskReplicas <= 0 {
		return false
	}

	for _, pod := range taskPods {
		if pod.Status.Phase == corev1.PodSucceeded {
			completed++
		}
	}
	return completed >= taskReplicas
}

// TaskFailed judges whether a specific task (located by jobKey and taskName) in cache is failed.
func (jc *jobCache) TaskFailed(jobKey, taskName string) bool {
	jc.Lock()
	defer jc.Unlock()

	var taskReplicas, retried, maxRetry int32

	ji, found := jc.jobInfos[jobKey]
	if !found {
		return false
	}
	taskPods, found := ji.Pods[taskName]
	if !found || ji.Job == nil {
		return false
	}

	for _, task := range ji.Job.Spec.Tasks {
		if task.Name == taskName {
			maxRetry = task.MaxRetry
			taskReplicas = task.Replicas
			break
		}
	}
	// maxRetry == -1 means no limit
	if taskReplicas == 0 || maxRetry == -1 {
		return false
	}

	// Compatible with existing job
	if maxRetry == 0 {
		maxRetry = 3
	}

	for _, pod := range taskPods {
		if pod.Status.Phase == corev1.PodRunning || pod.Status.Phase == corev1.PodPending {
			for j := range pod.Status.InitContainerStatuses {
				stat := pod.Status.InitContainerStatuses[j]
				retried += stat.RestartCount
			}
			for j := range pod.Status.ContainerStatuses {
				stat := pod.Status.ContainerStatuses[j]
				retried += stat.RestartCount
			}
		}
	}
	return retried > maxRetry
}

// processCleanupJob safely delete the terminated jobs (or add the to-be-delete job to the `deletedJobs` work-queue.
func (jc *jobCache) processCleanupJob() bool {
	obj, shutdown := jc.deletedJobs.Get()
	if shutdown {
		return false
	}
	defer jc.deletedJobs.Done(obj)

	ji, ok := obj.(*controllerapis.JobInfo)
	if !ok {
		klog.Errorf("failed to convert %v to *controllerapis.JobInfo", obj)
		return true
	}

	jc.Lock()
	defer jc.Unlock()

	if jobTerminated(ji) {
		// the job could finally be deleted
		jc.deletedJobs.Forget(obj)
		key := keyFn(ji.Namespace, ji.Name)
		delete(jc.jobInfos, key)
		klog.V(3).Infof("Job <%s> was deleted.", key)
	} else {
		jc.deleteJob(ji)
	}

	return true
}

func (jc *jobCache) worker() {
	// the for-loop is the easiest way to run processCleanupJob() continuously
	for jc.processCleanupJob() {
	}
}

// Run clean up deleted jobs in cache until the stop signal is captured.
func (jc *jobCache) Run(stopCh <-chan struct{}) {
	wait.Until(jc.worker, 0, stopCh)
}
