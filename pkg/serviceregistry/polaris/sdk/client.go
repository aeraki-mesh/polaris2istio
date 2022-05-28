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

	"github.com/polarismesh/polaris-go/api"
	"github.com/polarismesh/polaris-go/pkg/config"
	"github.com/polarismesh/polaris-go/pkg/model"
	"k8s.io/klog"
)

const (
	defaultProtocol       = "grpc"
	defaultConnectTimeout = time.Second * 5
)

type PolarisClient struct {
	conn       api.ConsumerAPI
	polarisMap *sync.Map
}

type syncSECallBack func(string, string)

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

func (c *PolarisClient) GetConn() api.ConsumerAPI {
	return c.conn
}

// get all instances with the given name and namespace
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

// watch polaris service
func (c *PolarisClient) WatchPolarisService(polarisNamespace string, polarisService string, cb syncSECallBack, force bool, stop <-chan struct{}) error {
	polarisName := getName(polarisNamespace, polarisService)
	_, exists := c.polarisMap.Load(polarisName)
	if !exists {
		c.polarisMap.Store(polarisName, 1)
	}
	if !force && exists {
		klog.Infof("[WatchPolarisService] already exist polaris service: %v, %v", polarisNamespace, polarisService)
		return nil
	}

	req := &api.WatchServiceRequest{WatchServiceRequest: model.WatchServiceRequest{
		Key: model.ServiceKey{
			Namespace: polarisNamespace,
			Service:   polarisService,
		}}}

	rsp, err := c.conn.WatchService(req)
	if err != nil {
		klog.Errorf("[WatchPolarisService] watch polaris service failed, err: %v", err.Error())
		return err
	}

	cb(polarisNamespace, polarisService)
	go c.waitForEvents(rsp.EventChannel, polarisNamespace, polarisService, cb, stop)
	return nil
}

func (c *PolarisClient) waitForEvents(ch <-chan model.SubScribeEvent, polarisNamespace string, polarisService string, cb syncSECallBack, stop <-chan struct{}) {
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
			c.dealEvent(insEvent, polarisNamespace, polarisService, cb)
		}
	}
}

func (c *PolarisClient) dealEvent(ins *model.InstanceEvent, polarisNamespace string, polarisService string, cb syncSECallBack) {
	if ins.AddEvent != nil {
		c.dealAddEvent(polarisNamespace, polarisService, cb)
	}
	if ins.UpdateEvent != nil {
		c.dealUpdateEvent(polarisNamespace, polarisService, cb)
	}
	if ins.DeleteEvent != nil {
		c.dealDeleteEvent(polarisNamespace, polarisService, cb)
	}
}

func (c *PolarisClient) dealAddEvent(polarisNamespace string, polarisService string, cb syncSECallBack) {
	klog.Infof("dealAddEvent %s %s", polarisNamespace, polarisService)
	cb(polarisNamespace, polarisService)
}

func (c *PolarisClient) dealUpdateEvent(polarisNamespace string, polarisService string, cb syncSECallBack) {
	klog.Infof("dealUpdateEvent %s %s", polarisNamespace, polarisService)
	cb(polarisNamespace, polarisService)
}

func (c *PolarisClient) dealDeleteEvent(polarisNamespace string, polarisService string, cb syncSECallBack) {
	klog.Infof("dealDeleteEvent %s %s", polarisNamespace, polarisService)
	cb(polarisNamespace, polarisService)
}
