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

package buildkit

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"

	crd "github.com/containerbuilding/cbi/pkg/apis/cbi/v1alpha1"
	pluginapi "github.com/containerbuilding/cbi/pkg/plugin/api"
	"github.com/containerbuilding/cbi/pkg/plugin/base/backend"
	"github.com/containerbuilding/cbi/pkg/plugin/base/backend/util"
)

type BuildKit struct {
	BuildctlImage string
	BuildkitdAddr string
}

var _ backend.Backend = &BuildKit{}

func (b *BuildKit) Info(ctx context.Context, req *pluginapi.InfoRequest) (*pluginapi.InfoResponse, error) {
	res := &pluginapi.InfoResponse{
		Labels: map[string]string{
			pluginapi.LPluginName:         "buildkit",
			pluginapi.LLanguageDockerfile: "",
			pluginapi.LContextGit:         "",
			pluginapi.LContextConfigMap:   "",
		},
	}
	return res, nil
}

func (b *BuildKit) commonPodSpec(buildJob crd.BuildJob) corev1.PodSpec {
	podSpec := corev1.PodSpec{
		RestartPolicy: corev1.RestartPolicyNever,
		Containers: []corev1.Container{
			{
				Name:  "buildctl-job",
				Image: b.BuildctlImage,
				Command: []string{"buildctl", "--addr", b.BuildkitdAddr,
					"build",
					"--frontend=dockerfile.v0",
					"--no-progress", "--trace", "/dev/stdout",
				},
			},
		},
	}
	return podSpec
}

func (b *BuildKit) CreatePodTemplateSpec(ctx context.Context, buildJob crd.BuildJob) (*corev1.PodTemplateSpec, error) {
	if buildJob.Spec.Push {
		return nil, fmt.Errorf("unsupported Spec.Push: %v", buildJob.Spec.Push)
	}
	if buildJob.Spec.Language.Kind != crd.LanguageKindDockerfile {
		return nil, fmt.Errorf("unsupported Spec.Language: %v", buildJob.Spec.Language)
	}

	podSpec := b.commonPodSpec(buildJob)
	switch k := buildJob.Spec.Context.Kind; k {
	case crd.ContextKindGit:
		podSpec.Containers[0].Command = append(podSpec.Containers[0].Command, []string{
			"--frontend-opt", "context=" + buildJob.Spec.Context.GitRef.URL,
		}...)
	case crd.ContextKindConfigMap:
		volMountPath := util.InjectConfigMap(&podSpec, 0, buildJob.Spec.Context.ConfigMapRef.Name)
		podSpec.Containers[0].Command = append(podSpec.Containers[0].Command, []string{
			"--local", "context=" + volMountPath,
			"--local", "dockerfile=" + volMountPath,
		}...)
	default:
		return nil, fmt.Errorf("unsupported Spec.Context: %v", k)
	}
	return &corev1.PodTemplateSpec{
		Spec: podSpec,
	}, nil
}
