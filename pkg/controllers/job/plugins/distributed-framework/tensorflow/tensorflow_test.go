/*
Copyright 2021-2022 hliangzhao.

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

package tensorflow

// TODO: just copied. Not checked.
// Passed.

import (
	batchv1alpha1 "github.com/hliangzhao/volcano/pkg/apis/batch/v1alpha1"
	plugininterface "github.com/hliangzhao/volcano/pkg/controllers/job/plugins/interface"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
	"testing"
)

func TestTensorflow(t *testing.T) {
	testjob := &batchv1alpha1.Job{
		ObjectMeta: metav1.ObjectMeta{Name: "train-123"},
		Spec: batchv1alpha1.JobSpec{
			Tasks: []batchv1alpha1.TaskSpec{
				{
					Name:     "ps",
					Replicas: 2,
					Template: v1.PodTemplateSpec{},
				},
				{
					Name:     "worker",
					Replicas: 2,
					Template: v1.PodTemplateSpec{},
				},
				{
					Name:     "chief",
					Replicas: 1,
					Template: v1.PodTemplateSpec{},
				},
			},
		},
	}
	testcases := []struct {
		Name string
		Job  *batchv1alpha1.Job
		Pod  *v1.Pod
	}{
		{
			Name: "ps case",
			Job:  testjob,
			Pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "train-123-ps-0",
					Annotations: map[string]string{
						batchv1alpha1.TaskSpecKey: "ps",
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name: "main",
						},
					},
				},
			},
		},
		{
			Name: "worker case",
			Job:  testjob,
			Pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "train-123-worker-0",
					Annotations: map[string]string{
						batchv1alpha1.TaskSpecKey: "worker",
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name: "main",
						},
					},
				},
			},
		},
		{
			Name: "chief case",
			Job:  testjob,
			Pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "train-123-chief-0",
					Annotations: map[string]string{
						batchv1alpha1.TaskSpecKey: "chief",
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name: "main",
						},
					},
				},
			},
		},
	}

	for i, testcase := range testcases {
		t.Run(testcase.Name, func(t *testing.T) {
			tp := New(plugininterface.PluginClient{}, []string{"--port=5000"})
			if err := tp.OnPodCreate(testcase.Pod, testcase.Job); err != nil {
				t.Errorf("Case %d (%s): expect no error, but got error %v", i, testcase.Name, err)
			}
			if testcase.Pod.Spec.Containers[0].Env[0].Name != "TF_CONFIG" {
				t.Errorf("Case %d (%s): wrong env name, got %s", i, testcase.Name, testcase.Pod.Spec.Containers[0].Env[0].Name)
			}

			switch {
			case strings.Contains(testcase.Pod.Name, "ps"):
				if testcase.Pod.Spec.Containers[0].Env[0].Value != `{"cluster":{"ps":["train-123-ps-0.train-123:5000","train-123-ps-1.train-123:5000"],"worker":["train-123-worker-0.train-123:5000","train-123-worker-1.train-123:5000"],"chief":["train-123-chief-0.train-123:5000"]},"task":{"type":"ps","index":0}}` {
					t.Errorf("Case %d (%s): wrong env value, got %s", i, testcase.Name, testcase.Pod.Spec.Containers[0].Env[0].Value)
				}
			case strings.Contains(testcase.Pod.Name, "worker"):
				if testcase.Pod.Spec.Containers[0].Env[0].Value != `{"cluster":{"ps":["train-123-ps-0.train-123:5000","train-123-ps-1.train-123:5000"],"worker":["train-123-worker-0.train-123:5000","train-123-worker-1.train-123:5000"],"chief":["train-123-chief-0.train-123:5000"]},"task":{"type":"worker","index":0}}` {
					t.Errorf("Case %d (%s): wrong env value, got %s", i, testcase.Name, testcase.Pod.Spec.Containers[0].Env[0].Value)
				}
			case strings.Contains(testcase.Pod.Name, "chief"):
				if testcase.Pod.Spec.Containers[0].Env[0].Value != `{"cluster":{"ps":["train-123-ps-0.train-123:5000","train-123-ps-1.train-123:5000"],"worker":["train-123-worker-0.train-123:5000","train-123-worker-1.train-123:5000"],"chief":["train-123-chief-0.train-123:5000"]},"task":{"type":"chief","index":0}}` {
					t.Errorf("Case %d (%s): wrong env value, got %s", i, testcase.Name, testcase.Pod.Spec.Containers[0].Env[0].Value)
				}
			}
		})
	}
}
