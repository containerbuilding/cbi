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

package pluginselector

import (
	"context"
	"fmt"
	"strings"

	crd "github.com/containerbuilding/cbi/pkg/apis/cbi/v1alpha1"
	api "github.com/containerbuilding/cbi/pkg/plugin/api"
	"google.golang.org/grpc"
)

type PluginSelectorFunc func(plugins []api.InfoResponse, bj crd.BuildJob) int

func NewPluginSelector(fn PluginSelectorFunc, conns ...*grpc.ClientConn) *PluginSelector {
	ps := &PluginSelector{
		fn:         fn,
		cachedInfo: make(map[*grpc.ClientConn]*api.InfoResponse, len(conns)),
	}
	for _, conn := range conns {
		ps.cachedInfo[conn] = nil
	}
	return ps
}

type PluginSelector struct {
	fn         PluginSelectorFunc
	cachedInfo map[*grpc.ClientConn]*api.InfoResponse
}

func (ps *PluginSelector) UpdateCachedInfo(ctx context.Context) error {
	var errors []error
	for conn := range ps.cachedInfo {
		client := api.NewPluginClient(conn)
		info, err := client.Info(ctx, &api.InfoRequest{})
		if err != nil {
			errors = append(errors, err)
			info = nil
		}
		ps.cachedInfo[conn] = info
	}
	if len(errors) > 0 {
		return fmt.Errorf("%v", errors)
	}
	return nil
}

func (ps *PluginSelector) Select(bj crd.BuildJob) api.PluginClient {
	var (
		conns []*grpc.ClientConn
		info  []api.InfoResponse
	)

	for c, i := range ps.cachedInfo {
		if i != nil {
			conns = append(conns, c)
			info = append(info, *i)
		}
	}
	idx := ps.fn(info, bj)
	if idx >= 0 {
		conn := conns[idx]
		return api.NewPluginClient(conn)
	}
	return nil
}

func GenericPluginSelectorFunc(plugins []api.InfoResponse, bj crd.BuildJob) int {
	var supported []int
	for idx, info := range plugins {
		languageKindSupported := false
		contextKindSupported := false
		for _, l := range info.SupportedLanguageKind {
			languageKindSupported = strings.EqualFold(l, bj.Spec.Language.Kind)
			if languageKindSupported {
				break
			}
		}
		for _, c := range info.SupportedContextKind {
			contextKindSupported = strings.EqualFold(c, bj.Spec.Context.Kind)
			if contextKindSupported {
				break
			}
		}
		if languageKindSupported && contextKindSupported {
			supported = append(supported, idx)
		}
	}
	if len(supported) > 0 {
		// TODO: shuffle?
		return supported[0]
	}
	return -1
}
