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

import (
	"context"
	"fmt"
	nodeinfov1alpha1 "github.com/hliangzhao/volcano/pkg/apis/nodeinfo/v1alpha1"
	"github.com/hliangzhao/volcano/pkg/apis/scheduling"
	"github.com/hliangzhao/volcano/pkg/apis/scheduling/scheme"
	schedulingv1alpha1 "github.com/hliangzhao/volcano/pkg/apis/scheduling/v1alpha1"
	"github.com/hliangzhao/volcano/pkg/apis/utils"
	"github.com/hliangzhao/volcano/pkg/scheduler/apis"
	`github.com/hliangzhao/volcano/pkg/scheduler/apis/helpers`
	corev1 "k8s.io/api/core/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpumanager/topology"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpuset"
	"strconv"
)

/* In this file, we implement the functions that Add, Update, and Delete different fields of SchedulerCache. */

// isTerminated checks whether status is succeeded or failed.
func isTerminated(status apis.TaskStatus) bool {
	return status == apis.Succeeded || status == apis.Failed
}

// getJobID returns <podgroup-ns/podgroup-name>.
func getJobID(pg *apis.PodGroup) apis.JobID {
	return apis.JobID(fmt.Sprintf("%s/%s", pg.Namespace, pg.Name))
}

// AddJob adds obj to sc.Jobs.
func (sc *SchedulerCache) AddJob(obj interface{}) {
	job, ok := obj.(*apis.JobInfo)
	if !ok {
		klog.Errorf("Cannot convert to *apis.JobInfo: %v", obj)
		return
	}

	sc.Mutex.Lock()
	defer sc.Mutex.Unlock()
	sc.Jobs[job.UID] = job
}

// getOrCreateJob returns the apis.JobInfo that wraps ti if the job exists. Otherwise, create one and return it.
func (sc *SchedulerCache) getOrCreateJob(ti *apis.TaskInfo) *apis.JobInfo {
	if len(ti.Job) == 0 {
		if ti.Pod.Spec.SchedulerName != sc.schedulerName {
			klog.V(4).Infof("Pod %s/%s will not scheduled by %s, skip creating PodGroup and Job for it",
				ti.Pod.Namespace, ti.Pod.Name, sc.schedulerName)
		}
		return nil
	}

	if _, found := sc.Jobs[ti.Job]; !found {
		sc.Jobs[ti.Job] = apis.NewJobInfo(ti.Job)
	}

	return sc.Jobs[ti.Job]
}

/* handling Pod and Task */

// addTask adds task to the right job and then adds the job to sc.Jobs.
func (sc *SchedulerCache) addTask(task *apis.TaskInfo) error {
	if len(task.NodeName) != 0 {
		// create node for task if not found
		if _, found := sc.Nodes[task.NodeName]; !found {
			sc.Nodes[task.NodeName] = apis.NewNodeInfo(nil)
			sc.Nodes[task.NodeName].Name = task.NodeName
		}

		// add task to this node
		node := sc.Nodes[task.NodeName]
		if !isTerminated(task.Status) {
			if err := node.AddTask(task); err != nil {
				if _, outOfSync := err.(*apis.AllocateFailError); outOfSync {
					node.State = apis.NodeState{
						Phase:  apis.NotReady,
						Reason: "OutOfSync",
					}
				}
				return err
			}
		} else {
			klog.V(4).Infof("Pod <%v/%v> is in status %s.", task.Namespace, task.Name, task.Status.String())
		}
	}

	// get (or create) the job of task
	job := sc.getOrCreateJob(task)
	if job != nil {
		job.AddTaskInfo(task)
	}

	return nil
}

// addPod adds pod to sc.
func (sc *SchedulerCache) addPod(pod *corev1.Pod) error {
	// Assumes that lock is already acquired.
	task := apis.NewTaskInfo(pod)
	return sc.addTask(task)
}

// AddPod (wraps addPod) adds pod to sc.
func (sc *SchedulerCache) AddPod(obj interface{}) {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		klog.Errorf("Cannot convert to *corev1.Pod: %v", obj)
		return
	}

	sc.Mutex.Lock()
	defer sc.Mutex.Unlock()

	err := sc.addPod(pod)
	if err != nil {
		klog.Errorf("Failed to add pod <%s/%s> into cache: %v",
			pod.Namespace, pod.Name, err)
		return
	}

	klog.V(3).Infof("Added pod <%s/%v> into cache.", pod.Namespace, pod.Name)
}

