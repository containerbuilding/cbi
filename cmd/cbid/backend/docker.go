/*
Copyright The CBI Authors.

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

package backend

import (
	"fmt"

	cbiv1alpha1 "github.com/containerbuilding/cbi/pkg/apis/cbi/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// NewDockerJob creates a new Job for a BuildJob resource.
func NewDockerJob(jobName string, buildJob *cbiv1alpha1.BuildJob) (*batchv1.Job, error) {
	if buildJob.Spec.Push {
		return nil, fmt.Errorf("unsupported Spec.Push: %v", buildJob.Spec.Push)
	}
	if buildJob.Spec.Language.Kind != cbiv1alpha1.LanguageKindDockerfile {
		return nil, fmt.Errorf("unsupported Spec.Language: %v", buildJob.Spec.Language)
	}
	if buildJob.Spec.Context.Kind != cbiv1alpha1.ContextKindGit {
		return nil, fmt.Errorf("unsupported Spec.Context: %v", buildJob.Spec.Context)
	}

	hostPathFile := corev1.HostPathFile
	podSpec := corev1.PodSpec{
		RestartPolicy: corev1.RestartPolicyNever,
		Containers: []corev1.Container{
			{
				Name: "docker-job",
				// The upstream docker:18.03 lacks git
				Image: "nathanielc/docker-client:17.03.1-ce",
				Command: []string{"docker", "build", "-t", buildJob.Spec.Image,
					buildJob.Spec.Context.GitRef.URL,
				},
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      "docker-sock-volume",
						MountPath: "/var/run/docker.sock",
					},
				},
			},
		},
		Volumes: []corev1.Volume{
			{
				Name: "docker-sock-volume",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: "/var/run/docker.sock",
						Type: &hostPathFile,
					},
				},
			},
		},
	}
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: buildJob.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(buildJob, schema.GroupVersionKind{
					Group:   cbiv1alpha1.SchemeGroupVersion.Group,
					Version: cbiv1alpha1.SchemeGroupVersion.Version,
					Kind:    "BuildJob",
				}),
			},
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: podSpec,
			},
		},
	}, nil
}
