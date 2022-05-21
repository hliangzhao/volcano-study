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

package tasktopology

import (
	"fmt"
	"github.com/hliangzhao/volcano/pkg/scheduler/apis"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"math"
	"sort"
	"strings"
)

type topologyType int

const (
	selfAntiAffinity topologyType = iota
	interAntiAffinity
	selfAffinity
	interAffinity
)

// map[topologyType]priority, the larger number means the higher priority
var affinityPriority = map[topologyType]int{
	selfAntiAffinity:  4,
	interAffinity:     3,
	selfAffinity:      2,
	interAntiAffinity: 1,
}

// JobManager is struct used to save infos about affinity and buckets of a job.
type JobManager struct {
	jobID apis.JobID

	buckets     []*Bucket
	podInBucket map[types.UID]int
	podInTask   map[types.UID]string
	// taskOverPod map[string]map[types.UID]struct{}

	taskAffinityPriority map[string]int // [taskName] -> priority
	taskExistOrder       map[string]int
	interAffinity        map[string]map[string]struct{} // [taskName]->[taskName]
	selfAffinity         map[string]struct{}
	interAntiAffinity    map[string]map[string]struct{} // [taskName]->[taskName]
	selfAntiAffinity     map[string]struct{}

	bucketMaxSize int
	nodeTaskSet   map[string]map[string]int // [nodeName]->{[taskName]->[taskNum (podNum)]}
}

func (jm *JobManager) String() string {
	// saa: selfAntiAffinity
	// iaa: interAntiAffinity
	// sa: selfAffinity
	// ia: interAffinity
	msg := []string{
		fmt.Sprintf("%s - job %s max %d || saa: %v - iaa: %v - sa: %v - ia: %v || priority: %v - order: %v || ",
			PluginName, jm.jobID, jm.bucketMaxSize,
			jm.selfAntiAffinity, jm.interAntiAffinity,
			jm.selfAffinity, jm.interAffinity,
			jm.taskAffinityPriority, jm.taskExistOrder,
		),
	}

	for _, bucket := range jm.buckets {
		bucketMsg := fmt.Sprintf("b:%d -- ", bucket.index)
		var info []string
		for _, task := range bucket.tasks {
			info = append(info, task.Pod.Name)
		}
		bucketMsg += strings.Join(info, ", ")
		bucketMsg += "|"

		info = nil
		for nodeName, count := range bucket.node {
			info = append(info, fmt.Sprintf("n%s-%d", nodeName, count))
		}
		bucketMsg += strings.Join(info, ", ")

		msg = append(msg, "["+bucketMsg+"]")
	}
	return strings.Join(msg, " ")
}

func NewJobManager(jobID apis.JobID) *JobManager {
	return &JobManager{
		jobID: jobID,

		buckets:     make([]*Bucket, 0),
		podInBucket: make(map[types.UID]int), // pod-uid: bucket-index
		podInTask:   make(map[types.UID]string),
		// taskOverPod: make(map[string]map[types.UID]struct{}),

		taskAffinityPriority: make(map[string]int), // task-name: priority
		taskExistOrder:       make(map[string]int), // task-name: order
		interAffinity:        make(map[string]map[string]struct{}),
		interAntiAffinity:    make(map[string]map[string]struct{}),
		selfAffinity:         make(map[string]struct{}),
		selfAntiAffinity:     make(map[string]struct{}),

		bucketMaxSize: 0,
		nodeTaskSet:   make(map[string]map[string]int),
	}
}

// NewBucket creates a new bucket and sets its index in jm.
func (jm *JobManager) NewBucket() *Bucket {
	bucket := NewBucket()
	bucket.index = len(jm.buckets)
	jm.buckets = append(jm.buckets, bucket)
	return bucket
}

// GetBucket get bucket inside which task has been.
func (jm *JobManager) GetBucket(task *apis.TaskInfo) *Bucket {
	index, ok := jm.podInBucket[task.Pod.UID]
	if !ok || index == OutOfBucket {
		return nil
	}
	return jm.buckets[index]
}

