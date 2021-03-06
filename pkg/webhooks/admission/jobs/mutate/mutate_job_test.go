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

package mutate

// TODO: just copied.
//  Passed.

import (
	batchv1alpha1 "github.com/hliangzhao/volcano/pkg/apis/batch/v1alpha1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestCreatePatchExecution(t *testing.T) {
	namespace := "test"
	testCase := struct {
		Name      string
		Job       batchv1alpha1.Job
		operation patchOperation
	}{
		Name: "patch default task name",
		Job: batchv1alpha1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "path-task-name",
				Namespace: namespace,
			},
			Spec: batchv1alpha1.JobSpec{
				MinAvailable: 1,
				Tasks: []batchv1alpha1.TaskSpec{
					{
						Replicas: 1,
						Template: v1.PodTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{
								Labels: map[string]string{"name": "test"},
							},
							Spec: v1.PodSpec{
								Containers: []v1.Container{
									{
										Name:  "fake-name",
										Image: "busybox:1.24",
									},
								},
							},
						},
					},
					{
						Replicas: 1,
						Template: v1.PodTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{
								Labels: map[string]string{"name": "test"},
							},
							Spec: v1.PodSpec{
								Containers: []v1.Container{
									{
										Name:  "fake-name",
										Image: "busybox:1.24",
									},
								},
							},
						},
					},
				},
			},
		},
		operation: patchOperation{
			Op:   "replace",
			Path: "/spec/tasks",
			Value: []batchv1alpha1.TaskSpec{
				{
					Name:     batchv1alpha1.DefaultTaskSpec + "0",
					Replicas: 1,
					Template: v1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{"name": "test"},
						},
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								{
									Name:  "fake-name",
									Image: "busybox:1.24",
								},
							},
						},
					},
				},
				{
					Name:     batchv1alpha1.DefaultTaskSpec + "1",
					Replicas: 1,
					Template: v1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{"name": "test"},
						},
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								{
									Name:  "fake-name",
									Image: "busybox:1.24",
								},
							},
						},
					},
				},
			},
		},
	}

	ret := mutateSpec(testCase.Job.Spec.Tasks, "/spec/tasks")
	if ret.Path != testCase.operation.Path || ret.Op != testCase.operation.Op {
		t.Errorf("testCase %s's expected patch operation %v, but got %v",
			testCase.Name, testCase.operation, *ret)
	}

	actualTasks, ok := ret.Value.([]batchv1alpha1.TaskSpec)
	if !ok {
		t.Errorf("testCase '%s' path value expected to be '[]batchv1alpha1.TaskSpec', but negative",
			testCase.Name)
	}
	expectedTasks, _ := testCase.operation.Value.([]batchv1alpha1.TaskSpec)
	for index, task := range expectedTasks {
		aTask := actualTasks[index]
		if aTask.Name != task.Name {
			t.Errorf("testCase '%s's expected patch operation with value %v, but got %v",
				testCase.Name, testCase.operation.Value, ret.Value)
		}
		if aTask.MaxRetry != defaultMaxRetry {
			t.Errorf("testCase '%s's expected patch 'task.MaxRetry' with value %v, but got %v",
				testCase.Name, defaultMaxRetry, aTask.MaxRetry)
		}
	}

}
