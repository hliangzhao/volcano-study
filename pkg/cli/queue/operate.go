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

package queue

// TODO: question exists

import (
	"context"
	"fmt"
	busv1alpha1 "github.com/hliangzhao/volcano/pkg/apis/bus/v1alpha1"
	`github.com/hliangzhao/volcano/pkg/cli/utils`
	volcanoclient "github.com/hliangzhao/volcano/pkg/client/clientset/versioned"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	// ActionOpen is `open` action
	ActionOpen = "open"
	// ActionClose is `close` action
	ActionClose = "close"
	// ActionUpdate is `update` action
	ActionUpdate = "update"
)

type operateFlags struct {
	commonFlags

	// Name is name of queue
	Name string
	// Weight is weight of queue
	Weight int32
	// Action is operation action of queue
	Action string
}

var operateQueueFlags = &operateFlags{}

// InitOperateFlags is used to init all flags during queue operating
func InitOperateFlags(cmd *cobra.Command) {
	initFlags(cmd, &operateQueueFlags.commonFlags)

	cmd.Flags().StringVarP(&operateQueueFlags.Name, "name", "n", "", "the name of queue")
	cmd.Flags().Int32VarP(&operateQueueFlags.Weight, "weight", "w", 0, "the weight of the queue")
	cmd.Flags().StringVarP(&operateQueueFlags.Action, "action", "a", "",
		"operate action to queue, valid actions are open, close, update")
}

// OperateQueue operates queue.
// TODO: why not need to specify the namespace of the queue resource?
func OperateQueue() error {
	config, err := utils.BuildConfig(operateQueueFlags.Master, operateQueueFlags.Kubeconfig)
	if err != nil {
		return err
	}

	if len(operateQueueFlags.Name) == 0 {
		return fmt.Errorf("queue name must be specified")
	}

	var action busv1alpha1.Action
	switch operateQueueFlags.Action {
	case ActionOpen:
		action = busv1alpha1.OpenQueueAction
	case ActionClose:
		action = busv1alpha1.CloseQueueAction
	case ActionUpdate:
		if operateQueueFlags.Weight == 0 {
			return fmt.Errorf("when %s queue %s, weight must be specified, "+
				"the value must be greater than 0", ActionUpdate, operateQueueFlags.Name)
		}
		queueClient := volcanoclient.NewForConfigOrDie(config)
		patchBytes := []byte(fmt.Sprintf(`{"spec":{"weight":%d}}`, operateQueueFlags.Weight))
		// use Patch to update the specific segments of the queue resource
		_, err := queueClient.SchedulingV1alpha1().Queues().Patch(context.TODO(),
			operateQueueFlags.Name, types.MergePatchType, patchBytes, metav1.PatchOptions{})
		return err
	case "":
		return fmt.Errorf("action can not be null")
	default:
		return fmt.Errorf("action %s invalid, valid actions are %s, %s and %s",
			operateQueueFlags.Action, ActionOpen, ActionClose, ActionUpdate)
	}

	// open/close queue are implemented through the `Command` CRD
	// Correspondingly, update queue can be simply implemented with the clientset
	return createQueueCommand(config, action)
}