func (jm *JobManager) buildTaskInfo(tasks map[apis.TaskID]*apis.TaskInfo) []*apis.TaskInfo {
	taskWithoutBucket := make([]*apis.TaskInfo, 0, len(tasks))
	for _, task := range tasks {
		pod := task.Pod

		// if this task is not the task volcano cares about (no `batchv1alpha1.TaskSpecKey`) found,
		// mark it as out-of-bucket
		taskName := getTaskName(task)
		if taskName == "" {
			jm.MarkOutOfBucket(pod.UID)
			continue
		}
		// if this task does not have topology info, topologyPlugin will not care it
		if _, hasTopology := jm.taskAffinityPriority[taskName]; !hasTopology {
			jm.MarkOutOfBucket(pod.UID)
			continue
		}

		// this task is interested, add it to taskWithoutBucket and waiting for add
		jm.podInTask[pod.UID] = taskName
		// taskSet, ok := jm.taskOverPod[taskName]
		// if !ok {
		// 	taskSet = make(map[types.UID]struct{})
		// 	jm.taskOverPod[taskName] = taskSet
		// }
		// taskSet[pod.UID] = struct{}{}
		taskWithoutBucket = append(taskWithoutBucket, task)
	}
	return taskWithoutBucket
}

// ConstructBucket builds bucket for tasks.
func (jm *JobManager) ConstructBucket(tasks map[apis.TaskID]*apis.TaskInfo) {
	taskWithoutBucket := jm.buildTaskInfo(tasks)

	o := TaskOrder{
		tasks:   taskWithoutBucket,
		manager: jm,
	}
	sort.Sort(sort.Reverse(&o))
	jm.buildBucket(o.tasks)
}

func (jm *JobManager) checkTaskSetAffinity(taskName string, taskNameSet map[string]int, onlyAnti bool) int {
	bucketPodAff := 0

	if taskName == "" {
		return bucketPodAff
	}

	for taskNameInBucket, count := range taskNameSet {
		theSameTask := taskNameInBucket == taskName

		if !onlyAnti {
			affinity := false
			if theSameTask {
				_, affinity = jm.selfAffinity[taskName]
			} else {
				_, affinity = jm.interAffinity[taskName][taskNameInBucket]
			}
			if affinity {
				bucketPodAff += count
			}
		}

		antiAffinity := false
		if theSameTask {
			_, antiAffinity = jm.selfAntiAffinity[taskName]
		} else {
			_, antiAffinity = jm.interAntiAffinity[taskName][taskNameInBucket]
		}
		if antiAffinity {
			bucketPodAff -= count
		}
	}

	return bucketPodAff
}

func (jm *JobManager) buildBucket(taskWithOrder []*apis.TaskInfo) {
	nodeBucketMapping := make(map[string]*Bucket)

	for _, task := range taskWithOrder {
		klog.V(5).Infof("jobID %s task with order task %s/%s",
			jm.jobID, task.Namespace, task.Name)

		var selectedBucket *Bucket
		maxAffinity := math.MinInt32

		taskName := getTaskName(task)

		if task.NodeName != "" {
			// generate bucket by node
			maxAffinity = 0
			selectedBucket = nodeBucketMapping[task.NodeName]
		} else {
			for _, bucket := range jm.buckets {
				bucketPodAff := jm.checkTaskSetAffinity(taskName, bucket.taskNameSet, false)

				// choose the best fit affinity, or balance resource between bucket
				if bucketPodAff > maxAffinity {
					maxAffinity = bucketPodAff
					selectedBucket = bucket
				} else if bucketPodAff == maxAffinity && selectedBucket != nil &&
					bucket.reqScore < selectedBucket.reqScore {
					selectedBucket = bucket
				}
			}
		}

		if maxAffinity < 0 || selectedBucket == nil {
			selectedBucket = jm.NewBucket()
			if task.NodeName != "" {
				nodeBucketMapping[task.NodeName] = selectedBucket
			}
		}

		jm.AddTaskToBucket(selectedBucket.index, taskName, task)
	}
}

