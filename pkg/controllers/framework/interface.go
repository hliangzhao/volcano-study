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

package framework

import (
	volcano "github.com/hliangzhao/volcano/pkg/client/clientset/versioned"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
)

type ControllerOption struct {
	KubeClient            kubernetes.Interface
	VolcanoClient         volcano.Interface
	SharedInformerFactory informers.SharedInformerFactory
	SchedulerNames        []string
	WorkerNum             uint32
	MaxRequeueNum         int
}

type Controller interface {
	Name() string
	Initialize(opt *ControllerOption) error
	Run(stopCh <-chan struct{})
}