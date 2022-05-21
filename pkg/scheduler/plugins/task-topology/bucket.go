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
	"github.com/hliangzhao/volcano/pkg/scheduler/apis"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
)

type reqAction int

const (
	reqSub reqAction = iota
	reqAdd
)

// Bucket is struct used to classify tasks by affinity and anti-affinity.
type Bucket struct {
	index       int                          // the index of itself in the corresponding job manager
	tasks       map[types.UID]*apis.TaskInfo // where to put tasks (pods)
	taskNameSet map[string]int               // classify tasks by names

	// reqScore is score of resource
	// now, we regard 1 CPU and 1 GPU and 1Gi memory as the same score.
	reqScore float64
	request  *apis.Resource

	boundTask int
	node      map[string]int // node-by-name: node-score
}

func NewBucket() *Bucket {
	return &Bucket{
		index:       0,
		tasks:       map[types.UID]*apis.TaskInfo{},
		taskNameSet: map[string]int{},

		reqScore: 0,
		request:  apis.EmptyResource(),

		boundTask: 0,
		node:      map[string]int{},
	}
}

// CalcResReq calculates task resources request.
func (b *Bucket) CalcResReq(req *apis.Resource, action reqAction) {
	if req == nil {
		return
	}

	// calculate score from req
	cpu := req.MilliCPU
	// treat 1Mi the same as 1m cpu 1m gpu
	mem := req.Memory / 1024 / 1024
	score := cpu + mem
	for _, request := range req.ScalarResources {
		score += request
	}

	// update bucket by action
	switch action {
	case reqSub:
		b.reqScore -= score
		b.request.Sub(req)
	case reqAdd:
		b.reqScore += score
		b.request.Add(req)
	default:
		klog.V(3).Infof("Invalid action <%v> for resource <%v>",
			action, req)
	}
}

// AddTask adds task to bucket.tasks.
func (b *Bucket) AddTask(taskName string, task *apis.TaskInfo) {
	b.taskNameSet[taskName]++
	// if task is scheduled, add it to the corresponding node and set it as bounded
	if task.NodeName != "" {
		b.node[task.NodeName]++
		b.boundTask++
		return
	}

	b.tasks[task.Pod.UID] = task
	b.CalcResReq(task.ResReq, reqAdd)
}

// TaskBound bounds task to b. After bound to node, task should be removed from bucket.tasks.
func (b *Bucket) TaskBound(task *apis.TaskInfo) {
	b.node[task.NodeName]++
	b.boundTask++

	delete(b.tasks, task.Pod.UID)
	b.CalcResReq(task.ResReq, reqSub)
}
