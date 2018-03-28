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
	Image    string   `json:"image"`
	Push     bool     `json:"push"`
	Language Language `json:"language"`
	Context  Context  `json:"context"`
}

// Language
type Language struct {
	Kind string `json:"kind"`
}

// LanguageKindDockerfile
const LanguageKindDockerfile = "Dockerfile"

// Context
type Context struct {
	Kind   string `json:"kind"`
	GitRef GitRef `json:"gitRef"`
}

// ContextKindGit
const ContextKindGit = "Git"

type GitRef struct {
	URL string `json:"url"`
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
