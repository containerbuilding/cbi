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

package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net"

	crd "github.com/containerbuilding/cbi/pkg/apis/cbi/v1alpha1"
	api "github.com/containerbuilding/cbi/pkg/plugin/api"
	"github.com/containerbuilding/cbi/pkg/plugin/base/backend"
	"google.golang.org/grpc"
	"k8s.io/client-go/kubernetes"
)

type Service struct {
	Backend       backend.Backend
	KubeClientset *kubernetes.Clientset
	Port          int
}

func (s *Service) Serve() error {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", s.Port))
	if err != nil {
		return err
	}
	ps := &PluginServer{
		s: s,
	}
	gs := grpc.NewServer()
	api.RegisterPluginServer(gs, ps)
	return gs.Serve(ln)
}

type PluginServer struct {
	s *Service
}

func (ps *PluginServer) Info(ctx context.Context, req *api.InfoRequest) (*api.InfoResponse, error) {
	return ps.s.Backend.Info(ctx, req)
}

func (ps *PluginServer) Build(ctx context.Context, req *api.BuildRequest) (*api.BuildResponse, error) {
	var buildJob crd.BuildJob
	if err := json.Unmarshal([]byte(req.BuildJobJson), &buildJob); err != nil {
		return nil, err
	}
	jn := jobName(buildJob)
	jobManifest, err := ps.s.Backend.CreateJobManifest(ctx, jn, buildJob)
	if err != nil {
		return nil, err
	}
	// FIXME(AkihiroSuda): update the job if already exists
	_, err = ps.s.KubeClientset.BatchV1().Jobs(buildJob.Namespace).Create(jobManifest)
	if err != nil {
		return nil, err
	}
	res := &api.BuildResponse{
		JobName: jn,
	}
	return res, nil
}

func jobName(buildJob crd.BuildJob) string {
	return buildJob.Name + "-job"
}
