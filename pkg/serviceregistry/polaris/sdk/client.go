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

package polarisclient

import (
	"fmt"
	"sync"
	"time"

	registryModel "github.com/aeraki-framework/polaris2istio/pkg/serviceregistry/polaris/model"
	"github.com/polarismesh/polaris-go/api"
	"github.com/polarismesh/polaris-go/pkg/config"
	"github.com/polarismesh/polaris-go/pkg/model"
	"k8s.io/klog"
)

const (
	defaultProtocol       = "grpc"
	defaultConnectTimeout = time.Second * 5
)

// PolarisClient is a client for interacting with the polaris
type PolarisClient struct {
	conn       api.ConsumerAPI
	polarisMap *sync.Map
}

type syncSECallBack func(polarisInfo *registryModel.PolarisInfo)

// NewPolarisClient creates a new client for the polaris
func NewPolarisClient(polarisAddress string) (*PolarisClient, error) {
	cf := config.NewDefaultConfiguration([]string{polarisAddress})
	cf.Global.ServerConnector.Protocol = defaultProtocol
	cf.Global.ServerConnector.ConnectTimeout = model.ToDurationPtr(defaultConnectTimeout)
	conn, err := api.NewConsumerAPIByConfig(cf)
	if err != nil {
		return nil, err
	}

	return &PolarisClient{
		conn:       conn,
		polarisMap: new(sync.Map),
	}, nil
}

// GetConn get the connection of the client
func (c *PolarisClient) GetConn() api.ConsumerAPI {
	return c.conn
}

// GetPolarisAllInstances get all instances with the given name and namespace
func (c *PolarisClient) GetPolarisAllInstances(namespace string, service string) (*model.InstancesResponse, error) {
	req := &api.GetAllInstancesRequest{}
	req.Namespace = namespace
	req.Service = service
	rsp, err := c.GetConn().GetAllInstances(req)
	if err != nil {
		return nil, err
	}
	return rsp, nil
}

func getName(namespace, serviceName string) string {
	return fmt.Sprintf("%s.%s", namespace, serviceName)
}

// WatchPolarisService watch polaris services
func (c *PolarisClient) WatchPolarisService(polarisInfo *registryModel.PolarisInfo, cb syncSECallBack, force bool,
	stop <-chan struct{}) error {
	polarisName := getName(polarisInfo.PolarisNamespace, polarisInfo.PolarisService)
	_, exists := c.polarisMap.Load(polarisName)
	if !exists {
		c.polarisMap.Store(polarisName, 1)
	}
	if !force && exists {
		klog.Infof("[WatchPolarisService] already exist polaris service: %v, %v",
			polarisInfo.PolarisNamespace, polarisInfo.PolarisService)
		return nil
	}

	req := &api.WatchServiceRequest{WatchServiceRequest: model.WatchServiceRequest{
		Key: model.ServiceKey{
			Namespace: polarisInfo.PolarisNamespace,
			Service:   polarisInfo.PolarisService,
		}}}

	rsp, err := c.conn.WatchService(req)
	if err != nil {
		klog.Errorf("[WatchPolarisService] watch polaris service failed, err: %v", err.Error())
		return err
	}

	cb(polarisInfo)
	go c.waitForEvents(rsp.EventChannel, polarisInfo, cb, stop)
	return nil
}

func (c *PolarisClient) waitForEvents(ch <-chan model.SubScribeEvent, polarisInfo *registryModel.PolarisInfo,
	cb syncSECallBack, stop <-chan struct{}) {
	for {
		select {
		case <-stop:
			klog.Info("[waitForEvents] stopping")
			return
		default:
			e := <-ch
			if e == nil {
				klog.Error("[waitForEvents] has nothing to do, event is nil")
				return
			}
			eType := e.GetSubScribeEventType()
			if eType != api.EventInstance {
				klog.Errorf("[waitForEvents] has nothing to do, event type is not EventInstance, event type: %v", eType)
				return
			}
			insEvent := e.(*model.InstanceEvent)
			c.dealEvent(insEvent, polarisInfo, cb)
		}
	}
}

func (c *PolarisClient) dealEvent(ins *model.InstanceEvent, polarisInfo *registryModel.PolarisInfo, cb syncSECallBack) {
	if ins.AddEvent != nil {
		c.dealAddEvent(polarisInfo, cb)
	}
	if ins.UpdateEvent != nil {
		c.dealUpdateEvent(polarisInfo, cb)
	}
	if ins.DeleteEvent != nil {
		c.dealDeleteEvent(polarisInfo, cb)
	}
}

func (c *PolarisClient) dealAddEvent(polarisInfo *registryModel.PolarisInfo, cb syncSECallBack) {
	klog.Infof("dealAddEvent polarisInfo %v", polarisInfo)
	cb(polarisInfo)
}

func (c *PolarisClient) dealUpdateEvent(polarisInfo *registryModel.PolarisInfo, cb syncSECallBack) {
	klog.Infof("dealUpdateEvent polarisInfo %v", polarisInfo)
	cb(polarisInfo)
}

func (c *PolarisClient) dealDeleteEvent(polarisInfo *registryModel.PolarisInfo, cb syncSECallBack) {
	klog.Infof("dealDeleteEvent polarisInfo %v", polarisInfo)
	cb(polarisInfo)
}