// updatePod deletes oldPod and adds newPod to sc.
// Assumes that lock is already acquired.
func (sc *SchedulerCache) updatePod(oldPod, newPod *corev1.Pod) error {
	// ignore the update event if pod is allocated in cache but not present in NodeName
	if sc.allocatedPodInCache(newPod) && newPod.Spec.NodeName == "" {
		klog.V(4).Infof("Pod <%s/%v> already in cache with allocated status, ignore the update event",
			newPod.Namespace, newPod.Name)
		return nil
	}

	if err := sc.deletePod(oldPod); err != nil {
		return err
	}
	// when delete pod, the owner reference of pod will be set nil,just as orphan pod
	if len(utils.GetController(newPod)) == 0 {
		newPod.OwnerReferences = oldPod.OwnerReferences
	}
	return sc.addPod(newPod)
}

// UpdatePod (wraps updatePod) deletes oldObj and adds newObj to sc.
func (sc *SchedulerCache) UpdatePod(oldObj, newObj interface{}) {
	oldPod, ok := oldObj.(*corev1.Pod)
	if !ok {
		klog.Errorf("Cannot convert oldObj to *corev1.Pod: %v", oldObj)
		return
	}
	newPod, ok := newObj.(*corev1.Pod)
	if !ok {
		klog.Errorf("Cannot convert newObj to *corev1.Pod: %v", newObj)
		return
	}

	sc.Mutex.Lock()
	defer sc.Mutex.Unlock()

	err := sc.updatePod(oldPod, newPod)
	if err != nil {
		klog.Errorf("Failed to update pod %v in cache: %v", oldPod.Name, err)
		return
	}

	klog.V(4).Infof("Updated pod <%s/%v> in cache.", oldPod.Namespace, oldPod.Name)
}

// deletePod deletes the task (of the input pod) from sc.
// Further, if the job of the task is terminated, delete the job by the way.
func (sc *SchedulerCache) deletePod(pod *corev1.Pod) error {
	ti := apis.NewTaskInfo(pod)

	// Delete the Task (pod) in cache to handle Binding status.
	task := ti
	if job, found := sc.Jobs[ti.Job]; found {
		if t, found := job.Tasks[ti.UID]; found {
			task = t
		}
	}
	if err := sc.deleteTask(task); err != nil {
		klog.Warningf("Failed to delete task: %v", err)
	}

	// Further, if job was terminated, delete it.
	if job, found := sc.Jobs[ti.Job]; found && helpers.JobTerminated(job) {
		sc.deleteJob(job)
	}

	return nil
}

// DeletePod (wraps deletePod) deletes obj from sc.
func (sc *SchedulerCache) DeletePod(obj interface{}) {
	var pod *corev1.Pod
	switch t := obj.(type) {
	case *corev1.Pod:
		pod = t
	case cache.DeletedFinalStateUnknown:
		var ok bool
		pod, ok = t.Obj.(*corev1.Pod)
		if !ok {
			klog.Errorf("Cannot convert to *corev1.Pod: %v", t.Obj)
			return
		}
	default:
		klog.Errorf("Cannot convert to *corev1.Pod: %v", t)
		return
	}

	sc.Mutex.Lock()
	defer sc.Mutex.Unlock()

	err := sc.deletePod(pod)
	if err != nil {
		klog.Errorf("Failed to delete pod %v from cache: %v", pod.Name, err)
		return
	}

	klog.V(3).Infof("Deleted pod <%s/%v> from cache.", pod.Namespace, pod.Name)
}

// deleteTask deletes task from sc. Specifically, it deletes task from the corresponding job and scheduled node.
func (sc *SchedulerCache) deleteTask(task *apis.TaskInfo) error {
	var jobErr, nodeErr, numaErr error

	// remove task from job
	if len(task.Job) != 0 {
		if job, found := sc.Jobs[task.Job]; found {
			jobErr = job.DeleteTaskInfo(task)
		} else {
			jobErr = fmt.Errorf("failed to find Job <%v> for Task %v/%v",
				task.Job, task.Namespace, task.Name)
		}
	}

	// remove task from node
	if len(task.NodeName) != 0 {
		node := sc.Nodes[task.NodeName]
		if node != nil {
			nodeErr = node.RemoveTask(task)
		}
	}

	// TODO: numaErr not set

	if jobErr != nil || nodeErr != nil {
		return helpers.MergeErrors(jobErr, nodeErr, numaErr)
	}

	return nil
}

