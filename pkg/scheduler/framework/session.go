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

package framework

import (
	"fmt"
	"github.com/hliangzhao/volcano/pkg/apis/scheduling"
	"github.com/hliangzhao/volcano/pkg/scheduler/apis"
	"github.com/hliangzhao/volcano/pkg/scheduler/cache"
	"github.com/hliangzhao/volcano/pkg/scheduler/conf"
	"github.com/hliangzhao/volcano/pkg/scheduler/metrics"
	"github.com/hliangzhao/volcano/pkg/scheduler/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/scheduler/framework/plugins/volumebinding"
)

type Session struct {
	UID types.UID

	kubeClient      kubernetes.Interface
	cache           cache.Cache
	informerFactory informers.SharedInformerFactory

	TotalResource *apis.Resource
	// podGroupStatus cache podgroup status during schedule
	// This should not be mutated after initiated.
	podGroupStatus map[apis.JobID]scheduling.PodGroupStatus

	Jobs           map[apis.JobID]*apis.JobInfo
	Nodes          map[string]*apis.NodeInfo
	RevocableNodes map[string]*apis.NodeInfo
	Queues         map[apis.QueueID]*apis.QueueInfo
	NamespaceInfo  map[apis.NamespaceName]*apis.NamespaceInfo

	Tiers          []conf.Tier
	Configurations []conf.Configuration
	NodeList       []*apis.NodeInfo

	plugins       map[string]Plugin
	eventHandlers []*EventHandler

	jobOrderFns       map[string]apis.CompareFn
	queueOrderFns     map[string]apis.CompareFn
	taskOrderFns      map[string]apis.CompareFn
	namespaceOrderFns map[string]apis.CompareFn
	clusterOrderFns   map[string]apis.CompareFn
	predicateFns      map[string]apis.PredicateFn
	bestNodeFns       map[string]apis.BestNodeFn
	nodeOrderFns      map[string]apis.NodeOrderFn
	batchNodeOrderFns map[string]apis.BatchNodeOrderFn
	nodeMapFns        map[string]apis.NodeMapFn
	nodeReduceFns     map[string]apis.NodeReduceFn
	preemptableFns    map[string]apis.EvictableFn
	reclaimableFns    map[string]apis.EvictableFn
	overUsedFns       map[string]apis.ValidateFn
	underUsedFns      map[string]apis.UnderUsedResourceFn
	jobReadyFns       map[string]apis.ValidateFn
	jobPipelinedFns   map[string]apis.VoteFn
	jobValidFns       map[string]apis.ValidateExFn
	jobEnqueuableFns  map[string]apis.VoteFn
	jobEnqueuedFns    map[string]apis.JobEnqueuedFn
	targetJobFns      map[string]apis.TargetJobFn
	reservedNodesFns  map[string]apis.ReservedNodesFn
	victimTasksFns    map[string]apis.VictimTasksFn
	jobStarvingFns    map[string]apis.ValidateFn
}

