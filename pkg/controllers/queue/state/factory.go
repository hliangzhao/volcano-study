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

package state

// fully checked and understood

import (
	busv1alpha1 "github.com/hliangzhao/volcano/pkg/apis/bus/v1alpha1"
	schedulingv1alpha1 "github.com/hliangzhao/volcano/pkg/apis/scheduling/v1alpha1"
)

type State interface {
	// Execute uses an internal status variable to update the status of the queue we care about
	Execute(action busv1alpha1.Action) error
}

// UpdateQueueStatusFn is a function that updates the variable `status`.
type UpdateQueueStatusFn func(status *schedulingv1alpha1.QueueStatus, pgList []string)

// QueueActionFn is a function that updates `queue` by executing the UpdateQueueStatusFn fn.
type QueueActionFn func(queue *schedulingv1alpha1.Queue, fn UpdateQueueStatusFn) error

var (
	// SyncQueue is a function that sync queue
	SyncQueue QueueActionFn
	// OpenQueue is a function that open queue
	OpenQueue QueueActionFn
	// CloseQueue is a function that close queue
	CloseQueue QueueActionFn
)

// NewState transforms the input queue into new state.
func NewState(queue *schedulingv1alpha1.Queue) State {
	switch queue.Status.State {
	case "", schedulingv1alpha1.QueueStateOpen:
		return &openState{queue: queue}
	case schedulingv1alpha1.QueueStateClosed:
		return &closedState{queue: queue}
	case schedulingv1alpha1.QueueStateClosing:
		return &closingState{queue: queue}
	case schedulingv1alpha1.QueueStateUnknown:
		return &unknownState{queue: queue}
	}
	return nil
}