// updateTask deletes the old task from sc and add the new one to sc.
func (sc *SchedulerCache) updateTask(oldTask, newTask *apis.TaskInfo) error {
	if err := sc.deleteTask(oldTask); err != nil {
		klog.Warningf("Failed to delete task: %v", err)
	}
	return sc.addTask(newTask)
}

// syncTask gets the latest task from the cluster and update it to sc.
func (sc *SchedulerCache) syncTask(oldTask *apis.TaskInfo) error {
	latestPod, err := sc.kubeClient.CoreV1().Pods(oldTask.Namespace).Get(context.TODO(), oldTask.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			err := sc.deleteTask(oldTask)
			if err != nil {
				klog.Errorf("Failed to delete Pod <%v/%v> and remove from cache: %s", oldTask.Namespace, oldTask.Name, err.Error())
				return err
			}
			klog.V(3).Infof("Pod <%v/%v> was deleted, removed from cache.", oldTask.Namespace, oldTask.Name)
			return nil
		}
		return fmt.Errorf("failed to get Pod <%v/%v>: err %v", oldTask.Namespace, oldTask.Name, err)
	}

	newTask := apis.NewTaskInfo(latestPod)
	sc.Mutex.Lock()
	defer sc.Mutex.Unlock()
	return sc.updateTask(oldTask, newTask)
}

// allocatedPodInCache checks whether pod is scheduled to a node. True is returned only when pod in sc, and it is scheduled.
func (sc *SchedulerCache) allocatedPodInCache(pod *corev1.Pod) bool {
	task := apis.NewTaskInfo(pod)
	if job, found := sc.Jobs[task.Job]; found {
		if t, found := job.Tasks[task.UID]; found {
			return apis.AllocatedStatus(t.Status)
		}
	}
	return false
}

/* handling Node */

// addNode adds node to sc.Nodes.
func (sc *SchedulerCache) addNode(node *corev1.Node) error {
	if sc.Nodes[node.Name] != nil {
		sc.Nodes[node.Name].SetNode(node)
	} else {
		sc.Nodes[node.Name] = apis.NewNodeInfo(node)
	}
	return nil
}

// updateNode updates the node in sc.Nodes which is denoted by newNode.
func (sc *SchedulerCache) updateNode(oldNode, newNode *corev1.Node) error {
	// TODO: oldNode not used
	if sc.Nodes[newNode.Name] != nil {
		sc.Nodes[newNode.Name].SetNode(newNode)
		return nil
	}
	return fmt.Errorf("node <%s> does not exist", newNode.Name)
}

// deleteNode deletes node from sc.Nodes.
func (sc *SchedulerCache) deleteNode(node *corev1.Node) error {
	if _, ok := sc.Nodes[node.Name]; !ok {
		return fmt.Errorf("node <%s> does not exist", node.Name)
	}

	numaInfo := sc.Nodes[node.Name].NumaInfo
	if numaInfo != nil {
		klog.V(3).Infof("delete numatopo <%s/%s>", numaInfo.Namespace, numaInfo.Name)
		err := sc.vcClient.NodeinfoV1alpha1().Numatopologies().Delete(context.TODO(), numaInfo.Name, metav1.DeleteOptions{})
		if err != nil {
			klog.Errorf("delete numatopo <%s/%s> failed.", numaInfo.Namespace, numaInfo.Name)
		}
	}

	delete(sc.Nodes, node.Name)

	return nil
}

// AddNode (wraps addNode) adds obj to sc.Nodes.
func (sc *SchedulerCache) AddNode(obj interface{}) {
	node, ok := obj.(*corev1.Node)
	if !ok {
		klog.Errorf("Cannot convert to *corev1.Node: %v", obj)
		return
	}

	sc.Mutex.Lock()
	defer sc.Mutex.Unlock()

	err := sc.addNode(node)
	if err != nil {
		klog.Errorf("Failed to add node %s into cache: %v", node.Name, err)
		return
	}
	sc.NodeList = append(sc.NodeList, node.Name)
}

