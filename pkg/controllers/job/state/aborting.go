/*
Copyright 2021 hliangzhao.

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

type abortingState struct {
	job *apis.JobInfo
}

func (state *abortingState) Execute(act busv1alpha1.Action) error {
	switch act {
	case busv1alpha1.ResumeJobAction:
		return KillJob(state.job, PodRetainPhaseSoft, func(status *batchv1alpha1.JobStatus) bool {
			status.State.Phase = batchv1alpha1.Restarting
			status.RetryCount++
			return true
		})
	default:
		return KillJob(state.job, PodRetainPhaseSoft, func(status *batchv1alpha1.JobStatus) bool {
			if status.Terminating != 0 || status.Pending != 0 || status.Running != 0 {
				return false
			}
			status.State.Phase = batchv1alpha1.Aborted
			return true
		})
	}
}
