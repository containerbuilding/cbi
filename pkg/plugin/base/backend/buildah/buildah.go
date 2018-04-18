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

package buildah

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"

	crd "github.com/containerbuilding/cbi/pkg/apis/cbi/v1alpha1"
	pluginapi "github.com/containerbuilding/cbi/pkg/plugin/api"
	"github.com/containerbuilding/cbi/pkg/plugin/base/backend"
	"github.com/containerbuilding/cbi/pkg/plugin/base/backend/util"
)

type Buildah struct {
	Image string
}

var _ backend.Backend = &Buildah{}

func (b *Buildah) Info(ctx context.Context, req *pluginapi.InfoRequest) (*pluginapi.InfoResponse, error) {
	res := &pluginapi.InfoResponse{
		Labels: map[string]string{
			pluginapi.LPluginName:         "buildah",
			pluginapi.LLanguageDockerfile: "",
			pluginapi.LContextGit:         "",
			pluginapi.LContextConfigMap:   "",
		},
	}
	return res, nil
}

func (b *Buildah) commonPodSpec(buildJob crd.BuildJob) corev1.PodSpec {
	push := "0"
	if buildJob.Spec.Registry.Push {
		push = "1"
	}
	// TODO(AkihiroSuda): support non-privileged
	privileged := true
	podSpec := corev1.PodSpec{
		RestartPolicy: corev1.RestartPolicyNever,
		Containers: []corev1.Container{
			{
				Name:  "buildah-job",
				Image: b.Image,
				Command: []string{
					"/docker-build-push.sh",
				},
				Env: []corev1.EnvVar{
					{
						Name:  "DBP_DOCKER_BINARY",
						Value: "buildah",
					},
					{
						Name:  "DBP_IMAGE_NAME",
						Value: buildJob.Spec.Registry.Target,
					},
					{
						Name:  "DBP_DIALECT",
						Value: "buildah",
					},
					{
						Name:  "DBP_PUSH",
						Value: push,
					},
				},
				VolumeMounts: []corev1.VolumeMount{
					{
						Name: "buildah-storage-volume",
						// we need this volume for overlay storage driver.
						MountPath: "/var/lib/containers/storage",
					},
				},
				SecurityContext: &corev1.SecurityContext{
					Privileged: &privileged,
				},
			},
		},
		Volumes: []corev1.Volume{
			{
				Name: "buildah-storage-volume",
				VolumeSource: corev1.VolumeSource{
					// not persistent until buildah supports preserving cache
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
		},
	}
	return podSpec
}

func (b *Buildah) CreatePodTemplateSpec(ctx context.Context, buildJob crd.BuildJob) (*corev1.PodTemplateSpec, error) {
	if buildJob.Spec.Language.Kind != crd.LanguageKindDockerfile {
		return nil, fmt.Errorf("unsupported Spec.Language: %v", buildJob.Spec.Language)
	}
	podSpec := b.commonPodSpec(buildJob)
	if buildJob.Spec.Registry.Push {
		if len(buildJob.Spec.Registry.SecretRefs) != 1 {
			return nil, fmt.Errorf("expected 1 Spec.Registry.SecretRefs, got %d", len(buildJob.Spec.Registry.SecretRefs))
		}
		util.InjectRegistrySecret(&podSpec, 0, "/root", buildJob.Spec.Registry.SecretRefs[0])
	}
	switch k := buildJob.Spec.Context.Kind; k {
	case crd.ContextKindGit:
		podSpec.Containers[0].Command = append(podSpec.Containers[0].Command, []string{
			buildJob.Spec.Context.Git.URL,
		}...)
	case crd.ContextKindConfigMap:
		volMountPath := util.InjectConfigMap(&podSpec, 0, buildJob.Spec.Context.ConfigMapRef)
		podSpec.Containers[0].Command = append(podSpec.Containers[0].Command, []string{
			volMountPath,
		}...)
	default:
		return nil, fmt.Errorf("unsupported Spec.Context: %v", k)
	}
	return &corev1.PodTemplateSpec{
		Spec: podSpec,
	}, nil
}