// UpdateNode (wraps updateNode) updates the specific node in sc.Nodes.
func (sc *SchedulerCache) UpdateNode(oldObj, newObj interface{}) {
	oldNode, ok := oldObj.(*corev1.Node)
	if !ok {
		klog.Errorf("Cannot convert oldObj to *corev1.Node: %v", oldObj)
		return
	}
	newNode, ok := newObj.(*corev1.Node)
	if !ok {
		klog.Errorf("Cannot convert newObj to *corev1.Node: %v", newObj)
		return
	}

	sc.Mutex.Lock()
	defer sc.Mutex.Unlock()

	err := sc.updateNode(oldNode, newNode)
	if err != nil {
		klog.Errorf("Failed to update node %v in cache: %v", oldNode.Name, err)
		return
	}
}

// DeleteNode (wraps deleteNode) deletes obj from sc.Nodes and sc.NodeList.
func (sc *SchedulerCache) DeleteNode(obj interface{}) {
	var node *corev1.Node
	switch t := obj.(type) {
	case *corev1.Node:
		node = t
	case cache.DeletedFinalStateUnknown:
		var ok bool
		node, ok = t.Obj.(*corev1.Node)
		if !ok {
			klog.Errorf("Cannot convert to *corev1.Node: %v", t.Obj)
			return
		}
	default:
		klog.Errorf("Cannot convert to *corev1.Node: %v", t)
		return
	}

	sc.Mutex.Lock()
	defer sc.Mutex.Unlock()

	err := sc.deleteNode(node)
	if err != nil {
		klog.Errorf("Failed to delete node %s from cache: %v", node.Name, err)
		return
	}

	for i, name := range sc.NodeList {
		if name == node.Name {
			sc.NodeList = append(sc.NodeList[:i], sc.NodeList[i+1:]...)
			break
		}
	}
}

/* handling PodGroup */

// setPodGroup sets the specific job in sc with the pg info.
func (sc *SchedulerCache) setPodGroup(pg *apis.PodGroup) error {
	job := getJobID(pg)
	// create job for pg if not found
	if _, found := sc.Jobs[job]; !found {
		sc.Jobs[job] = apis.NewJobInfo(job)
	}

	sc.Jobs[job].SetPodGroup(pg)

	// TODO: set default queue in admission
	if len(pg.Spec.Queue) == 0 {
		sc.Jobs[job].Queue = apis.QueueID(sc.defaultQueue)
	}

	return nil
}

// updatePodGroup updates sc with the pg info.
func (sc *SchedulerCache) updatePodGroup(pg *apis.PodGroup) error {
	return sc.setPodGroup(pg)
}

// deletePodGroup deletes job with jobID from sc.
func (sc *SchedulerCache) deletePodGroup(jobID apis.JobID) error {
	job, found := sc.Jobs[jobID]
	if !found {
		return fmt.Errorf("can not found job %v", jobID)
	}

	// delete pg and job
	job.UnsetPodGroup()
	sc.deleteJob(job)

	return nil
}

// AddPodGroupV1alpha1 (wraps setPodGroup) adds podgroup obj to sc.
func (sc *SchedulerCache) AddPodGroupV1alpha1(obj interface{}) {
	// assert obj as podgroup
	pg, ok := obj.(*schedulingv1alpha1.PodGroup)
	if !ok {
		klog.Errorf("Cannot convert to *schedulingv1alpha1.PodGroup: %v", obj)
		return
	}

	// convert podgroup to internal podgroup
	podgroup := scheduling.PodGroup{}
	if err := scheme.Scheme.Convert(pg, &podgroup, nil); err != nil {
		klog.Errorf("Failed to convert podgroup from %T to %T", pg, podgroup)
		return
	}

	// construct pg info with podgroup
	pgInfo := &apis.PodGroup{PodGroup: podgroup, Version: apis.PodGroupVersionV1Alpha1}
	klog.V(4).Infof("Add PodGroup(%s) into cache, spec(%#v)", pg.Name, pg.Spec)

	sc.Mutex.Lock()
	defer sc.Mutex.Unlock()

	// set the pg info
	if err := sc.setPodGroup(pgInfo); err != nil {
		klog.Errorf("Failed to add PodGroup %s into cache: %v", pg.Name, err)
		return
	}
}