func openSession(cache cache.Cache) *Session {
	sess := &Session{
		UID:             uuid.NewUUID(),
		kubeClient:      cache.Client(),
		cache:           cache,
		informerFactory: cache.SharedInformerFactory(),

		TotalResource:  apis.EmptyResource(),
		podGroupStatus: map[apis.JobID]scheduling.PodGroupStatus{},

		Jobs:           map[apis.JobID]*apis.JobInfo{},
		Nodes:          map[string]*apis.NodeInfo{},
		RevocableNodes: map[string]*apis.NodeInfo{},
		Queues:         map[apis.QueueID]*apis.QueueInfo{},

		plugins:           map[string]Plugin{},
		jobOrderFns:       map[string]apis.CompareFn{},
		queueOrderFns:     map[string]apis.CompareFn{},
		taskOrderFns:      map[string]apis.CompareFn{},
		namespaceOrderFns: map[string]apis.CompareFn{},
		clusterOrderFns:   map[string]apis.CompareFn{},
		predicateFns:      map[string]apis.PredicateFn{},
		bestNodeFns:       map[string]apis.BestNodeFn{},
		nodeOrderFns:      map[string]apis.NodeOrderFn{},
		batchNodeOrderFns: map[string]apis.BatchNodeOrderFn{},
		nodeMapFns:        map[string]apis.NodeMapFn{},
		nodeReduceFns:     map[string]apis.NodeReduceFn{},
		preemptableFns:    map[string]apis.EvictableFn{},
		reclaimableFns:    map[string]apis.EvictableFn{},
		overUsedFns:       map[string]apis.ValidateFn{},
		underUsedFns:      map[string]apis.UnderUsedResourceFn{},
		jobReadyFns:       map[string]apis.ValidateFn{},
		jobPipelinedFns:   map[string]apis.VoteFn{},
		jobValidFns:       map[string]apis.ValidateExFn{},
		jobEnqueuableFns:  map[string]apis.VoteFn{},
		jobEnqueuedFns:    map[string]apis.JobEnqueuedFn{},
		targetJobFns:      map[string]apis.TargetJobFn{},
		reservedNodesFns:  map[string]apis.ReservedNodesFn{},
		victimTasksFns:    map[string]apis.VictimTasksFn{},
		jobStarvingFns:    map[string]apis.ValidateFn{},
	}

	snapshot := cache.Snapshot()

	sess.Jobs = snapshot.Jobs
	for _, job := range sess.Jobs {
		// only conditions will be updated periodically
		if job.PodGroup != nil && job.PodGroup.Status.Conditions != nil {
			sess.podGroupStatus[job.UID] = *job.PodGroup.Status.DeepCopy()
		}

		if vjr := sess.JobValid(job); vjr != nil {
			// job is not pass, update its condition to unschedulable
			if !vjr.Pass {
				jc := &scheduling.PodGroupCondition{
					Type:               scheduling.PodGroupUnschedulableType,
					Status:             corev1.ConditionTrue,
					LastTransitionTime: metav1.Now(),
					TransitionID:       string(sess.UID),
					Reason:             vjr.Reason,
					Message:            vjr.Message,
				}

				if err := sess.UpdatePodGroupCondition(job, jc); err != nil {
					klog.Errorf("Failed to update job condition: %v", err)
				}
			}

			delete(sess.Jobs, job.UID)
		}
	}

	sess.NodeList = utils.GetNodeList(snapshot.Nodes, snapshot.NodeList)
	sess.Nodes = snapshot.Nodes
	sess.RevocableNodes = snapshot.RevocableNodes
	sess.Queues = snapshot.Queues
	sess.NamespaceInfo = snapshot.NamespaceInfo

	// calculate all nodes' resource only once in each schedule cycle, other plugins can clone it when need
	for _, n := range sess.Nodes {
		sess.TotalResource.Add(n.Allocatable)
	}

	klog.V(3).Infof("Open Session %v with <%d> Job and <%d> Queues",
		sess.UID, len(sess.Jobs), len(sess.Queues))

	return sess
}

func closeSession(sess *Session) {
	ju := newJobUpdater(sess)
	ju.UpdateAll()

	sess.Jobs = nil
	sess.Nodes = nil
	sess.RevocableNodes = nil
	sess.plugins = nil
	sess.eventHandlers = nil
	sess.jobOrderFns = nil
	sess.namespaceOrderFns = nil
	sess.queueOrderFns = nil
	sess.clusterOrderFns = nil
	sess.NodeList = nil
	sess.TotalResource = nil

	klog.V(3).Infof("Close Session %v", sess.UID)
}

// String return nodes and jobs information in the session.
func (sess Session) String() string {
	msg := fmt.Sprintf("Session %v: \n", sess.UID)

	for _, job := range sess.Jobs {
		msg = fmt.Sprintf("%s%v\n", msg, job)
	}

	for _, node := range sess.Nodes {
		msg = fmt.Sprintf("%s%v\n", msg, node)
	}

	return msg
}

// InformerFactory returns the scheduler ShareInformerFactory
func (sess Session) InformerFactory() informers.SharedInformerFactory {
	return sess.informerFactory
}

// KubeClient returns the kubernetes client.
func (sess Session) KubeClient() kubernetes.Interface {
	return sess.kubeClient
}

// AddEventHandler add event handlers.
func (sess *Session) AddEventHandler(eh *EventHandler) {
	sess.eventHandlers = append(sess.eventHandlers, eh)
}

// UpdateSchedulerNumaInfo update SchedulerNumaInfo.
func (sess *Session) UpdateSchedulerNumaInfo(AllocatedSets map[string]apis.ResNumaSets) {
	_ = sess.cache.UpdateSchedulerNumaInfo(AllocatedSets)
}

