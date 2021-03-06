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

package framework

// fully checked and understood

import (
	"fmt"
	"k8s.io/klog/v2"
)

var controllers = map[string]Controller{}

// RegisterController add controller to the global variable controllers.
func RegisterController(controller Controller) error {
	if controller == nil {
		return fmt.Errorf("controller is nil")
	}
	if _, found := controllers[controller.Name()]; found {
		return fmt.Errorf("duplicated controller")
	}

	klog.V(3).Infof("Controller <%s> is registered.", controller.Name())
	controllers[controller.Name()] = controller
	return nil
}

// ForeachController executes fn for each controller.
func ForeachController(fn func(controller Controller)) {
	for _, controller := range controllers {
		fn(controller)
	}
}
