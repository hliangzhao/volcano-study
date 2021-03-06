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

package main

// fully checked and understood

import (
	"fmt"
	"github.com/hliangzhao/volcano/cmd/cli/utils"
	"github.com/hliangzhao/volcano/pkg/cli/vresume"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	"os"
	"time"
)

var logFlushFreq = pflag.Duration("log-flush-frequency", 5*time.Second, "Maximum number of seconds between log flushes")

// `vresume` is same to `vcctl job resume`, which is abstracted as an independent command

func main() {
	klog.InitFlags(nil)

	// The default klog flush interval is 30 seconds, which is frighteningly long.
	go wait.Until(klog.Flush, *logFlushFreq, wait.NeverStop)
	defer klog.Flush()

	rootCmd := cobra.Command{
		Use:   "vresume",
		Short: "resume a job",
		Long:  `resume an aborted job with specified name in default or specified namespace`,
		Run: func(cmd *cobra.Command, args []string) {
			utils.CheckError(cmd, vresume.ResumeJob())
		},
	}

	jobResumeCmd := &rootCmd
	vresume.InitResumeFlags(jobResumeCmd)
	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("Failed to execute vresume: %v\n", err)
		os.Exit(-2)
	}
}
