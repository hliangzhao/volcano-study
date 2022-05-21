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

package validate

import (
	"context"
	"fmt"
	batchv1alpha1 "github.com/hliangzhao/volcano/pkg/apis/batch/v1alpha1"
	schedulingv1alpha1 "github.com/hliangzhao/volcano/pkg/apis/scheduling/v1alpha1"
	jobhelpers "github.com/hliangzhao/volcano/pkg/controllers/job/helpers"
	"github.com/hliangzhao/volcano/pkg/controllers/job/plugins"
	"github.com/hliangzhao/volcano/pkg/webhooks/router"
	"github.com/hliangzhao/volcano/pkg/webhooks/schema"
	"github.com/hliangzhao/volcano/pkg/webhooks/utils"
	admissionv1 "k8s.io/api/admission/v1"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/klog/v2"
	k8score "k8s.io/kubernetes/pkg/apis/core"
	k8scorev1 "k8s.io/kubernetes/pkg/apis/core/v1"
	v1qos "k8s.io/kubernetes/pkg/apis/core/v1/helper/qos"
	k8scorevalid "k8s.io/kubernetes/pkg/apis/core/validation"
	"strings"
)

func init() {
	_ = router.RegisterAdmission(service)
}

var service = &router.AdmissionService{
	Path: "/jobs/validate",
	Func: AdmitJobs,

	Config: config,

	ValidatingConfig: &admissionregistrationv1.ValidatingWebhookConfiguration{
		Webhooks: []admissionregistrationv1.ValidatingWebhook{{
			Name: "validatejob.hliangzhao.io",
			Rules: []admissionregistrationv1.RuleWithOperations{
				{
					Operations: []admissionregistrationv1.OperationType{
						admissionregistrationv1.Create,
						admissionregistrationv1.Update,
					},
					Rule: admissionregistrationv1.Rule{
						APIGroups:   []string{"batch.hliangzhao.io"},
						APIVersions: []string{"v1alpha1"},
						Resources:   []string{"jobs"},
					},
				},
			},
		}},
	},
}

var config = &router.AdmissionServiceConfig{}

// AdmitJobs is to admit jobs and return response.
func AdmitJobs(ar admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	klog.V(3).Infof("admitting jobs -- %s", ar.Request.Operation)

	job, err := schema.DecodeJob(ar.Request.Object, ar.Request.Resource)
	if err != nil {
		return utils.ToAdmissionResponse(err)
	}
	var msg string
	reviewResponse := admissionv1.AdmissionResponse{}
	reviewResponse.Allowed = true

	switch ar.Request.Operation {
	case admissionv1.Create:
		msg = validateJobCreate(job, &reviewResponse)
	case admissionv1.Update:
		oldJob, err := schema.DecodeJob(ar.Request.OldObject, ar.Request.Resource)
		if err != nil {
			return utils.ToAdmissionResponse(err)
		}
		err = validateJobUpdate(oldJob, job)
		if err != nil {
			return utils.ToAdmissionResponse(err)
		}
	default:
		err := fmt.Errorf("expect operation to be 'CREATE' or 'UPDATE'")
		return utils.ToAdmissionResponse(err)
	}

	if !reviewResponse.Allowed {
		reviewResponse.Result = &metav1.Status{Message: strings.TrimSpace(msg)}
	}
	return &reviewResponse
}