// UpdatePodGroupV1alpha1 (wraps updatePodGroup) updates the specific podgroup in sc.
func (sc *SchedulerCache) UpdatePodGroupV1alpha1(oldObj, newObj interface{}) {
	oldPg, ok := oldObj.(*schedulingv1alpha1.PodGroup)
	if !ok {
		klog.Errorf("Cannot convert oldObj to *schedulingv1alpha1.SchedulingSpec: %v", oldObj)
		return
	}
	newPg, ok := newObj.(*schedulingv1alpha1.PodGroup)
	if !ok {
		klog.Errorf("Cannot convert newObj to *schedulingv1alpha1.SchedulingSpec: %v", newObj)
		return
	}

	if oldPg.ResourceVersion == newPg.ResourceVersion {
		return
	}

	// update required
	podgroup := scheduling.PodGroup{}
	if err := scheme.Scheme.Convert(newPg, &podgroup, nil); err != nil {
		klog.Errorf("Failed to convert podgroup from %T to %T", newPg, podgroup)
		return
	}

	pgInfo := &apis.PodGroup{PodGroup: podgroup, Version: apis.PodGroupVersionV1Alpha1}

	sc.Mutex.Lock()
	defer sc.Mutex.Unlock()

	if err := sc.updatePodGroup(pgInfo); err != nil {
		klog.Errorf("Failed to update SchedulingSpec %s into cache: %v", pgInfo.Name, err)
		return
	}
}

// DeletePodGroupV1alpha1 (wraps deletePodGroup) deletes the podgroup obj from sc.
func (sc *SchedulerCache) DeletePodGroupV1alpha1(obj interface{}) {
	// set podgroup based on obj type
	var pg *schedulingv1alpha1.PodGroup
	switch t := obj.(type) {
	case *schedulingv1alpha1.PodGroup:
		pg = t
	case cache.DeletedFinalStateUnknown:
		var ok bool
		pg, ok = t.Obj.(*schedulingv1alpha1.PodGroup)
		if !ok {
			klog.Errorf("Cannot convert to podgroup: %v", t.Obj)
			return
		}
	default:
		klog.Errorf("Cannot convert to podgroup: %v", t)
		return
	}

	jobID := apis.JobID(fmt.Sprintf("%s/%s", pg.Namespace, pg.Name))

	sc.Mutex.Lock()
	defer sc.Mutex.Unlock()

	if err := sc.deletePodGroup(jobID); err != nil {
		klog.Errorf("Failed to delete podgroup %s from cache: %v", pg.Name, err)
		return
	}
}

/* handling Queue */

func (sc *SchedulerCache) addQueue(queue *scheduling.Queue) {
	qi := apis.NewQueueInfo(queue)
	sc.Queues[qi.UID] = qi
}

func (sc *SchedulerCache) updateQueue(queue *scheduling.Queue) {
	sc.addQueue(queue)
}

func (sc *SchedulerCache) deleteQueue(queueID apis.QueueID) {
	delete(sc.Queues, queueID)
}

// AddQueueV1alpha1 (wraps addQueue) adds the queue obj to sc.
func (sc *SchedulerCache) AddQueueV1alpha1(obj interface{}) {
	q, ok := obj.(*schedulingv1alpha1.Queue)
	if !ok {
		klog.Errorf("Cannot convert to *schedulingv1alpha1.Queue: %v", obj)
		return
	}

	queue := &scheduling.Queue{}
	if err := scheme.Scheme.Convert(q, queue, nil); err != nil {
		klog.Errorf("Failed to convert queue from %T to %T", q, queue)
		return
	}

	sc.Mutex.Lock()
	defer sc.Mutex.Unlock()

	klog.V(4).Infof("Add Queue(%s) into cache, spec(%#v)", q.Name, q.Spec)
	sc.addQueue(queue)
}

// UpdateQueueV1alpha1 (wraps updateQueue) updates the specific queue object in sc.
func (sc *SchedulerCache) UpdateQueueV1alpha1(oldObj, newObj interface{}) {
	oldQ, ok := oldObj.(*schedulingv1alpha1.Queue)
	if !ok {
		klog.Errorf("Cannot convert oldObj to *schedulingv1alpha1.Queue: %v", oldObj)
		return
	}
	newQ, ok := newObj.(*schedulingv1alpha1.Queue)
	if !ok {
		klog.Errorf("Cannot convert newObj to *schedulingv1alpha1.Queue: %v", newObj)
		return
	}

	if oldQ.ResourceVersion == newQ.ResourceVersion {
		return
	}

	newQueue := &scheduling.Queue{}
	if err := scheme.Scheme.Convert(newQ, newQueue, nil); err != nil {
		klog.Errorf("Failed to convert queue from %T to %T", newQ, newQueue)
		return
	}

	sc.Mutex.Lock()
	defer sc.Mutex.Unlock()
	sc.updateQueue(newQueue)
}

