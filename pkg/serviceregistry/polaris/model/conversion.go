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

package model

import (
	"fmt"
	"strings"

	"istio.io/pkg/log"

	"github.com/polarismesh/polaris-go/pkg/model"
	istio "istio.io/api/networking/v1alpha3"
	"istio.io/istio/pkg/config/protocol"
)

type PolarisInfo struct {
	PolarisService   string
	PolarisNamespace string
}

func replaceSpecialStr(s string) string {
	s = strings.ReplaceAll(s, "_", "-")
	s = strings.ReplaceAll(s, ":", "-")
	s = strings.ToLower(s)
	return s
}

func CovertServiceHostname(namespace string, name string) string {
	return fmt.Sprintf("%s.polaris-%s.polaris", replaceSpecialStr(namespace), replaceSpecialStr(name))
}

func CovertServiceName(namespace string, name string) string {
	return fmt.Sprintf("%s.polaris-%s", replaceSpecialStr(namespace), replaceSpecialStr(name))
}

func GetPolarisInfoFromSEAnnotations(annotations map[string]string) (polarisInfo *PolarisInfo, err error) {
	polarisService, exists := annotations["aeraki.net/polarisService"]
	if !exists {
		return nil, fmt.Errorf("polaris info should have [annotation]: aeraki.net/polarisService")
	}

	polarisNamespace, exists := annotations["aeraki.net/polarisNamespace"]
	if !exists {
		return nil, fmt.Errorf("polaris info should have [annotation]: aeraki.net/polarisNamespace")
	}

	return &PolarisInfo{
		PolarisService:   polarisService,
		PolarisNamespace: polarisNamespace,
	}, nil
}

func ConvertServiceEntry(rsp *model.InstancesResponse) (*istio.ServiceEntry, map[string]string) {
	log.Infof("[ConvertServiceEntry] starting covert serviceentry for polairs service: %v, namespace: %v", rsp.GetService(), rsp.GetNamespace())
	host := CovertServiceHostname(rsp.GetNamespace(), rsp.GetService())
	location := istio.ServiceEntry_MESH_INTERNAL
	resolution := istio.ServiceEntry_STATIC
	ports := make(map[uint32]*istio.Port)
	workloadEntries := make([]*istio.WorkloadEntry, 0)
	annotations := make(map[string]string)

	for _, instance := range rsp.Instances {
		log.Debugf("[ConvertServiceEntry] sync instance: [host]%v, [port]%v, [revision]%v [weight]%v [metadata]%v",
			instance.GetHost(), instance.GetPort(), instance.GetRevision(), instance.GetWeight(), instance.GetMetadata())
		port := convertPort(int(instance.GetPort()), instance.GetProtocol())

		if svcPort, exists := ports[port.Number]; exists && svcPort.Protocol != port.Protocol {
			log.Warnf("Service %v has two instances on same port %v but different protocols (%v, %v)",
				rsp.GetService(), port.Number, svcPort.Protocol, port.Protocol)
		} else {
			ports[port.Number] = port
		}

		workloadEntries = append(workloadEntries, convertWorkloadEntry(instance))
	}

	svcPorts := make([]*istio.Port, 0, len(ports))
	for _, port := range ports {
		svcPorts = append(svcPorts, port)
	}

	annotations["aeraki.net/polarisNamespace"] = rsp.GetNamespace()
	annotations["aeraki.net/polarisService"] = rsp.GetService()
	annotations["aeraki.net/revision"] = rsp.GetRevision()

	out := &istio.ServiceEntry{
		Hosts:      []string{host},
		Ports:      svcPorts,
		Location:   location,
		Resolution: resolution,
		Endpoints:  workloadEntries,
	}

	return out, annotations
}

func convertWorkloadEntry(instance model.Instance) *istio.WorkloadEntry {
	addr := instance.GetHost()
	port := convertPort(int(instance.GetPort()), instance.GetProtocol())

	return &istio.WorkloadEntry{
		Address: addr,
		Ports:   map[string]uint32{port.Name: port.Number},
		Weight:  uint32(instance.GetWeight()),
	}
}

func convertPort(port int, name string) *istio.Port {
	if name == "" {
		name = "tcp"
	}

	return &istio.Port{
		Number:     uint32(port),
		Protocol:   convertProtocol(name),
		Name:       name,
		TargetPort: uint32(port),
	}
}

func convertProtocol(name string) string {
	p := protocol.Parse(name)
	if p == protocol.Unsupported {
		log.Warnf("unsupported protocol value: %s", name)
		return string(protocol.TCP)
	}
	return string(p)
}