func validateJobCreate(job *batchv1alpha1.Job, reviewResponse *admissionv1.AdmissionResponse) string {
	var msg string
	taskNames := map[string]string{}
	var totalReplicas int32

	if job.Spec.MinAvailable < 0 {
		reviewResponse.Allowed = false
		return "job 'minAvailable' must be >= 0."
	}

	if job.Spec.MaxRetry < 0 {
		reviewResponse.Allowed = false
		return "'maxRetry' cannot be less than zero."
	}

	if job.Spec.TTLSecondsAfterFinished != nil && *job.Spec.TTLSecondsAfterFinished < 0 {
		reviewResponse.Allowed = false
		return "'ttlSecondsAfterFinished' cannot be less than zero."
	}

	if len(job.Spec.Tasks) == 0 {
		reviewResponse.Allowed = false
		return "No task specified in job spec"
	}

	hasDependenciesBetweenTasks := false
	for index, task := range job.Spec.Tasks {
		if task.DependsOn != nil {
			hasDependenciesBetweenTasks = true
		}

		if task.Replicas < 0 {
			msg += fmt.Sprintf(" 'replicas' < 0 in task: %s, job: %s;", task.Name, job.Name)
		}

		if task.MinAvailable != nil {
			if *task.MinAvailable < 0 {
				msg += fmt.Sprintf(" 'minAvailable' < 0 in task: %s, job: %s;", task.Name, job.Name)
			} else if *task.MinAvailable > task.Replicas {
				msg += fmt.Sprintf(" 'minAvailable' is greater than 'replicas' in task: %s, job: %s;", task.Name, job.Name)
			}
		}

		// count replicas
		totalReplicas += task.Replicas

		// validate task name
		if errMsgs := validation.IsDNS1123Label(task.Name); len(errMsgs) > 0 {
			msg += fmt.Sprintf(" %v;", errMsgs)
		}

		// duplicate task name
		if _, found := taskNames[task.Name]; found {
			msg += fmt.Sprintf(" duplicated task name %s;", task.Name)
			break
		} else {
			taskNames[task.Name] = task.Name
		}

		if err := validatePolicies(task.Policies, field.NewPath("spec.tasks.policies")); err != nil {
			msg += err.Error() + fmt.Sprintf(" valid events are %v, valid actions are %v;",
				getValidEvents(), getValidActions())
		}
		podName := jobhelpers.MakePodName(job.Name, task.Name, index)
		msg += validateK8sPodNameLength(podName)
		msg += validateTaskTemplate(task, job, index)
	}

	msg += validateJobName(job)

	if totalReplicas < job.Spec.MinAvailable {
		msg += " job 'minAvailable' should not be greater than total replicas in tasks;"
	}

	if err := validatePolicies(job.Spec.Policies, field.NewPath("spec.policies")); err != nil {
		msg = msg + err.Error() + fmt.Sprintf(" valid events are %v, valid actions are %v;",
			getValidEvents(), getValidActions())
	}

	// invalid job plugins
	if len(job.Spec.Plugins) != 0 {
		for name := range job.Spec.Plugins {
			if _, found := plugins.GetPluginBuilder(name); !found {
				msg += fmt.Sprintf(" unable to find job plugin: %s;", name)
			}
		}
	}

	if err := validateIO(job.Spec.Volumes); err != nil {
		msg += err.Error()
	}

	queue, err := config.VolcanoClient.SchedulingV1alpha1().Queues().Get(context.TODO(), job.Spec.Queue, metav1.GetOptions{})
	if err != nil {
		msg += fmt.Sprintf(" unable to find job queue: %v;", err)
	} else if queue.Status.State != schedulingv1alpha1.QueueStateOpen {
		msg += fmt.Sprintf(" can only submit job to queue with state `Open`, "+
			"queue `%s` status is `%s`;", queue.Name, queue.Status.State)
	}

	if hasDependenciesBetweenTasks {
		_, isDag := topoSort(job)
		if !isDag {
			msg += " job has dependencies between tasks, but doesn't form a directed acyclic graph(DAG);"
		}
	}

	if msg != "" {
		reviewResponse.Allowed = false
	}

	return msg
}

func validateJobUpdate(old, new *batchv1alpha1.Job) error {
	var totalReplicas int32
	for _, task := range new.Spec.Tasks {
		if task.Replicas < 0 {
			return fmt.Errorf("'replicas' must be >= 0 in task: %s", task.Name)
		}

		if task.MinAvailable != nil {
			if *task.MinAvailable < 0 {
				return fmt.Errorf("'minAvailable' must be >= 0 in task: %s", task.Name)
			} else if *task.MinAvailable > task.Replicas {
				return fmt.Errorf("'minAvailable' must be <= 'replicas' in task: %s", task.Name)
			}
		}

		// count replicas
		totalReplicas += task.Replicas
	}
	if new.Spec.MinAvailable > totalReplicas {
		return fmt.Errorf("job 'minAvailable' must not be greater than total replicas")
	}
	if new.Spec.MinAvailable < 0 {
		return fmt.Errorf("job 'minAvailable' must be >= 0")
	}

	if len(old.Spec.Tasks) != len(new.Spec.Tasks) {
		return fmt.Errorf("job updates may not add or remove tasks")
	}
	// other fields under spec are not allowed to mutate
	new.Spec.MinAvailable = old.Spec.MinAvailable
	new.Spec.PriorityClassName = old.Spec.PriorityClassName
	for i := range new.Spec.Tasks {
		new.Spec.Tasks[i].Replicas = old.Spec.Tasks[i].Replicas
		new.Spec.Tasks[i].MinAvailable = old.Spec.Tasks[i].MinAvailable
	}

	// job controller will update the pvc name if not provided
	for i := range new.Spec.Volumes {
		if new.Spec.Volumes[i].VolumeClaim != nil {
			new.Spec.Volumes[i].VolumeClaimName = ""
		}
	}
	for i := range old.Spec.Volumes {
		if old.Spec.Volumes[i].VolumeClaim != nil {
			old.Spec.Volumes[i].VolumeClaimName = ""
		}
	}

	if !equality.Semantic.DeepEqual(new.Spec, old.Spec) {
		return fmt.Errorf("job updates may not change fields other than `minAvailable`, `tasks[*].replicas under spec`")
	}

	return nil
}

