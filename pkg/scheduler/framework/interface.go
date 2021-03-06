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

// This file defines two interfaces, Action and Plugin.
// A legal Action should have the `Initialize`, `Execute`, and `UnInitialize` functions.
// A legal Plugin should have the `OnSessionOpen` and `OnSessionClose` functions.

// Action is the action interface. Typical actions including enqueue, allocate, backfill, etc.
type Action interface {
	Name() string
	Initialize()           // install plugins
	Execute(sess *Session) // execution of the action in the input session
	UnInitialize()         // uninstall plugins
}

// Plugin is the plugin interface. Custom scheduling algorithms are implemented as plugins.
type Plugin interface {
	Name() string
	OnSessionOpen(sess *Session)
	OnSessionClose(sess *Session)
}
