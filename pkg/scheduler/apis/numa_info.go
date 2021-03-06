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

package apis

// fully checked and understood

import (
	nodeinfov1alpha1 "github.com/hliangzhao/volcano/pkg/apis/nodeinfo/v1alpha1"
	"gopkg.in/square/go-jose.v2/json"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpumanager/topology"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpuset"
)

type NumaChangeFlag int

const (
	// NumaInfoResetFlag indicate reset operate
	NumaInfoResetFlag NumaChangeFlag = 0b00

	// NumaInfoMoreFlag indicate the received allocatable resource is getting more
	NumaInfoMoreFlag NumaChangeFlag = 0b11

	// NumaInfoLessFlag indicate the received allocatable resource is getting less
	NumaInfoLessFlag NumaChangeFlag = 0b10

	// DefaultMaxNodeScore indicates the default max node score
	DefaultMaxNodeScore = 100
)

// PodResourceDecision is the resource allocation determined by scheduler, and passed to kubelet through pod annotation.
type PodResourceDecision struct {
	// key is NUMA id
	NumaResources map[int]corev1.ResourceList `json:"numa,omitempty"`
}

type TopologyInfo struct {
	Policy string
	ResMap map[int]corev1.ResourceList // key: NUMA ID
}

func (info *TopologyInfo) Clone() *TopologyInfo {
	copyInfo := &TopologyInfo{
		Policy: info.Policy,
		ResMap: map[int]corev1.ResourceList{},
	}

	for numaId, resList := range info.ResMap {
		copyInfo.ResMap[numaId] = resList.DeepCopy()
	}

	return copyInfo
}

// ResourceInfo is the allocatable information for the resource.
type ResourceInfo struct {
	Allocatable        cpuset.CPUSet
	Capacity           int
	AllocatablePerNuma map[int]float64 // key: NUMA id
	UsedPerNuma        map[int]float64 // key: NUMA id
}

// NumaTopoInfo is the information about topology manager on the node.
type NumaTopoInfo struct {
	Namespace   string
	Name        string
	Policies    map[nodeinfov1alpha1.PolicyName]string
	NumaResMap  map[string]*ResourceInfo
	CPUDetail   topology.CPUDetails // CPUDetails is a map from CPU ID to Core ID, Socket ID, and NUMA ID
	ResReserved corev1.ResourceList
}

func (info *NumaTopoInfo) DeepCopy() *NumaTopoInfo {
	ret := &NumaTopoInfo{
		Namespace:   info.Namespace,
		Name:        info.Name,
		Policies:    map[nodeinfov1alpha1.PolicyName]string{},
		NumaResMap:  map[string]*ResourceInfo{},
		CPUDetail:   topology.CPUDetails{},
		ResReserved: corev1.ResourceList{},
	}

	// clone polices
	for name, policy := range info.Policies {
		ret.Policies[name] = policy
	}

	// clone numa resMap
	for resName, resInfo := range info.NumaResMap {
		tmpResInfo := &ResourceInfo{
			AllocatablePerNuma: map[int]float64{},
			UsedPerNuma:        map[int]float64{},
		}

		tmpResInfo.Allocatable = resInfo.Allocatable.Clone()
		tmpResInfo.Capacity = resInfo.Capacity
		for numaId, data := range resInfo.AllocatablePerNuma {
			tmpResInfo.AllocatablePerNuma[numaId] = data
		}
		for numaId, data := range resInfo.UsedPerNuma {
			tmpResInfo.UsedPerNuma[numaId] = data
		}
		ret.NumaResMap[resName] = tmpResInfo
	}

	// clone cpu detail
	for cpuId, cpuInfo := range info.CPUDetail {
		ret.CPUDetail[cpuId] = cpuInfo
	}

	// clone resource reserved
	for resName, res := range info.ResReserved {
		ret.ResReserved[resName] = res
	}

	return ret
}

// Compare is the function to show the change of the resource on kubelet.
// Return val:
// true: at least one resource on kubelet is getting more or no change;
// false: otherwise.
func (info *NumaTopoInfo) Compare(newInfo *NumaTopoInfo) bool {
	for resName := range info.NumaResMap {
		oldSize := info.NumaResMap[resName].Allocatable.Size()
		newSize := newInfo.NumaResMap[resName].Allocatable.Size()
		if oldSize <= newSize {
			return true
		}
	}
	return false
}

// ResNumaSets key is the resource name
type ResNumaSets map[string]cpuset.CPUSet