// DeleteQueueV1alpha1 (wraps deleteQueue) deletes the queue obj from sc.
func (sc *SchedulerCache) DeleteQueueV1alpha1(obj interface{}) {
	var q *schedulingv1alpha1.Queue

	switch t := obj.(type) {
	case *schedulingv1alpha1.Queue:
		q = t
	case cache.DeletedFinalStateUnknown:
		var ok bool
		q, ok = t.Obj.(*schedulingv1alpha1.Queue)
		if !ok {
			klog.Errorf("Cannot convert to *schedulingv1alpha1.Queue: %v", t.Obj)
			return
		}
	default:
		klog.Errorf("Cannot convert to *schedulingv1alpha1.Queue: %v", t)
		return
	}

	sc.Mutex.Lock()
	defer sc.Mutex.Unlock()
	sc.deleteQueue(apis.QueueID(q.Name))
}

/* handling PriorityClass  */

func (sc *SchedulerCache) deletePriorityClass(pc *schedulingv1.PriorityClass) {
	if pc.GlobalDefault {
		sc.defaultPriorityClass = nil
		sc.defaultPriority = 0
	}
	delete(sc.PriorityClasses, pc.Name)
}

func (sc *SchedulerCache) addPriorityClass(pc *schedulingv1.PriorityClass) {
	if pc.GlobalDefault {
		if sc.defaultPriorityClass != nil {
			klog.Errorf("Updated default priority class from <%s> to <%s> forcefully.",
				sc.defaultPriorityClass.Name, pc.Name)
		}
		sc.defaultPriorityClass = pc
		sc.defaultPriority = pc.Value
	}

	sc.PriorityClasses[pc.Name] = pc
}

// DeletePriorityClass (wraps deletePriorityClass) deletes the PriorityClass obj from sc.
func (sc *SchedulerCache) DeletePriorityClass(obj interface{}) {
	var pc *schedulingv1.PriorityClass
	switch t := obj.(type) {
	case *schedulingv1.PriorityClass:
		pc = t
	case cache.DeletedFinalStateUnknown:
		var ok bool
		pc, ok = t.Obj.(*schedulingv1.PriorityClass)
		if !ok {
			klog.Errorf("Cannot convert to *schedulingv1.PriorityClass: %v", t.Obj)
			return
		}
	default:
		klog.Errorf("Cannot convert to *schedulingv1.PriorityClass: %v", t)
		return
	}

	sc.Mutex.Lock()
	defer sc.Mutex.Unlock()

	sc.deletePriorityClass(pc)
}

// UpdatePriorityClass updates the specific PriorityClass in sc.
func (sc *SchedulerCache) UpdatePriorityClass(oldObj, newObj interface{}) {
	oldPC, ok := oldObj.(*schedulingv1.PriorityClass)
	if !ok {
		klog.Errorf("Cannot convert oldObj to *schedulingv1.PriorityClass: %v", oldObj)

		return
	}

	newPC, ok := newObj.(*schedulingv1.PriorityClass)
	if !ok {
		klog.Errorf("Cannot convert newObj to *schedulingv1.PriorityClass: %v", newObj)
		return
	}

	sc.Mutex.Lock()
	defer sc.Mutex.Unlock()

	sc.deletePriorityClass(oldPC)
	sc.addPriorityClass(newPC)
}

// AddPriorityClass (wraps addPriorityClass) adds the PriorityClass obj to sc.
func (sc *SchedulerCache) AddPriorityClass(obj interface{}) {
	pc, ok := obj.(*schedulingv1.PriorityClass)
	if !ok {
		klog.Errorf("Cannot convert to *schedulingv1.PriorityClass: %v", obj)
		return
	}

	sc.Mutex.Lock()
	defer sc.Mutex.Unlock()

	sc.addPriorityClass(pc)
}

/* handling ResourceQuota  */

func (sc *SchedulerCache) updateResourceQuota(quota *corev1.ResourceQuota) {
	collection, ok := sc.NamespaceCollection[quota.Namespace]
	if !ok {
		collection = apis.NewNamespaceCollection(quota.Namespace)
		sc.NamespaceCollection[quota.Namespace] = collection
	}
	collection.Update(quota)
}

