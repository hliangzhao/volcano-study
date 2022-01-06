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

package podgroup

import (
	`context`
	`github.com/hliangzhao/volcano/pkg/apis/helpers`
	schedulingv1alpha1 `github.com/hliangzhao/volcano/pkg/apis/scheduling/v1alpha1`
	corev1 "k8s.io/api/core/v1"
	apierrors `k8s.io/apimachinery/pkg/api/errors`
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	`k8s.io/apimachinery/pkg/runtime/schema`
	"k8s.io/klog/v2"
)

type podRequest struct {
	podName      string
	podNamespace string
}

func (pgC *pgController) addPod(obj interface{}) {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		klog.Errorf("Failed to convert %v to corev1.Pod", obj)
		return
	}
	pgC.queue.Add(podRequest{
		podName:      pod.Name,
		podNamespace: pod.Namespace,
	})
}

// setPGForPod sets the podgroup that the input pod belongs to as pgName.
func (pgC *pgController) setPGForPod(pod *corev1.Pod, pgName string) error {
	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string)
	}
	if pod.Annotations[schedulingv1alpha1.KubeGroupNameAnnotationKey] == "" {
		pod.Annotations[schedulingv1alpha1.KubeGroupNameAnnotationKey] = pgName
	} else {
		if pod.Annotations[schedulingv1alpha1.KubeGroupNameAnnotationKey] != pgName {
			klog.Errorf("normal pod %s/%s annotations %s value is not %s, but %s",
				pod.Namespace, pod.Name, schedulingv1alpha1.KubeGroupNameAnnotationKey,
				pod.Annotations[schedulingv1alpha1.KubeGroupNameAnnotationKey])
		}
		return nil
	}

	if _, err := pgC.kubeClient.CoreV1().Pods(pod.Namespace).Update(context.TODO(), pod, metav1.UpdateOptions{}); err != nil {
		klog.Errorf("Failed to update pod <%s/%s>: %v", pod.Namespace, pod.Name, err)
		return err
	}
	return nil
}

func (pgC *pgController) createNormalPodPGIfNotExist(pod *corev1.Pod) error {
	// TODO: judge `pod.Annotations == nil` should be placed here
	pgName := helpers.GeneratePodGroupName(pod)
	if _, err := pgC.pgLister.PodGroups(pod.Namespace).Get(pgName); err != nil {
		if !apierrors.IsNotFound(err) {
			klog.Errorf("Failed to get normal PodGroup for Pod <%s/%s>: %v", pod.Namespace, pod.Name, err)
			return err
		}

		// podgroup not found, create one for the pod
		obj := &schedulingv1alpha1.PodGroup{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:       pod.Namespace,
				Name:            pgName,
				OwnerReferences: newPGOwnerReferences(pod),
				Annotations:     map[string]string{},
				Labels:          map[string]string{},
			},
			Spec: schedulingv1alpha1.PodGroupSpec{
				MinMember:         1,
				PriorityClassName: pod.Spec.PriorityClassName,
				MinResources:      calcPGMinResources(pod),
			},
		}

		// set podgroup's queue
		if queueName, ok := pod.Annotations[schedulingv1alpha1.QueueNameAnnotationKey]; ok {
			obj.Spec.Queue = queueName
		}

		// set other annotations
		if value, ok := pod.Annotations[schedulingv1alpha1.PodPreemptable]; ok {
			obj.Annotations[schedulingv1alpha1.PodPreemptable] = value
		}
		if value, ok := pod.Annotations[schedulingv1alpha1.RevocableZone]; ok {
			obj.Annotations[schedulingv1alpha1.RevocableZone] = value
		}
		if value, ok := pod.Labels[schedulingv1alpha1.PodPreemptable]; ok {
			obj.Labels[schedulingv1alpha1.PodPreemptable] = value
		}
		if value, found := pod.Annotations[schedulingv1alpha1.JDBMinAvailable]; found {
			obj.Annotations[schedulingv1alpha1.JDBMinAvailable] = value
		} else if value, found = pod.Annotations[schedulingv1alpha1.JDBMaxUnavailable]; found {
			obj.Annotations[schedulingv1alpha1.JDBMaxUnavailable] = value
		}

		if _, err = pgC.volcanoClient.SchedulingV1alpha1().PodGroups(pod.Namespace).Create(context.TODO(),
			obj, metav1.CreateOptions{}); err != nil {
			klog.Errorf("Failed to create normal PodGroup for Pod <%s/%s>: %v",
				pod.Namespace, pod.Name, err)
			return err
		}
	}
	return pgC.setPGForPod(pod, pgName)
}

// newPGOwnerReferences sets controller for the given pod.
func newPGOwnerReferences(pod *corev1.Pod) []metav1.OwnerReference {
	if len(pod.OwnerReferences) != 0 {
		for _, ownerRef := range pod.OwnerReferences {
			if ownerRef.Controller != nil && *ownerRef.Controller {
				return pod.OwnerReferences
			}
		}
	}
	gvk := schema.GroupVersionKind{
		Group:   corev1.SchemeGroupVersion.Group,
		Version: corev1.SchemeGroupVersion.Version,
		Kind:    "Pod",
	}
	ref := metav1.NewControllerRef(pod, gvk)
	return []metav1.OwnerReference{*ref}
}

func addResourceList(list, req, limit corev1.ResourceList) {
	// update Requests to list
	for name, quantity := range req {
		if value, ok := list[name]; !ok {
			list[name] = quantity.DeepCopy()
		} else {
			value.Add(quantity)
			list[name] = value
		}
	}

	if req != nil {
		return
	}
	// If Requests is omitted for a container,
	// it defaults to Limits if that is explicitly specified.
	for name, quantity := range limit {
		if value, ok := list[name]; !ok {
			list[name] = quantity.DeepCopy()
		} else {
			value.Add(quantity)
			list[name] = value
		}
	}
}

func calcPGMinResources(pod *corev1.Pod) *corev1.ResourceList {
	pgMinRes := corev1.ResourceList{}
	for _, c := range pod.Spec.Containers {
		addResourceList(pgMinRes, c.Resources.Requests, c.Resources.Limits)
	}
	return &pgMinRes
}