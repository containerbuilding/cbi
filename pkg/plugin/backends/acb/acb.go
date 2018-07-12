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

package acb

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	corev1 "k8s.io/api/core/v1"

	crd "github.com/containerbuilding/cbi/pkg/apis/cbi/v1alpha1"
	pluginapi "github.com/containerbuilding/cbi/pkg/plugin/api"
	"github.com/containerbuilding/cbi/pkg/plugin/base"
	"github.com/containerbuilding/cbi/pkg/plugin/base/cbipluginhelper"
)

const (
	// Secret name. Needs to contain PEM/DER key, as "cert"
	AnnotationSecret = "cbi-acb/secret"
	// Azure App ID
	AnnotationAppID = "cbi-acb/app-id"
	// Azure Tenant
	AnnotationTenant = "cbi-acb/tenant"
)

type ACB struct {
	Image  string
	Helper cbipluginhelper.Helper
}

var _ base.Backend = &ACB{}

func (b *ACB) Info(ctx context.Context, req *pluginapi.InfoRequest) (*pluginapi.InfoResponse, error) {
	res := &pluginapi.InfoResponse{
		Labels: map[string]string{
			pluginapi.LPluginName:                           "acb",
			pluginapi.LLanguage(crd.LanguageKindDockerfile): "",
		},
	}
	for k, v := range cbipluginhelper.Labels {
		res.Labels[k] = v
	}
	return res, nil
}

func (b *ACB) commonPodSpec(buildJob crd.BuildJob) (*corev1.PodSpec, error) {
	rootConfigVol := corev1.Volume{
		Name: "cbi-acb-root-config",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}
	rootConfigVolMount := corev1.VolumeMount{
		Name:      rootConfigVol.Name,
		MountPath: "/root/.azure",
	}
	secretDefaultMode := int32(0400)
	secretVol := corev1.Volume{
		Name: "cbi-acb-secret",
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName:  buildJob.Annotations[AnnotationSecret],
				DefaultMode: &secretDefaultMode,
			},
		},
	}
	secretVolMount := corev1.VolumeMount{
		Name:      secretVol.Name,
		MountPath: "/cbi-acbsecret",
	}
	reg, image, err := splitTarget(buildJob.Spec.Registry.Target)
	if err != nil {
		return nil, err
	}
	podSpec := corev1.PodSpec{
		RestartPolicy: corev1.RestartPolicyNever,
		Volumes:       []corev1.Volume{rootConfigVol, secretVol},
		InitContainers: []corev1.Container{
			{
				Name:  "cbi-acb-init",
				Image: b.Image,
				Command: []string{"az", "login",
					"--service-principal",
					"--username", buildJob.Annotations[AnnotationAppID],
					"--tenant", buildJob.Annotations[AnnotationTenant],
					// NOTE: --password needs to be an absolute path.
					// Otherwise it does not working when cwd is changed.
					"--password", secretVolMount.MountPath + "/cert"},
				VolumeMounts: []corev1.VolumeMount{rootConfigVolMount, secretVolMount},
			},
		},
		Containers: []corev1.Container{
			{
				Name:    "acb-job",
				Image:   b.Image,
				Command: []string{"az", "acr", "build", "--registry", reg, "--image", image},
				// ~/.azure/accessTokens.json refers to the PEM in the secret volume.
				VolumeMounts: []corev1.VolumeMount{rootConfigVolMount, secretVolMount},
			},
		},
	}
	return &podSpec, nil
}

func splitTarget(target string) (string, string, error) {
	re := regexp.MustCompile(`(.*)\.azurecr\.io/(.*)`)
	matches := re.FindStringSubmatch(target)
	if len(matches) != 3 {
		return "", "", fmt.Errorf("needs to be in *azurecr.io namespace: %q", target)
	}
	return matches[1], matches[2], nil
}

func (b *ACB) CreatePodTemplateSpec(ctx context.Context, buildJob crd.BuildJob) (*corev1.PodTemplateSpec, error) {
	switch k := strings.ToLower(string(buildJob.Spec.Language.Kind)); k {
	case strings.ToLower(string(crd.LanguageKindDockerfile)):
	default:
		return nil, fmt.Errorf("unsupported Spec.Language: %v", buildJob.Spec.Language)
	}
	if buildJob.Spec.Registry.SecretRef.Name != "" {
		return nil, fmt.Errorf("ACB plugin requires Spec.Registry.SecretRef to be empty (use cbi-acb/secret annotation instead with Azure service principal)")
	}
	for _, a := range []string{AnnotationSecret, AnnotationAppID, AnnotationTenant} {
		if buildJob.Annotations[a] == "" {
			return nil, fmt.Errorf("ACB plugin requires annotation %q", a)
		}
	}
	podSpec, err := b.commonPodSpec(buildJob)
	if err != nil {
		return nil, err
	}
	if !buildJob.Spec.Registry.Push {
		podSpec.Containers[0].Args = append(podSpec.Containers[0].Args, []string{"--no-push", "true"}...)
	}
	injector := cbipluginhelper.Injector{
		Helper:        b.Helper,
		TargetPodSpec: podSpec,
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
		Spec: *podSpec,
	}, nil
}
