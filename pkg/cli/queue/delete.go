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

// fully checked and understood

import (
	"context"
	"fmt"
	`github.com/hliangzhao/volcano/pkg/cli/utils`
	volcanoclient "github.com/hliangzhao/volcano/pkg/client/clientset/versioned"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type deleteFlags struct {
	commonFlags
	// Name is name of queue
	Name string
}

var deleteQueueFlags = &deleteFlags{}

// InitDeleteFlags is used to init all flags during queue deleting.
func InitDeleteFlags(cmd *cobra.Command) {
	initFlags(cmd, &deleteQueueFlags.commonFlags)

	cmd.Flags().StringVarP(&deleteQueueFlags.Name, "name", "n", "", "the name of queue")
}

// DeleteQueue delete queue.
func DeleteQueue() error {
	config, err := utils.BuildConfig(deleteQueueFlags.Master, deleteQueueFlags.Kubeconfig)
	if err != nil {
		return err
	}

	if len(deleteQueueFlags.Name) == 0 {
		return fmt.Errorf("queue name must be specified")
	}

	// delete the queue resource from the cluster by calling the clientset directly
	queueClient := volcanoclient.NewForConfigOrDie(config)
	return queueClient.SchedulingV1alpha1().Queues().Delete(context.TODO(), deleteQueueFlags.Name, metav1.DeleteOptions{})
}