// UpdatePodGroupCondition update job condition accordingly.
func (sess *Session) UpdatePodGroupCondition(jobInfo *apis.JobInfo, cond *scheduling.PodGroupCondition) error {
	job, ok := sess.Jobs[jobInfo.UID]
	if !ok {
		return fmt.Errorf("failed to find job <%s/%s>", jobInfo.Namespace, jobInfo.Name)
	}

	index := -1
	for i, c := range job.PodGroup.Status.Conditions {
		if c.Type == cond.Type {
			index = i
			break
		}
	}

	// Update condition to the new condition.
	if index < 0 {
		job.PodGroup.Status.Conditions = append(job.PodGroup.Status.Conditions, *cond)
	} else {
		job.PodGroup.Status.Conditions[index] = *cond
	}

	return nil
}

// BindPodGroup bind PodGroup to specified cluster.
func (sess *Session) BindPodGroup(job *apis.JobInfo, cluster string) error {
	return sess.cache.BindPodGroup(job, cluster)
}

// jobStatus sets status of given jobInfo.
func jobStatus(sess *Session, jobInfo *apis.JobInfo) scheduling.PodGroupStatus {
	status := jobInfo.PodGroup.Status

	unschedulable := false
	for _, c := range status.Conditions {
		if c.Type == scheduling.PodGroupUnschedulableType &&
			c.Status == corev1.ConditionTrue &&
			c.TransitionID == string(sess.UID) {
			unschedulable = true
			break
		}
	}

	// If running tasks && unschedulable, unknown phase
	if len(jobInfo.TaskStatusIndex[apis.Running]) != 0 && unschedulable {
		status.Phase = scheduling.PodGroupUnknown
	} else {
		allocated := 0
		for status, tasks := range jobInfo.TaskStatusIndex {
			if apis.AllocatedStatus(status) || status == apis.Succeeded {
				allocated += len(tasks)
			}
		}

		// If there are enough allocated resource, it's running
		if int32(allocated) >= jobInfo.PodGroup.Spec.MinMember {
			status.Phase = scheduling.PodGroupRunning
		} else if jobInfo.PodGroup.Status.Phase != scheduling.PodGroupInqueue {
			status.Phase = scheduling.PodGroupPending
		}
	}

	status.Running = int32(len(jobInfo.TaskStatusIndex[apis.Running]))
	status.Failed = int32(len(jobInfo.TaskStatusIndex[apis.Failed]))
	status.Succeeded = int32(len(jobInfo.TaskStatusIndex[apis.Succeeded]))

	return status
}

func (sess *Session) Statement() *Statement {
	return &Statement{
		sess: sess,
	}
}

func (sess *Session) dispatch(task *apis.TaskInfo, volumes *volumebinding.PodVolumes) error {
	if err := sess.cache.AddBindTask(task); err != nil {
		return err
	}

	// Update status in session
	if job, found := sess.Jobs[task.Job]; found {
		if err := job.UpdateTaskStatus(task, apis.Binding); err != nil {
			klog.Errorf("Failed to update task <%v/%v> status to %v in Session <%v>: %v",
				task.Namespace, task.Name, apis.Binding, sess.UID, err)
			return err
		}
	} else {
		klog.Errorf("Failed to find Job <%s> in Session <%s> index when binding.",
			task.Job, sess.UID)
		return fmt.Errorf("failed to find job %s", task.Job)
	}

	metrics.UpdateTaskScheduleDuration(metrics.Duration(task.Pod.CreationTimestamp.Time))
	return nil
}

