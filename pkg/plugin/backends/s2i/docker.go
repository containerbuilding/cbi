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

package s2i

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"

	crd "github.com/containerbuilding/cbi/pkg/apis/cbi/v1alpha1"
	pluginapi "github.com/containerbuilding/cbi/pkg/plugin/api"
	"github.com/containerbuilding/cbi/pkg/plugin/base"
	"github.com/containerbuilding/cbi/pkg/plugin/base/cbipluginhelper"
	"github.com/containerbuilding/cbi/pkg/plugin/base/registryutil"
)

type S2I struct {
	Image  string
	Helper cbipluginhelper.Helper
}

var _ base.Backend = &S2I{}

func (b *S2I) Info(ctx context.Context, req *pluginapi.InfoRequest) (*pluginapi.InfoResponse, error) {
	res := &pluginapi.InfoResponse{
		Labels: map[string]string{
			pluginapi.LPluginName:                    "s2i",
			pluginapi.LLanguage(crd.LanguageKindS2I): "",
		},
	}
	for k, v := range cbipluginhelper.Labels {
		res.Labels[k] = v
	}
	return res, nil
}

func (b *S2I) commonPodSpec(buildJob crd.BuildJob) corev1.PodSpec {
	push := "0"
	if buildJob.Spec.Registry.Push {
		push = "1"
	}
	hostPathFile := corev1.HostPathFile
	// TODO(AkihiroSuda): support NodeSelector
	podSpec := corev1.PodSpec{
		RestartPolicy: corev1.RestartPolicyNever,
		Containers: []corev1.Container{
			{
				Name:    "s2i-job",
				Image:   b.Image,
				Command: []string{
					// needs to be "s2i-build-push.sh", not s2i itself.
				},
				Env: []corev1.EnvVar{
					{
						Name:  "SBP_IMAGE_NAME",
						Value: buildJob.Spec.Registry.Target,
					},
					{
						Name:  "SBP_PUSH",
						Value: push,
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

func (b *S2I) CreatePodTemplateSpec(ctx context.Context, buildJob crd.BuildJob) (*corev1.PodTemplateSpec, error) {
	switch k := strings.ToLower(string(buildJob.Spec.Language.Kind)); k {
	case strings.ToLower(string(crd.LanguageKindS2I)):
		// NOP
	default:
		return nil, fmt.Errorf("unsupported Spec.Language: %v", buildJob.Spec.Language)
	}
	if buildJob.Spec.Language.S2I.BaseImage == "" {
		return nil, fmt.Errorf("Spec.Language.S2I.BaseImage is required")
	}
	if buildJob.Spec.Registry.Target == "" {
		return nil, fmt.Errorf("Spec.Registry.Target is required")
	}
	podSpec := b.commonPodSpec(buildJob)
	if buildJob.Spec.Registry.Push && buildJob.Spec.Registry.SecretRef.Name != "" {
		if err := registryutil.InjectRegistrySecret(&podSpec, 0, "/root", buildJob.Spec.Registry.SecretRef); err != nil {
			return nil, err
		}
	}
	injector := cbipluginhelper.Injector{
		Helper:        b.Helper,
		TargetPodSpec: &podSpec,
	}
	sbpPath, err := injector.InjectFile("/s2i-build-push.sh")
	if err != nil {
		return nil, err
	}
	ctxInjector := cbipluginhelper.ContextInjector{
		Injector: injector,
	}
	ctxPath, err := ctxInjector.Inject(buildJob.Spec.Context)
	if err != nil {
		return nil, err
	}
	podSpec.Containers[0].Command = []string{sbpPath, ctxPath, buildJob.Spec.Language.S2I.BaseImage, buildJob.Spec.Registry.Target}
	return &corev1.PodTemplateSpec{
		Spec: podSpec,
	}, nil
}