func validateTaskTemplate(task batchv1alpha1.TaskSpec, job *batchv1alpha1.Job, index int) string {
	var v1PodTemplate corev1.PodTemplate
	v1PodTemplate.Template = *task.Template.DeepCopy()
	k8scorev1.SetObjectDefaults_PodTemplate(&v1PodTemplate)

	var coreTemplateSpec k8score.PodTemplateSpec
	_ = k8scorev1.Convert_v1_PodTemplateSpec_To_core_PodTemplateSpec(&v1PodTemplate.Template, &coreTemplateSpec, nil)

	// Skip verify container SecurityContext.Privileged as it depends on
	// the kube api-server `allow-privileged` flag.
	for i, container := range coreTemplateSpec.Spec.InitContainers {
		if container.SecurityContext != nil && container.SecurityContext.Privileged != nil {
			coreTemplateSpec.Spec.InitContainers[i].SecurityContext.Privileged = nil
		}
	}
	for i, container := range coreTemplateSpec.Spec.Containers {
		if container.SecurityContext != nil && container.SecurityContext.Privileged != nil {
			coreTemplateSpec.Spec.Containers[i].SecurityContext.Privileged = nil
		}
	}

	corePodTemplate := k8score.PodTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      task.Name,
			Namespace: job.Namespace,
		},
		Template: coreTemplateSpec,
	}

	opts := k8scorevalid.PodValidationOptions{}
	if allErrs := k8scorevalid.ValidatePodTemplate(&corePodTemplate, opts); len(allErrs) > 0 {
		msg := fmt.Sprintf("spec.task[%d].", index)
		for index := range allErrs {
			msg += allErrs[index].Error() + ". "
		}
		return msg
	}

	msg := validateTaskTopoPolicy(task, index)
	if msg != "" {
		return msg
	}

	return ""
}

func validateK8sPodNameLength(podName string) string {
	if errMsgs := validation.IsQualifiedName(podName); len(errMsgs) > 0 {
		return fmt.Sprintf("create pod with name %s validate failed %v;", podName, errMsgs)
	}
	return ""
}

func validateJobName(job *batchv1alpha1.Job) string {
	if errMsgs := validation.IsQualifiedName(job.Name); len(errMsgs) > 0 {
		return fmt.Sprintf("create job with name %s validate failed %v", job.Name, errMsgs)
	}
	return ""
}

func validateTaskTopoPolicy(task batchv1alpha1.TaskSpec, index int) string {
	if task.TopologyPolicy == "" || task.TopologyPolicy == batchv1alpha1.None {
		return ""
	}

	template := task.Template.DeepCopy()

	for id, container := range template.Spec.Containers {
		if len(container.Resources.Requests) == 0 {
			template.Spec.Containers[id].Resources.Requests = container.Resources.Limits.DeepCopy()
		}
	}

	for id, container := range template.Spec.InitContainers {
		if len(container.Resources.Requests) == 0 {
			template.Spec.InitContainers[id].Resources.Requests = container.Resources.Limits.DeepCopy()
		}
	}

	pod := &corev1.Pod{
		Spec: template.Spec,
	}

	if v1qos.GetPodQOS(pod) != corev1.PodQOSGuaranteed {
		return fmt.Sprintf("spec.task[%d] isn't Guaranteed pod, kind=%v", index, v1qos.GetPodQOS(pod))
	}

	for id, container := range append(template.Spec.Containers, template.Spec.InitContainers...) {
		requestNum := guaranteedCPUs(container)
		if requestNum == 0 {
			return fmt.Sprintf("the cpu request isn't  an integer in spec.task[%d] container[%d].",
				index, id)
		}
	}

	return ""
}

func guaranteedCPUs(container corev1.Container) int {
	cpuQuantity := container.Resources.Requests[corev1.ResourceCPU]
	if cpuQuantity.Value()*1000 != cpuQuantity.MilliValue() {
		return 0
	}

	return int(cpuQuantity.Value())
}
