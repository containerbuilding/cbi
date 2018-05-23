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

package gcb

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"

	crd "github.com/containerbuilding/cbi/pkg/apis/cbi/v1alpha1"
	pluginapi "github.com/containerbuilding/cbi/pkg/plugin/api"
	"github.com/containerbuilding/cbi/pkg/plugin/base"
	"github.com/containerbuilding/cbi/pkg/plugin/base/cbipluginhelper"
)

const (
	// Secret name. Needs to contain Google Cloud key, as "json"
	AnnotationSecret = "cbi-gcb/secret"
	// Google Cloud project name
	AnnotationProject = "cbi-gcb/project"
)

type GCB struct {
	Image  string
	Helper cbipluginhelper.Helper
}

var _ base.Backend = &GCB{}

func (b *GCB) Info(ctx context.Context, req *pluginapi.InfoRequest) (*pluginapi.InfoResponse, error) {
	res := &pluginapi.InfoResponse{
		Labels: map[string]string{
			pluginapi.LPluginName:         "gcb",
			pluginapi.LLanguageDockerfile: "",
		},
	}
	for k, v := range cbipluginhelper.Labels {
		res.Labels[k] = v
	}
	return res, nil
}

func (b *GCB) commonPodSpec(buildJob crd.BuildJob) corev1.PodSpec {
	rootConfigVol := corev1.Volume{
		Name: "cbi-gcb-root-config",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}
	rootConfigVolMount := corev1.VolumeMount{
		Name:      rootConfigVol.Name,
		MountPath: "/root/.config",
	}
	secretDefaultMode := int32(0400)
	secretVol := corev1.Volume{
		Name: "cbi-gcb-secret",
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName:  buildJob.Annotations[AnnotationSecret],
				DefaultMode: &secretDefaultMode,
			},
		},
	}
	secretVolMount := corev1.VolumeMount{
		Name:      secretVol.Name,
		MountPath: "/cbi-gcbsecret",
	}
	podSpec := corev1.PodSpec{
		RestartPolicy: corev1.RestartPolicyNever,
		Volumes:       []corev1.Volume{rootConfigVol, secretVol},
		InitContainers: []corev1.Container{
			{
				Name:         "cbi-gcb-init",
				Image:        b.Image,
				WorkingDir:   secretVolMount.MountPath,
				Command:      []string{"gcloud", "auth", "activate-service-account", "--key-file=json"},
				VolumeMounts: []corev1.VolumeMount{rootConfigVolMount, secretVolMount},
			},
		},
		Containers: []corev1.Container{
			{
				Name:         "gcb-job",
				Image:        b.Image,
				Env:          []corev1.EnvVar{{Name: "CLOUDSDK_CORE_PROJECT", Value: buildJob.Annotations[AnnotationProject]}},
				Command:      []string{"gcloud", "container", "builds", "submit", "-t", buildJob.Spec.Registry.Target},
				VolumeMounts: []corev1.VolumeMount{rootConfigVolMount},
			},
		},
	}
	return podSpec
}

func (b *GCB) CreatePodTemplateSpec(ctx context.Context, buildJob crd.BuildJob) (*corev1.PodTemplateSpec, error) {
	if buildJob.Spec.Language.Kind != crd.LanguageKindDockerfile {
		return nil, fmt.Errorf("unsupported Spec.Language: %v", buildJob.Spec.Language)
	}
	if !buildJob.Spec.Registry.Push {
		return nil, fmt.Errorf("GCB plugin requires Spec.Registry.Push to be true")
	}
	if buildJob.Spec.Registry.SecretRef.Name != "" {
		return nil, fmt.Errorf("GCB plugin requires Spec.Registry.SecretRef to be empty (use cbi-gcb/secret annotation instead with Google Cloud service account)")
	}
	for _, a := range []string{AnnotationSecret, AnnotationProject} {
		if buildJob.Annotations[a] == "" {
			return nil, fmt.Errorf("GCB plugin requires annotation %q", a)
		}
	}
	podSpec := b.commonPodSpec(buildJob)
	injector := cbipluginhelper.Injector{
		Helper:        b.Helper,
		TargetPodSpec: &podSpec,
	}
	ctxInjector := cbipluginhelper.ContextInjector{
		Injector: injector,
	}
	ctxPath, err := ctxInjector.Inject(buildJob.Spec.Context)
	if err != nil {
		return nil, err
	}
	podSpec.Containers[0].Args = append(podSpec.Containers[0].Args, ctxPath)
	return &corev1.PodTemplateSpec{
		Spec: podSpec,
	}, nil
}
