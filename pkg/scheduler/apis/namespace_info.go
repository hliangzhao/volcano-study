/*
Copyright 2021-2022 hliangzhao.

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

import (
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

const (
	// NamespaceWeightKey is the key in ResourceQuota.spec.hard indicating the weight of this namespace
	NamespaceWeightKey = "hliangzhao.io/namespace.weight"

	// DefaultNamespaceWeight is the default weight of namespace
	DefaultNamespaceWeight = 1
)

type NamespaceName string

type NamespaceInfo struct {
	// Name is the name of this namespace
	Name NamespaceName

	// Weight is the highest weight among many ResourceQuota
	Weight int64

	// QuotaStatus stores the ResourceQuotaStatus of all ResourceQuotas in this namespace
	QuotaStatus map[string]corev1.ResourceQuotaStatus
}

func (ni *NamespaceInfo) GetWeight() int64 {
	if ni == nil || ni.Weight == 0 {
		return DefaultNamespaceWeight
	}
	return ni.Weight
}

type quotaItem struct {
	name   string
	weight int64
}

// quotaItemKeyFunc parses obj as quotaItem and returns the quotaItem's name.
func quotaItemKeyFunc(obj interface{}) (string, error) {
	item, ok := obj.(*quotaItem)
	if !ok {
		return "", fmt.Errorf("obj with type %T could not parse", obj)
	}
	return item.name, nil
}

// quotaItemLessFunc parses a and b as quotaItem and judges whether a.weight > b.weight.
func quotaItemLessFunc(a interface{}, b interface{}) bool {
	// for big root heap
	A := a.(*quotaItem)
	B := b.(*quotaItem)
	return A.weight > B.weight
}

// NamespaceCollection is used to collect quotaItems.
// quotaItems are saved into a heap. Thus, quotaItemKeyFunc and quotaItemLessFunc are required for creating the heap.
type NamespaceCollection struct {
	Name        string
	quotaWeight *cache.Heap
	QuotaStatus map[string]corev1.ResourceQuotaStatus
}

func NewNamespaceCollection(name string) *NamespaceCollection {
	n := &NamespaceCollection{
		Name:        name,
		quotaWeight: cache.NewHeap(quotaItemKeyFunc, quotaItemLessFunc),
		QuotaStatus: map[string]corev1.ResourceQuotaStatus{},
	}
	// add at least one item into quotaWeight.
	// Because cache.Heap.Pop would be blocked until queue is not empty
	n.updateWeight(&quotaItem{
		name:   NamespaceWeightKey,
		weight: DefaultNamespaceWeight,
	})
	return n
}

/* delete and update func of NamespaceCollection */

func (nc *NamespaceCollection) deleteWeight(q *quotaItem) {
	_ = nc.quotaWeight.Delete(q)
}

func (nc *NamespaceCollection) updateWeight(q *quotaItem) {
	_ = nc.quotaWeight.Update(q)
}

// itemFromQuota creates a quotaItem instance from corev1.ResourceQuota.
func itemFromQuota(quota *corev1.ResourceQuota) *quotaItem {
	var weight int64 = DefaultNamespaceWeight
	quotaWeight, ok := quota.Spec.Hard[NamespaceWeightKey]
	if ok {
		weight = quotaWeight.Value()
	}
	return &quotaItem{
		name:   quota.Name,
		weight: weight,
	}
}

func (nc *NamespaceCollection) Update(quota *corev1.ResourceQuota) {
	nc.updateWeight(itemFromQuota(quota))
	nc.QuotaStatus[quota.Name] = quota.Status
}

func (nc *NamespaceCollection) Delete(quota *corev1.ResourceQuota) {
	nc.deleteWeight(itemFromQuota(quota))
	delete(nc.QuotaStatus, quota.Name)
}

// Snapshot clones a NamespaceInfo without Heap according NamespaceCollection.
func (nc *NamespaceCollection) Snapshot() *NamespaceInfo {
	var weight int64 = DefaultNamespaceWeight

	// get the weight of the first obj in the heap, and then put it back
	obj, err := nc.quotaWeight.Pop()
	if err != nil {
		klog.Warningf("namespace %s, quota weight meets error %v when pop", nc.Name, err)
	} else {
		item := obj.(*quotaItem)
		weight = item.weight
		_ = nc.quotaWeight.Add(item)
	}

	// the weight we get is used to create the snapshot
	return &NamespaceInfo{
		Name:        NamespaceName(nc.Name),
		Weight:      weight,
		QuotaStatus: nc.QuotaStatus,
	}
}
