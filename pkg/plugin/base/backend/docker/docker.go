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

package docker

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"

	crd "github.com/containerbuilding/cbi/pkg/apis/cbi/v1alpha1"
	pluginapi "github.com/containerbuilding/cbi/pkg/plugin/api"
	"github.com/containerbuilding/cbi/pkg/plugin/base/backend"
	"github.com/containerbuilding/cbi/pkg/plugin/base/backend/util"
)

type Docker struct {
	Image string
}

var _ backend.Backend = &Docker{}

func (b *Docker) Info(ctx context.Context, req *pluginapi.InfoRequest) (*pluginapi.InfoResponse, error) {
	res := &pluginapi.InfoResponse{
		Labels: map[string]string{
			pluginapi.LPluginName:         "docker",
			pluginapi.LLanguageDockerfile: "",
			pluginapi.LContextGit:         "",
			pluginapi.LContextConfigMap:   "",
		},
	}
	return res, nil
}

func (b *Docker) commonPodSpec(buildJob crd.BuildJob) corev1.PodSpec {
	hostPathFile := corev1.HostPathFile
	// TODO(AkihiroSuda): support NodeSelector
	podSpec := corev1.PodSpec{
		RestartPolicy: corev1.RestartPolicyNever,
		Containers: []corev1.Container{
			{
				Name:  "docker-job",
				Image: b.Image,
				Command: []string{
					"/docker-build-push.sh",
				},
				Env: []corev1.EnvVar{
					{
						Name:  "DBP_DOCKER_BINARY",
						Value: "docker",
					},
					{
						Name:  "DBP_IMAGE_NAME",
						Value: buildJob.Spec.Registry.Target,
					},
					{
						Name:  "DBP_PUSH",
						Value: "0",
					},
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
	return podSpec

}

func (b *Docker) CreatePodTemplateSpec(ctx context.Context, buildJob crd.BuildJob) (*corev1.PodTemplateSpec, error) {
	if buildJob.Spec.Registry.Push {
		return nil, fmt.Errorf("unsupported Spec.Registry.Push: %v", buildJob.Spec.Registry.Push)
	}
	if buildJob.Spec.Language.Kind != crd.LanguageKindDockerfile {
		return nil, fmt.Errorf("unsupported Spec.Language: %v", buildJob.Spec.Language)
	}

	podSpec := b.commonPodSpec(buildJob)
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
