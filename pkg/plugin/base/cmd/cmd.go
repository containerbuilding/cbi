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

package cmd

import (
	"flag"
	"fmt"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	// Uncomment the following line to load the gcp plugin (only required to authenticate against GKE clusters).
	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	"github.com/containerbuilding/cbi/pkg/plugin"
	"github.com/containerbuilding/cbi/pkg/plugin/base/backend"
	"github.com/containerbuilding/cbi/pkg/plugin/base/service"
)

type Opts struct {
	FlagSet *flag.FlagSet
	Args    []string
	Backend backend.Backend
}

func Main(o Opts) error {
	var (
		masterURL  string
		kubeconfig string
		port       int
	)
	o.FlagSet.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	o.FlagSet.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	o.FlagSet.IntVar(&port, "cbi-plugin-port", plugin.DefaultPort, "Port for listening CBI Plugin gRPC API")
	if err := o.FlagSet.Parse(o.Args); err != nil {
		return err
	}

	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		return fmt.Errorf("Error building kubeconfig: %s", err.Error())
	}

	kubeClientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return fmt.Errorf("Error building kubernetes clientset: %s", err.Error())
	}

	s := &service.Service{
		Backend:       o.Backend,
		KubeClientset: kubeClientset,
		Port:          port,
	}
	if err := s.Serve(); err != nil {
		return fmt.Errorf("Error serving CBI plugin API: %s", err.Error())
	}
	return nil
}