// Allocate is the function to remove the allocated resource from info.
func (info *NumaTopoInfo) Allocate(resSets ResNumaSets) {
	for resName := range resSets {
		info.NumaResMap[resName].Allocatable = info.NumaResMap[resName].Allocatable.Difference(resSets[resName])
	}
}

// Release is the function to release the allocated resource to info.
func (info *NumaTopoInfo) Release(resSets ResNumaSets) {
	for resName := range resSets {
		info.NumaResMap[resName].Allocatable = info.NumaResMap[resName].Allocatable.Union(resSets[resName])
	}
}

// GetPodResourceNumaInfo returns numa resources from (or the annotation of) the given task.
func GetPodResourceNumaInfo(ti *TaskInfo) map[int]corev1.ResourceList {
	// has already been filled into ti, return directly
	if ti.NumaInfo != nil && len(ti.NumaInfo.ResMap) > 0 {
		return ti.NumaInfo.ResMap
	}

	// parse from annotation
	if _, ok := ti.Pod.Annotations[TopologyDecisionAnnotation]; !ok {
		return nil
	}
	decision := PodResourceDecision{}
	if err := json.Unmarshal([]byte(ti.Pod.Annotations[TopologyDecisionAnnotation]), &decision); err != nil {
		return nil
	}
	return decision.NumaResources
}

// AddTask is the function to update the used resource of per numa node.
func (info *NumaTopoInfo) AddTask(ti *TaskInfo) {
	decision := GetPodResourceNumaInfo(ti)
	if decision == nil {
		return
	}

	// update numa topo info
	for numaId, resList := range decision {
		for resName, resQuantity := range resList {
			info.NumaResMap[string(resName)].UsedPerNuma[numaId] += ResQuantity2Float64(resName, resQuantity)
		}
	}
}

// RemoveTask is the function to update the used resource of per numa node.
func (info *NumaTopoInfo) RemoveTask(ti *TaskInfo) {
	decision := GetPodResourceNumaInfo(ti)
	if decision == nil {
		return
	}

	// update numa topo info
	for numaId, resList := range ti.NumaInfo.ResMap {
		for resName, resQuantity := range resList {
			info.NumaResMap[string(resName)].UsedPerNuma[numaId] -= ResQuantity2Float64(resName, resQuantity)
		}
	}
}

// GenerateNodeResNumaSets returns the idle resource sets of nodes.
func GenerateNodeResNumaSets(nodes map[string]*NodeInfo) map[string]ResNumaSets {
	nodeSlice := make(map[string]ResNumaSets)
	for _, node := range nodes {
		if node.NumaSchedulerInfo == nil {
			continue
		}
		resMaps := make(ResNumaSets)
		for resName, resMap := range node.NumaSchedulerInfo.NumaResMap {
			resMaps[resName] = resMap.Allocatable.Clone()
		}
		nodeSlice[node.Name] = resMaps
	}
	return nodeSlice
}

// GenerateNumaNodes return the numa IDs of nodes. Key is the node name, value is the numa ids in this node.
func GenerateNumaNodes(nodes map[string]*NodeInfo) map[string][]int {
	nodeNumaMap := make(map[string][]int)
	for _, node := range nodes {
		if node.NumaSchedulerInfo == nil {
			continue
		}
		nodeNumaMap[node.Name] = node.NumaSchedulerInfo.CPUDetail.NUMANodes().ToSlice()
	}
	return nodeNumaMap
}

// Allocate is to remove the allocated resource which is assigned to task from resSets.
func (resSets ResNumaSets) Allocate(taskSets ResNumaSets) {
	for resName := range taskSets {
		if _, ok := resSets[resName]; !ok {
			continue
		}
		resSets[resName] = resSets[resName].Difference(taskSets[resName])
	}
}

// Release is to reclaim the allocated resource which is assigned to task to resSets.
func (resSets ResNumaSets) Release(taskSets ResNumaSets) {
	for resName := range taskSets {
		if _, ok := resSets[resName]; !ok {
			continue
		}
		resSets[resName] = resSets[resName].Union(taskSets[resName])
	}
}

// Clone is the deep copy action.
func (resSets ResNumaSets) Clone() ResNumaSets {
	newSets := make(ResNumaSets)
	for resName := range resSets {
		newSets[resName] = resSets[resName].Clone()
	}
	return newSets
}

// ScoredNode is the wrapper for node during scoring
type ScoredNode struct {
	NodeName string
	Score    int64
}
