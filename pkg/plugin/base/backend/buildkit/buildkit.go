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
)

type BuildKit struct {
	BuildctlImage string
	BuildkitdAddr string
}

var _ backend.Backend = &BuildKit{}

func (b *BuildKit) Info(ctx context.Context, req *pluginapi.InfoRequest) (*pluginapi.InfoResponse, error) {
	res := &pluginapi.InfoResponse{
		SupportedLanguageKind: []string{
			crd.LanguageKindDockerfile,
		},
		SupportedContextKind: []string{
			crd.ContextKindGit,
		},
	}
	return res, nil
}

func (b *BuildKit) CreatePodTemplateSpec(ctx context.Context, buildJob crd.BuildJob) (*corev1.PodTemplateSpec, error) {
	if buildJob.Spec.Push {
		return nil, fmt.Errorf("unsupported Spec.Push: %v", buildJob.Spec.Push)
	}
	if buildJob.Spec.Language.Kind != crd.LanguageKindDockerfile {
		return nil, fmt.Errorf("unsupported Spec.Language: %v", buildJob.Spec.Language)
	}
	if buildJob.Spec.Context.Kind != crd.ContextKindGit {
		return nil, fmt.Errorf("unsupported Spec.Context: %v", buildJob.Spec.Context)
	}

	podSpec := corev1.PodSpec{
		RestartPolicy: corev1.RestartPolicyNever,
		Containers: []corev1.Container{
			{
				Name:  "buildctl-job",
				Image: b.BuildctlImage,
				Command: []string{"buildctl", "--addr", b.BuildkitdAddr,
					"build",
					"--frontend=dockerfile.v0",
					"--frontend-opt", "context=" + buildJob.Spec.Context.GitRef.URL,
					"--no-progress", "--trace", "/dev/stdout",
				},
			},
		},
	}
	return &corev1.PodTemplateSpec{
		Spec: podSpec,
	}, nil
}