// AddTaskToBucket adds task into the index-th bucket of jm.
func (jm *JobManager) AddTaskToBucket(bucketIndex int, taskName string, task *apis.TaskInfo) {
	bucket := jm.buckets[bucketIndex]
	jm.podInBucket[task.Pod.UID] = bucketIndex
	bucket.AddTask(taskName, task)

	// update jm.bucketMaxSize
	if size := len(bucket.tasks) + bucket.boundTask; size > jm.bucketMaxSize {
		jm.bucketMaxSize = size
	}
}

// TaskBound binds task to bucket.
func (jm *JobManager) TaskBound(task *apis.TaskInfo) {
	if taskName := getTaskName(task); taskName != "" {
		set, ok := jm.nodeTaskSet[task.NodeName]
		if !ok {
			set = map[string]int{}
			jm.nodeTaskSet[task.NodeName] = set
		}
		set[taskName]++
	}
	bucket := jm.GetBucket(task)
	if bucket != nil {
		bucket.TaskBound(task)
	}
}

// MarkOutOfBucket indicates task is out of any bucket.
func (jm *JobManager) MarkOutOfBucket(uid types.UID) {
	jm.podInBucket[uid] = OutOfBucket
}

// MarkTaskHasTopology indicates task has topology settings
func (jm *JobManager) MarkTaskHasTopology(taskName string, topoType topologyType) {
	priority := affinityPriority[topoType]
	if priority > jm.taskAffinityPriority[taskName] {
		jm.taskAffinityPriority[taskName] = priority
	}
}

// ApplyTaskTopology transforms taskTopology to matrix
// affinity: [[a, b], [c]]
// interAffinity:
//      a   b   c
//  a   -   x   -
//  b   x   -   -
//  c   -   -   -
//  selfAffinity:
//      a   b   c
//      -   -   x
func (jm *JobManager) ApplyTaskTopology(topo *TaskTopology) {
	for _, aff := range topo.Affinity {
		if len(aff) == 1 {
			taskName := aff[0]
			jm.selfAffinity[taskName] = struct{}{}
			jm.MarkTaskHasTopology(taskName, selfAffinity)
			continue
		}
		for index, src := range aff {
			for _, dst := range aff[:index] {
				addAffinity(jm.interAffinity, src, dst)
				addAffinity(jm.interAffinity, dst, src)
			}
			jm.MarkTaskHasTopology(src, interAffinity)
		}
	}

	for _, aff := range topo.AntiAffinity {
		if len(aff) == 1 {
			taskName := aff[0]
			jm.selfAntiAffinity[taskName] = struct{}{}
			jm.MarkTaskHasTopology(taskName, selfAntiAffinity)
			continue
		}
		for index, src := range aff {
			for _, dst := range aff[:index] {
				addAffinity(jm.interAntiAffinity, src, dst)
				addAffinity(jm.interAntiAffinity, dst, src)
			}
			jm.MarkTaskHasTopology(src, interAntiAffinity)
		}
	}

	length := len(topo.TaskOrder)
	for index, taskName := range topo.TaskOrder {
		jm.taskExistOrder[taskName] = length - index
	}
}

func (jm *JobManager) taskAffinityOrder(L, R *apis.TaskInfo) int {
	LTaskName := jm.podInTask[L.Pod.UID]
	RTaskName := jm.podInTask[R.Pod.UID]

	if LTaskName == RTaskName {
		return 0
	}

	// use user defined order firstly
	LOrder := jm.taskExistOrder[LTaskName]
	ROrder := jm.taskExistOrder[RTaskName]
	if LOrder != ROrder {
		if LOrder > ROrder {
			return 1
		}
		return -1
	}

	LPriority := jm.taskAffinityPriority[LTaskName]
	RPriority := jm.taskAffinityPriority[RTaskName]
	if LPriority != RPriority {
		if LPriority > RPriority {
			return 1
		}
		return -1
	}

	// all affinity setting of L and R are the same, they are equal
	return 0
}
