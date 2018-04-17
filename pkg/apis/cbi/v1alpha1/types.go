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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// BuildJob is a specification for a BuildJob resource
type BuildJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BuildJobSpec   `json:"spec"`
	Status BuildJobStatus `json:"status"`
}

// BuildJobSpec is the spec for a BuildJob resource
type BuildJobSpec struct {
	// Image is used for pushing the artifact to the registry.
	// e.g. `example.com:foo/bar:latest`
	Image string `json:"image"`
	// Push pushes the image.
	// +optional
	Push bool `json:"push"`
	// Language specifies the language.
	Language Language `json:"language"`
	// Context specifies the context.
	Context Context `json:"context"`
	// PluginSelector specifies additional hints for selecting the plugin
	// using the plugin labels.
	// e.g. `plugin.name = docker`.
	//
	// Controller implementations MUST have logic for selecting the default
	// plugin using Language and Context.
	// So, usually users don't need to set PluginSelector explicitly, especially
	// `language.*` labels and `context.*` labels.
	//
	// When PluginSelector is specified, Controller SHOULD select a plugin
	// that satisfies both its default logic and PluginSelector.
	//
	// +optional
	PluginSelector string `json:"pluginSelector"`
}

// Language specifies the language.
type Language struct {
	Kind string `json:"kind"`
}

// LanguageKindDockerfile stands for Dockerfile.
// When BuildJob.Language.Kind is set to LanguageKindDockerfile, the controller
// MUST add "language.dockerfile" to its default plugin selector logic.
const LanguageKindDockerfile = "Dockerfile"

// Context specifies the context.
type Context struct {
	Kind         string       `json:"kind"`
	GitRef       GitRef       `json:"gitRef"`
	ConfigMapRef ConfigMapRef `json:"configMapRef"`
}

// ContextKindGit stands for Git context.
// When BuildJob.Context.Kind is set to ContextKindGit, the controller
// MUST add "context.git" to its default plugin selector logic.
const ContextKindGit = "Git"

type GitRef struct {
	// URL is the docker-style URL.
	// e.g. `git://example.com/myrepo#mybranch:myfolder`.
	// See https://docs.docker.com/engine/reference/commandline/build/#git-repositories
	URL string `json:"url"`
	// TODO: add separate fields for host, branch, subdir...
	// Then we should deprecate this docker-style URL.
}

// ContextKindConfigMap stands for ConfigMap context.
// When BuildJob.Context.Kind is set to ContextKindConfigMap, the controller
// MUST add "context.configmap" to its default plugin selector logic.
const ContextKindConfigMap = "ConfigMap"

type ConfigMapRef struct {
	// Kubernetes configmap in the current namespace
	Name string `json:"name"`
}

// BuildJobStatus is the status for a BuildJob resource
type BuildJobStatus struct {
	Job string `json:"job"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// BuildJobList is a list of BuildJob resources
type BuildJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []BuildJob `json:"items"`
}
