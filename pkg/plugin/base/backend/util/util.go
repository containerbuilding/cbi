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

package util

import (
	"path/filepath"

	corev1 "k8s.io/api/core/v1"
)

// InjectConfigMap injects a config map to podSpec and returns the context path
func InjectConfigMap(podSpec *corev1.PodSpec, containerIdx int, configMapName string) string {
	const (
		// cmVol is a configmap volume (with symlinks)
		cmVolName      = "cbi-cmcontext-tmp"
		cmVolMountPath = "/cbi-cmcontext-tmp"
		// vol is an emptyDir volume (without symlinks)
		volName           = "cbi-cmcontext"
		volMountPath      = "/cbi-cmcontext"
		volContextSubpath = "context"
		// initContainer is used for converting cmVol to vol so as to eliminate symlinks
		initContainerName  = "cbi-cmcontext-init"
		initContainerImage = "busybox"
	)
	contextPath := filepath.Join(volMountPath, volContextSubpath)
	cmVol := corev1.Volume{
		Name: cmVolName,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: configMapName,
				},
			},
		},
	}
	vol := corev1.Volume{
		Name: volName,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}
	podSpec.Volumes = append(podSpec.Volumes, cmVol, vol)

	initContainer := corev1.Container{
		Name:    initContainerName,
		Image:   initContainerImage,
		Command: []string{"cp", "-rL", cmVolMountPath, contextPath},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      volName,
				MountPath: volMountPath,
			},
			{
				Name:      cmVolName,
				MountPath: cmVolMountPath,
			},
		},
	}
	podSpec.InitContainers = append(podSpec.InitContainers, initContainer)

	podSpec.Containers[containerIdx].VolumeMounts = append(podSpec.Containers[containerIdx].VolumeMounts,
		corev1.VolumeMount{
			Name:      volName,
			MountPath: volMountPath,
		},
	)
	return contextPath
}