func (sc *SchedulerCache) deleteResourceQuota(quota *corev1.ResourceQuota) {
	collection, ok := sc.NamespaceCollection[quota.Namespace]
	if !ok {
		return
	}
	collection.Delete(quota)
}

// AddResourceQuota adds ResourceQuota obj to sc.
func (sc *SchedulerCache) AddResourceQuota(obj interface{}) {
	var r *corev1.ResourceQuota
	switch t := obj.(type) {
	case *corev1.ResourceQuota:
		r = t
	default:
		klog.Errorf("Cannot convert to *corev1.ResourceQuota: %v", t)
		return
	}

	sc.Mutex.Lock()
	defer sc.Mutex.Unlock()

	klog.V(3).Infof("Add ResourceQuota <%s/%v> in cache, with spec: %v.", r.Namespace, r.Name, r.Spec.Hard)
	sc.updateResourceQuota(r)
}

// UpdateResourceQuota updates the specific ResourceQuota in sc.
func (sc *SchedulerCache) UpdateResourceQuota(oldObj, newObj interface{}) {
	newR, ok := newObj.(*corev1.ResourceQuota)
	if !ok {
		klog.Errorf("Cannot convert newObj to *corev1.ResourceQuota: %v", newObj)
		return
	}

	sc.Mutex.Lock()
	defer sc.Mutex.Unlock()

	klog.V(3).Infof("Update ResourceQuota <%s/%v> in cache, with spec: %v.", newR.Namespace, newR.Name, newR.Spec.Hard)
	sc.updateResourceQuota(newR)
}

// DeleteResourceQuota deletes the ResourceQuota obj from sc.
func (sc *SchedulerCache) DeleteResourceQuota(obj interface{}) {
	var r *corev1.ResourceQuota
	switch t := obj.(type) {
	case *corev1.ResourceQuota:
		r = t
	case cache.DeletedFinalStateUnknown:
		var ok bool
		r, ok = t.Obj.(*corev1.ResourceQuota)
		if !ok {
			klog.Errorf("Cannot convert to *corev1.ResourceQuota: %v", t.Obj)
			return
		}
	default:
		klog.Errorf("Cannot convert to *corev1.ResourceQuota: %v", t)
		return
	}

	sc.Mutex.Lock()
	defer sc.Mutex.Unlock()

	klog.V(3).Infof("Delete ResourceQuota <%s/%v> in cache", r.Namespace, r.Name)
	sc.deleteResourceQuota(r)
}

/* handling NUMA Topology */

// getNumaInfo constructs an apis.NumaTopoInfo instance from the CRD nodeinfov1alpha1.Numatopology.
func getNumaInfo(srcInfo *nodeinfov1alpha1.Numatopology) *apis.NumaTopoInfo {
	numaInfo := &apis.NumaTopoInfo{
		Namespace:   srcInfo.Namespace,
		Name:        srcInfo.Name,
		Policies:    map[nodeinfov1alpha1.PolicyName]string{},
		NumaResMap:  map[string]*apis.ResourceInfo{},
		CPUDetail:   topology.CPUDetails{},
		ResReserved: make(corev1.ResourceList),
	}

	policies := srcInfo.Spec.Policies
	for name, policy := range policies {
		numaInfo.Policies[name] = policy
	}

	numaResMap := srcInfo.Spec.NumaResMap
	for name, resInfo := range numaResMap {
		tmp := apis.ResourceInfo{}
		tmp.Capacity = resInfo.Capacity
		tmp.Allocatable = cpuset.MustParse(resInfo.Allocatable)
		numaInfo.NumaResMap[name] = &tmp
	}

	cpuDetail := srcInfo.Spec.CPUDetail
	for key, detail := range cpuDetail {
		cpuID, _ := strconv.Atoi(key)
		numaInfo.CPUDetail[cpuID] = topology.CPUInfo{
			NUMANodeID: detail.NUMANodeID,
			SocketID:   detail.SocketID,
			CoreID:     detail.CoreID,
		}
	}

	resReserved, err := apis.ParseResourceList(srcInfo.Spec.ResReserved)
	if err != nil {
		klog.Errorf("ParseResourceList failed, err=%v", err)
	} else {
		numaInfo.ResReserved = resReserved
	}

	return numaInfo
}

