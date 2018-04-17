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

package generic

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"

	crd "github.com/containerbuilding/cbi/pkg/apis/cbi/v1alpha1"
	api "github.com/containerbuilding/cbi/pkg/plugin/api"
)

func defaultRequirements(bj crd.BuildJob) ([]labels.Requirement, error) {
	var requirements []labels.Requirement
	switch k := bj.Spec.Language.Kind; k {
	case crd.LanguageKindDockerfile:
		r, err := labels.NewRequirement(api.LLanguageDockerfile, selection.Exists, nil)
		if err != nil {
			return nil, err
		}
		requirements = append(requirements, *r)
	default:
		return nil, &errors.UnexpectedObjectError{
			Object: &bj,
		}
	}
	switch k := bj.Spec.Context.Kind; k {
	case crd.ContextKindGit:
		r, err := labels.NewRequirement(api.LContextGit, selection.Exists, nil)
		if err != nil {
			return nil, err
		}
		requirements = append(requirements, *r)
	default:
		return nil, &errors.UnexpectedObjectError{
			Object: &bj,
		}
	}
	return requirements, nil
}

func labelsSelector(bj crd.BuildJob) (labels.Selector, error) {
	sel := labels.NewSelector()
	reqs, err := defaultRequirements(bj)
	if err != nil {
		return nil, err
	}
	sel = sel.Add(reqs...)
	if s := bj.Spec.PluginSelector; s != "" {
		reqs, err = labels.ParseToRequirements(s)
		if err != nil {
			return nil, err
		}
		sel = sel.Add(reqs...)
	}
	return sel, nil
}

func SelectPlugin(plugins []api.InfoResponse, bj crd.BuildJob) (int, error) {
	sel, err := labelsSelector(bj)
	if err != nil {
		return -1, err
	}
	for idx, info := range plugins {
		lbls := labels.Set(info.Labels)
		if sel.Matches(lbls) {
			return idx, nil
		}
	}
	return -1, fmt.Errorf("no plugin can handle %s", bj.Name)
}
