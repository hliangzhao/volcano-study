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
	"github.com/hliangzhao/volcano/cmd/webhook-manager/app"
	"github.com/hliangzhao/volcano/cmd/webhook-manager/app/options"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/component-base/cli/flag"
	"k8s.io/klog/v2"
	"os"
	"runtime"
	"time"
)

var logFlushFreq = pflag.Duration("log-flush-frequency", 5*time.Second, "Maximum number of seconds between log flushes")

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	klog.InitFlags(nil)

	config := options.NewConfig()
	config.AddFlags(pflag.CommandLine)

	flag.InitFlags()

	go wait.Until(klog.Flush, *logFlushFreq, wait.NeverStop)
	defer klog.Flush()

	// check webhook-controller-server's port
	if err := config.CheckPortOrDie(); err != nil {
		klog.Fatalf("Configured port is invalid: %v", err)
	}

	// parse CA files for https communication
	if err := config.ParseCAFiles(nil); err != nil {
		klog.Fatalf("Failed to parse CA file: %v", err)
	}

	// start the webhook-controller-server
	if err := app.Run(config); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
