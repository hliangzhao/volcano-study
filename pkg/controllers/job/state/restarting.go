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

import (
	batchv1alpha1 "github.com/hliangzhao/volcano/pkg/apis/batch/v1alpha1"
	busv1alpha1 "github.com/hliangzhao/volcano/pkg/apis/bus/v1alpha1"
	"github.com/hliangzhao/volcano/pkg/controllers/apis"
)

type restartingState struct {
	job *apis.JobInfo
}

func (state *restartingState) Execute(act busv1alpha1.Action) error {
	return KillJob(state.job, PodRetainPhaseSoft, func(status *batchv1alpha1.JobStatus) bool {
		maxRetry := state.job.Job.Spec.MaxRetry
		if status.RetryCount >= maxRetry {
			status.State.Phase = batchv1alpha1.Failed
			return true
		}
		total := int32(0)
		for _, task := range state.job.Job.Spec.Tasks {
			total += task.Replicas
		}
		if total-status.Terminating >= status.MinAvailable {
			status.State.Phase = batchv1alpha1.Pending
			return true
		}
		return false
	})
}
