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

package job

// fully checked and understood

import (
	"fmt"
	busv1alpha1 "github.com/hliangzhao/volcano/pkg/apis/bus/v1alpha1"
	"github.com/hliangzhao/volcano/pkg/cli/utils"
	"github.com/spf13/cobra"
)

type suspendFlags struct {
	commonFlags

	Namespace string
	JobName   string
}

var suspendJobFlags = &suspendFlags{}

// InitSuspendFlags init suspend related flags.
func InitSuspendFlags(cmd *cobra.Command) {
	initFlags(cmd, &suspendJobFlags.commonFlags)

	cmd.Flags().StringVarP(&suspendJobFlags.Namespace, "namespace", "n", "default", "the namespace of job")
	cmd.Flags().StringVarP(&suspendJobFlags.JobName, "name", "N", "", "the name of job")
}

// SuspendJob suspends the job.
// Other than `run`, `list`, and `view`, `suspend` cannot be implemented with simple CRUD through the clientset.
// It is implemented by the `Command` CRD.
func SuspendJob() error {
	config, err := utils.BuildConfig(suspendJobFlags.Master, suspendJobFlags.Kubeconfig)
	if err != nil {
		return err
	}

	if suspendJobFlags.JobName == "" {
		err := fmt.Errorf("job name is mandatory to suspend a particular job")
		return err
	}

	return utils.CreateJobCommand(config, suspendJobFlags.Namespace, suspendJobFlags.JobName, busv1alpha1.AbortJobAction)
}
