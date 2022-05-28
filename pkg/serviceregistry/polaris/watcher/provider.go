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
	"context"

	istio "istio.io/api/networking/v1alpha3"
	"istio.io/client-go/pkg/apis/networking/v1alpha3"

	// "istio.io/client-go/pkg/apis/networking/v1beta1"
	"github.com/aeraki-framework/polaris2istio/pkg/serviceregistry/polaris/model"
	polaris "github.com/aeraki-framework/polaris2istio/pkg/serviceregistry/polaris/sdk"
	istioclient "istio.io/client-go/pkg/clientset/versioned"

	"istio.io/pkg/log"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

const (
	// aerakiFieldManager is the FileldManager for Aeraki CRDs
	aerakiFieldManager = "aeraki"
)

type ProviderWatcher struct {
	polarisclient *polaris.PolarisClient
	ic            *istioclient.Clientset
	configRootNS  string
}

// NewWatcher creates a ProviderWatcher
func NewProviderWatcher(ic *istioclient.Clientset, polarisclient *polaris.PolarisClient, configRootNS string) *ProviderWatcher {
	return &ProviderWatcher{
		polarisclient: polarisclient,
		ic:            ic,
		configRootNS:  configRootNS,
	}
}

// scan services, then  registered or update polaris services to the istio mesh
func (w *ProviderWatcher) Run(stop <-chan struct{}) {
	seList, err := w.getServiceEntryList()
	if err != nil {
		log.Errorf("Error getting service entry list: %v", err)
	}

	for _, se := range seList.Items {
		log.Debugf("ServiceEntry [name]: %v [namespace]: %v [hosts]: %v, [endpoints]: %s", se.Name, se.Namespace, se.Spec.Hosts, se.Spec.Endpoints)
		polarisInfo, err := model.GetPolarisInfoFromSEAnnotations(se.GetAnnotations())
		if err != nil {
			log.Errorf("Error get ServiceEntry's annotations: %v", err)
			continue
		}

		_, existsRevision := se.GetAnnotations()["aeraki.net/revision"]

		if err := w.polarisclient.WatchPolarisService(polarisInfo.PolarisNamespace, polarisInfo.PolarisService, w.syncPolarisServices2Istio, !existsRevision, stop); err != nil {
			log.Errorf("Watch polaris %v failed, error: %v", polarisInfo, err)
			continue
		}
	}
}

func (w *ProviderWatcher) getServiceEntryList() (*v1alpha3.ServiceEntryList, error) {
	services, err := w.ic.NetworkingV1alpha3().ServiceEntries(w.configRootNS).List(context.TODO(), v1.ListOptions{
		LabelSelector: "manager=" + aerakiFieldManager + ", registry=polaris",
	})

	if err != nil {
		log.Errorf("Error list services entry: %v", err)
		return nil, err
	}

	return services, nil
}

func (w *ProviderWatcher) syncPolarisServices2Istio(polarisNamespace string, polarisService string) {
	klog.Infof("[syncPolarisServices2Istio] [polarisNamespace]%s [polarisService]%s", polarisNamespace, polarisService)
	rsp, err := w.polarisclient.GetPolarisAllInstances(polarisNamespace, polarisService)
	if err != nil {
		klog.Errorf("[syncPolarisServices2Istio] query polaris services' instances failed, err: %v", err.Error())
		return
	}

	newServiceEntry, newAnnotations := model.ConvertServiceEntry(rsp)
	if newServiceEntry == nil {
		klog.Errorf("convertServiceEntry failed?")
		return
	}

	oldServiceEntry, err := w.ic.NetworkingV1alpha3().ServiceEntries(w.configRootNS).Get(context.TODO(), model.CovertServiceName(polarisNamespace, polarisService), v1.GetOptions{})
	if err != nil {
		klog.Infof("[syncPolarisServices2Istio] get old service entries failed, error: %v", err)
		return
	}

	if revision, exists := oldServiceEntry.GetAnnotations()["aeraki.net/revision"]; !exists || newAnnotations["aeraki.net/revision"] != revision {
		klog.Infof("[syncPolarisServices2Istio] update serviceentry: %v", newServiceEntry)
		_, err = w.ic.NetworkingV1alpha3().ServiceEntries(oldServiceEntry.Namespace).Update(context.TODO(),
			w.toServiceEntryCRD(model.CovertServiceName(polarisNamespace, polarisService), newServiceEntry, oldServiceEntry, newAnnotations),
			v1.UpdateOptions{FieldManager: aerakiFieldManager})
		if err != nil {
			klog.Errorf("failed to update ServiceEntry: %s", err.Error())
		}
	} else {
		log.Infof("[syncPolarisServices2Istio] serviceentry unchanged: %v", oldServiceEntry.GetName())
	}
}

func (w *ProviderWatcher) toServiceEntryCRD(name string, new *istio.ServiceEntry, old *v1alpha3.ServiceEntry, annotations map[string]string) *v1alpha3.ServiceEntry {
	serviceEntry := &v1alpha3.ServiceEntry{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: w.configRootNS,
			Labels: map[string]string{
				"manager":  aerakiFieldManager,
				"registry": "polaris",
			},
			Annotations: annotations,
		},
		Spec: *new,
	}

	if old != nil {
		serviceEntry.ResourceVersion = old.ResourceVersion
	}

	return serviceEntry
}
