// Copyright Aeraki Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package polaris

import (
	"time"

	polaris "github.com/aeraki-framework/polaris2istio/pkg/serviceregistry/polaris/sdk"
	istioclient "istio.io/client-go/pkg/clientset/versioned"
	"istio.io/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

// ServiceWatcher watches for newly created polaris services and creates a providerWatcher for each service
type ServiceWatcher struct {
	polarisclient  *polaris.PolarisClient
	ic             *istioclient.Clientset
	polarisAddress string
	registryMethod uint
	configRootNS   string
}

// NewServiceWatcher creates a new service watcher
func NewServiceWatcher(polarisAddress string, registryMethod uint, configRootNS string) (*ServiceWatcher, error) {
	polarisclient, err := polaris.NewPolarisClient(polarisAddress)
	if err != nil {
		log.Errorf("failed to new polaris client consumer client: %v", err)
		return nil, err
	}

	ic, err := getIstioClient()
	if err != nil {
		log.Errorf("failed to create istio client: %v", err)
		return nil, err
	}

	return &ServiceWatcher{
		ic:             ic,
		polarisclient:  polarisclient,
		polarisAddress: polarisAddress,
		registryMethod: registryMethod,
		configRootNS:   configRootNS,
	}, nil
}

func getIstioClient() (*istioclient.Clientset, error) {
	config, err := config.GetConfig()
	if err != nil {
		return nil, err
	}

	ic, err := istioclient.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return ic, nil
}

// Run a time ticker for watch
func (w *ServiceWatcher) Run(stop <-chan struct{}) {
	tickTimer := time.NewTicker(10 * time.Second)
	w.watchProviders(stop)
	for {
		select {
		case <-tickTimer.C:
			log.Info("received time ticker")
			w.watchProviders(stop)
		case <-stop:
			log.Info("recieve stop chan,stoped")
			return
		}
	}
}

func (w *ServiceWatcher) watchProviders(stop <-chan struct{}) {
	providerWatcher := NewProviderWatcher(w.ic, w.polarisclient, w.configRootNS)
	log.Infof("start to scan the matched services for watch on polaris")
	go providerWatcher.Run(stop)
}
