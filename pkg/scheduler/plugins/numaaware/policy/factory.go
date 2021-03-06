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

package policy

import (
	batchv1alpha1 "github.com/hliangzhao/volcano/pkg/apis/batch/v1alpha1"
	nodeinfov1alpha1 "github.com/hliangzhao/volcano/pkg/apis/nodeinfo/v1alpha1"
	"github.com/hliangzhao/volcano/pkg/scheduler/apis"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpuset"
	"k8s.io/kubernetes/pkg/kubelet/cm/topologymanager/bitmask"
)

// TopologyHint is a struct containing the NUMANodeAffinity for a Container
type TopologyHint struct {
	NumaNodeAffinity bitmask.BitMask
	// Preferred is set to true when the NUMANodeAffinity encodes a preferred
	// allocation for the Container. It is set to false otherwise.
	Preferred bool
}

// Policy is an interface for topology manager policy
type Policy interface {
	// Predicate gets the best hit
	Predicate(providersHints []map[string][]TopologyHint) (TopologyHint, bool)
}

// HintProvider is an interface for components that want to collaborate to
// achieve globally optimal concrete resource alignment with respect to
// NUMA locality.
type HintProvider interface {
	// Name returns provider name used for register and logging.
	Name() string
	// GetTopologyHints returns hints if this hint provider has a preference,
	GetTopologyHints(container *corev1.Container, topoInfo *apis.NumaTopoInfo,
		resNumaSets apis.ResNumaSets) map[string][]TopologyHint

	Allocate(container *corev1.Container, bestHit *TopologyHint, topoInfo *apis.NumaTopoInfo,
		resNumaSets apis.ResNumaSets) map[string]cpuset.CPUSet
}

// GetPolicy return the interface matched the input task topology config
func GetPolicy(node *apis.NodeInfo, numaNodes []int) Policy {
	switch batchv1alpha1.NumaPolicy(node.NumaSchedulerInfo.Policies[nodeinfov1alpha1.TopologyManagerPolicy]) {
	case batchv1alpha1.None:
		return NewPolicyNone(numaNodes)
	case batchv1alpha1.BestEffort:
		return NewPolicyBestEffort(numaNodes)
	case batchv1alpha1.Restricted:
		return NewPolicyRestricted(numaNodes)
	case batchv1alpha1.SingleNumaNode:
		return NewPolicySingleNumaNode(numaNodes)
	}
	return &policyNone{}
}

// AccumulateProvidersHints return all TopologyHint collection from different providers
func AccumulateProvidersHints(container *corev1.Container, topoInfo *apis.NumaTopoInfo,
	resNumaSets apis.ResNumaSets, hintProviders []HintProvider) (providersHints []map[string][]TopologyHint) {

	for _, provider := range hintProviders {
		hints := provider.GetTopologyHints(container, topoInfo, resNumaSets)
		providersHints = append(providersHints, hints)
	}

	return providersHints
}

// Allocate return all resource assignment collection from different providers
func Allocate(container *corev1.Container, bestHit *TopologyHint,
	topoInfo *apis.NumaTopoInfo, resNumaSets apis.ResNumaSets, hintProviders []HintProvider) map[string]cpuset.CPUSet {

	allResAlloc := make(map[string]cpuset.CPUSet)
	for _, provider := range hintProviders {
		resAlloc := provider.Allocate(container, bestHit, topoInfo, resNumaSets)
		for resName, assign := range resAlloc {
			allResAlloc[resName] = assign
		}
	}

	return allResAlloc
}