// addNumaInfo updates sc.Nodes based on the input numa info.
func (sc *SchedulerCache) addNumaInfo(info *nodeinfov1alpha1.Numatopology) error {
	if sc.Nodes[info.Name] == nil {
		sc.Nodes[info.Name] = apis.NewNodeInfo(nil)
		sc.Nodes[info.Name].Name = info.Name
	}

	if sc.Nodes[info.Name].NumaInfo == nil {
		// create
		sc.Nodes[info.Name].NumaInfo = getNumaInfo(info)
		sc.Nodes[info.Name].NumaChgFlag = apis.NumaInfoMoreFlag
	} else {
		// update
		newLocalInfo := getNumaInfo(info)
		if sc.Nodes[info.Name].NumaInfo.Compare(newLocalInfo) {
			sc.Nodes[info.Name].NumaChgFlag = apis.NumaInfoMoreFlag
		} else {
			sc.Nodes[info.Name].NumaChgFlag = apis.NumaInfoLessFlag
		}
		sc.Nodes[info.Name].NumaInfo = newLocalInfo
	}

	for resName, NumaResInfo := range sc.Nodes[info.Name].NumaInfo.NumaResMap {
		klog.V(3).Infof("resource %s Allocatable %v on node[%s] into cache",
			resName, NumaResInfo, info.Name)
	}

	klog.V(3).Infof("Policies %v on node[%s] into cache, change= %v",
		sc.Nodes[info.Name].NumaInfo.Policies, info.Name, sc.Nodes[info.Name].NumaChgFlag)
	return nil
}

// deleteNumaInfo updates sc.Nodes based on the input numa info.
func (sc *SchedulerCache) deleteNumaInfo(info *nodeinfov1alpha1.Numatopology) {
	if sc.Nodes[info.Name] != nil {
		sc.Nodes[info.Name].NumaInfo = nil
		sc.Nodes[info.Name].NumaChgFlag = apis.NumaInfoResetFlag
		klog.V(3).Infof("delete numaInfo in cache for node<%s>", info.Name)
	}
}

// AddNumaInfoV1alpha1 (wraps addNumaInfo) updates sc.Nodes (NumaInfo and NumaChgFlag) based on the input obj.
func (sc *SchedulerCache) AddNumaInfoV1alpha1(obj interface{}) {
	ss, ok := obj.(*nodeinfov1alpha1.Numatopology)
	if !ok {
		klog.Errorf("Cannot convert oldObj to *nodeinfov1alpha1.Numatopology: %v", obj)
		return
	}

	sc.Mutex.Lock()
	defer sc.Mutex.Unlock()

	_ = sc.addNumaInfo(ss)
}

// UpdateNumaInfoV1alpha1 (wraps addNumaInfo) updates sc.Nodes (NumaInfo and NumaChgFlag) based on the input obj.
func (sc *SchedulerCache) UpdateNumaInfoV1alpha1(oldObj, newObj interface{}) {
	nt, ok := newObj.(*nodeinfov1alpha1.Numatopology)
	if !ok {
		klog.Errorf("Cannot convert oldObj to *nodeinfov1alpha1.Numatopology: %v", newObj)
		return
	}

	sc.Mutex.Lock()
	defer sc.Mutex.Unlock()

	_ = sc.addNumaInfo(nt)
	klog.V(3).Infof("update numaInfo<%s> in cache, with spec: Policy: %v, resMap: %v",
		nt.Name, nt.Spec.Policies, nt.Spec.NumaResMap)
}

// DeleteNumaInfoV1alpha1 (wraps deleteNumaInfo) updates sc.Nodes (NumaInfo and NumaChgFlag) based on the input obj.
func (sc *SchedulerCache) DeleteNumaInfoV1alpha1(obj interface{}) {
	var nt *nodeinfov1alpha1.Numatopology
	switch t := obj.(type) {
	case *nodeinfov1alpha1.Numatopology:
		nt = t
	case cache.DeletedFinalStateUnknown:
		var ok bool
		nt, ok = t.Obj.(*nodeinfov1alpha1.Numatopology)
		if !ok {
			klog.Errorf("Cannot convert to Numatopo: %v", t.Obj)
			return
		}
	default:
		klog.Errorf("Cannot convert to Numatopo: %v", t)
		return
	}

	sc.Mutex.Lock()
	defer sc.Mutex.Unlock()

	sc.deleteNumaInfo(nt)
	klog.V(3).Infof("Delete numaInfo<%s> from cache, with spec: Policy: %v, resMap: %v",
		nt.Name, nt.Spec.Policies, nt.Spec.NumaResMap)
}