func (sess *Session) Allocate(task *apis.TaskInfo, nodeInfo *apis.NodeInfo) error {
	// TODO: this Allocate is similar to statement.Allocate. Why statement.Allocate calls this directly?

	podVolumes, err := sess.cache.GetPodVolumes(task, nodeInfo.Node)
	if err != nil {
		return err
	}

	hostname := nodeInfo.Name
	if err := sess.cache.AllocateVolumes(task, hostname, podVolumes); err != nil {
		return err
	}

	task.Pod.Spec.NodeName = hostname
	task.PodVolumes = podVolumes

	// Only update status in session
	job, found := sess.Jobs[task.Job]
	if found {
		if err := job.UpdateTaskStatus(task, apis.Allocated); err != nil {
			klog.Errorf("Failed to update task <%v/%v> status to %v in Session <%v>: %v",
				task.Namespace, task.Name, apis.Allocated, sess.UID, err)
			return err
		}
	} else {
		klog.Errorf("Failed to find Job <%s> in Session <%s> index when binding.",
			task.Job, sess.UID)
		return fmt.Errorf("failed to find job %s", task.Job)
	}

	task.NodeName = hostname

	if node, found := sess.Nodes[hostname]; found {
		if err := node.AddTask(task); err != nil {
			klog.Errorf("Failed to add task <%v/%v> to node <%v> in Session <%v>: %v",
				task.Namespace, task.Name, hostname, sess.UID, err)
			return err
		}
		klog.V(3).Infof("After allocated Task <%v/%v> to Node <%v>: idle <%v>, used <%v>, releasing <%v>",
			task.Namespace, task.Name, node.Name, node.Idle, node.Used, node.Releasing)
	} else {
		klog.Errorf("Failed to find Node <%s> in Session <%s> index when binding.",
			hostname, sess.UID)
		return fmt.Errorf("failed to find node %s", hostname)
	}

	// Callbacks
	for _, eh := range sess.eventHandlers {
		if eh.AllocateFunc != nil {
			eh.AllocateFunc(&Event{
				Task: task,
			})
		}
	}

	if sess.JobReady(job) {
		for _, task := range job.TaskStatusIndex[apis.Allocated] {
			if err := sess.dispatch(task, podVolumes); err != nil {
				klog.Errorf("Failed to dispatch task <%v/%v>: %v",
					task.Namespace, task.Name, err)
				return err
			}
		}
	}

	return nil
}

func (sess *Session) Evict(reclaimee *apis.TaskInfo, reason string) error {
	// TODO: this Evict is similar to statement.Evict. Why statement.Evict calls this directly?

	if err := sess.cache.Evict(reclaimee, reason); err != nil {
		return err
	}

	// Update status in session
	job, found := sess.Jobs[reclaimee.Job]
	if found {
		if err := job.UpdateTaskStatus(reclaimee, apis.Releasing); err != nil {
			klog.Errorf("Failed to update task <%v/%v> status to %v in Session <%v>: %v",
				reclaimee.Namespace, reclaimee.Name, apis.Releasing, sess.UID, err)
			return err
		}
	} else {
		klog.Errorf("Failed to find Job <%s> in Session <%s> index when binding.",
			reclaimee.Job, sess.UID)
		return fmt.Errorf("failed to find job %s", reclaimee.Job)
	}

	// Update task in node.
	if node, found := sess.Nodes[reclaimee.NodeName]; found {
		if err := node.UpdateTask(reclaimee); err != nil {
			klog.Errorf("Failed to update task <%v/%v> in Session <%v>: %v",
				reclaimee.Namespace, reclaimee.Name, sess.UID, err)
			return err
		}
	}

	for _, eh := range sess.eventHandlers {
		if eh.DeallocateFunc != nil {
			eh.DeallocateFunc(&Event{
				Task: reclaimee,
			})
		}
	}

	return nil
}

func (sess *Session) Pipeline(task *apis.TaskInfo, hostname string) error {
	// TODO: this Pipeline is similar to statement.Pipeline. Why statement.Pipeline calls this directly?

	// Only update status in session
	job, found := sess.Jobs[task.Job]
	if found {
		if err := job.UpdateTaskStatus(task, apis.Pipelined); err != nil {
			klog.Errorf("Failed to update task <%v/%v> status to %v in Session <%v>: %v",
				task.Namespace, task.Name, apis.Pipelined, sess.UID, err)
			return err
		}
	} else {
		klog.Errorf("Failed to find Job <%s> in Session <%s> index when binding.",
			task.Job, sess.UID)
		return fmt.Errorf("failed to find job %s when binding", task.Job)
	}

	task.NodeName = hostname

	if node, found := sess.Nodes[hostname]; found {
		if err := node.AddTask(task); err != nil {
			klog.Errorf("Failed to add task <%v/%v> to node <%v> in Session <%v>: %v",
				task.Namespace, task.Name, hostname, sess.UID, err)
			return err
		}
		klog.V(3).Infof("After added Task <%v/%v> to Node <%v>: idle <%v>, used <%v>, releasing <%v>",
			task.Namespace, task.Name, node.Name, node.Idle, node.Used, node.Releasing)
	} else {
		klog.Errorf("Failed to find Node <%s> in Session <%s> index when binding.",
			hostname, sess.UID)
		return fmt.Errorf("failed to find node %s", hostname)
	}

	for _, eh := range sess.eventHandlers {
		if eh.AllocateFunc != nil {
			eh.AllocateFunc(&Event{
				Task: task,
			})
		}
	}

	return nil
}
