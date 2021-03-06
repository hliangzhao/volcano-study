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
	"github.com/hliangzhao/volcano/pkg/cli/queue"
	"github.com/spf13/cobra"
)

func buildQueueCmd() *cobra.Command {
	queueCmd := &cobra.Command{
		Use:   "queue",
		Short: "Queue Operations",
	}

	// vcctl queue create
	queueCreateCmd := &cobra.Command{
		Use:   "create",
		Short: "creates queue",
		Run: func(cmd *cobra.Command, args []string) {
			checkError(cmd, queue.CreateQueue())
		},
	}
	queue.InitCreateFlags(queueCreateCmd)
	queueCmd.AddCommand(queueCreateCmd)

	// vcctl queue delete
	queueDeleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "delete queue",
		Run: func(cmd *cobra.Command, args []string) {
			checkError(cmd, queue.DeleteQueue())
		},
	}
	queue.InitDeleteFlags(queueDeleteCmd)
	queueCmd.AddCommand(queueDeleteCmd)

	// vcctl queue operate
	queueOperateCmd := &cobra.Command{
		Use:   "operate queue",
		Short: "operate queue",
		Run: func(cmd *cobra.Command, args []string) {
			checkError(cmd, queue.OperateQueue())
		},
	}
	queue.InitOperateFlags(queueOperateCmd)
	queueCmd.AddCommand(queueOperateCmd)

	// vcctl queue list
	queueListCmd := &cobra.Command{
		Use:   "list",
		Short: "lists all the queue",
		Run: func(cmd *cobra.Command, args []string) {
			checkError(cmd, queue.ListQueue())
		},
	}
	queue.InitListFlags(queueListCmd)
	queueCmd.AddCommand(queueListCmd)

	// vcctl queue get
	queueGetCmd := &cobra.Command{
		Use:   "get",
		Short: "get a queue",
		Run: func(cmd *cobra.Command, args []string) {
			checkError(cmd, queue.GetQueue())
		},
	}
	queue.InitGetFlags(queueGetCmd)
	queueCmd.AddCommand(queueGetCmd)

	return queueCmd
}
