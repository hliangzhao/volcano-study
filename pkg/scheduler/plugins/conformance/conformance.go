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

package conformance

// fully checked and understood

import (
	"github.com/hliangzhao/volcano/pkg/scheduler/apis"
	"github.com/hliangzhao/volcano/pkg/scheduler/framework"
	"github.com/hliangzhao/volcano/pkg/scheduler/plugins/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/apis/scheduling"
)

// PluginName indicates name of volcano scheduler plugin.
const PluginName = "conformance"

type conformancePlugin struct {
	pluginArguments framework.Arguments
}

func New(arguments framework.Arguments) framework.Plugin {
	return &conformancePlugin{pluginArguments: arguments}
}

func (cp *conformancePlugin) Name() string {
	return PluginName
}

// OnSessionOpen of conformancePlugin adds a evictableFn to sess.
// Specifically, evictableFn gets the victim tasks from the task list `evictees` by getting rid of the critical pods.
func (cp *conformancePlugin) OnSessionOpen(sess *framework.Session) {
	// evictableFn gets the victim tasks from the task list `evictees`
	evictableFn := func(evictor *apis.TaskInfo, evictees []*apis.TaskInfo) ([]*apis.TaskInfo, int) {
		var victims []*apis.TaskInfo
		for _, evictee := range evictees {
			className := evictee.Pod.Spec.PriorityClassName
			// Skip critical pod.
			if className == scheduling.SystemClusterCritical ||
				className == scheduling.SystemNodeCritical ||
				evictee.Namespace == metav1.NamespaceSystem {
				continue
			}
			victims = append(victims, evictee)
		}
		return victims, utils.Permit
	}
	sess.AddPreemptableFn(cp.Name(), evictableFn)
	sess.AddReclaimableFn(cp.Name(), evictableFn)
}

func (cp *conformancePlugin) OnSessionClose(sess *framework.Session) {}
