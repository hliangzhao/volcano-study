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

package schema

import (
	"fmt"
	batchv1alpha1 "github.com/hliangzhao/volcano/pkg/apis/batch/v1alpha1"
	schedulingv1alpha1 "github.com/hliangzhao/volcano/pkg/apis/scheduling/v1alpha1"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/klog/v2"
)

func init() {

}

var scheme = runtime.NewScheme()

// Codecs is for retrieving serializers for the supported wire formats
// and conversion wrappers to define preferred internal and external versions.
var Codecs = serializer.NewCodecFactory(scheme)

func addToScheme(scheme *runtime.Scheme) {
	_ = corev1.AddToScheme(scheme)
	_ = admissionv1.AddToScheme(scheme)
}

// DecodeJob decodes the job using deserializer from the raw object.
func DecodeJob(obj runtime.RawExtension, resource metav1.GroupVersionResource) (*batchv1alpha1.Job, error) {
	jobResource := metav1.GroupVersionResource{
		Group:    batchv1alpha1.SchemeGroupVersion.Group,
		Version:  batchv1alpha1.SchemeGroupVersion.Version,
		Resource: "jobs",
	}
	raw := obj.Raw
	job := batchv1alpha1.Job{}

	if resource != jobResource {
		err := fmt.Errorf("expect resource to be %s", jobResource)
		return &job, err
	}

	deserializer := Codecs.UniversalDeserializer()
	if _, _, err := deserializer.Decode(raw, nil, &job); err != nil {
		return &job, err
	}
	klog.V(3).Infof("the job struct is %+v", job)
	return &job, nil
}

// DecodeQueue decodes the queue using deserializer from the raw object.
func DecodeQueue(object runtime.RawExtension, resource metav1.GroupVersionResource) (*schedulingv1alpha1.Queue, error) {
	queueResource := metav1.GroupVersionResource{
		Group:    schedulingv1alpha1.SchemeGroupVersion.Group,
		Version:  schedulingv1alpha1.SchemeGroupVersion.Version,
		Resource: "queues",
	}

	if resource != queueResource {
		return nil, fmt.Errorf("expect resource to be %s", queueResource)
	}

	queue := schedulingv1alpha1.Queue{}
	if _, _, err := Codecs.UniversalDeserializer().Decode(object.Raw, nil, &queue); err != nil {
		return nil, err
	}

	return &queue, nil
}

// DecodePodGroup decodes the podgroup using deserializer from the raw object.
func DecodePodGroup(object runtime.RawExtension, resource metav1.GroupVersionResource) (*schedulingv1alpha1.PodGroup, error) {
	podgroupResource := metav1.GroupVersionResource{
		Group:    schedulingv1alpha1.SchemeGroupVersion.Group,
		Version:  schedulingv1alpha1.SchemeGroupVersion.Version,
		Resource: "podgroups",
	}

	if resource != podgroupResource {
		return nil, fmt.Errorf("expect resource to be %s", podgroupResource)
	}

	podgroup := schedulingv1alpha1.PodGroup{}
	if _, _, err := Codecs.UniversalDeserializer().Decode(object.Raw, nil, &podgroup); err != nil {
		return nil, err
	}

	return &podgroup, nil
}

func DecodePod(object runtime.RawExtension, resource metav1.GroupVersionResource) (*corev1.Pod, error) {
	podResource := metav1.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
	raw := object.Raw
	pod := corev1.Pod{}

	if resource != podResource {
		err := fmt.Errorf("expect resource to be %s", podResource)
		return &pod, err
	}

	deserializer := Codecs.UniversalDeserializer()
	if _, _, err := deserializer.Decode(raw, nil, &pod); err != nil {
		return &pod, err
	}
	klog.V(3).Infof("the pod struct is %+v", pod)

	return &pod, nil
}
