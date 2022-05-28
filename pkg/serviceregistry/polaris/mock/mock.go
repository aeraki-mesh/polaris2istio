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

package polarismock

import (
	"fmt"
	"log"
	"net"

	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/google/uuid"
	"github.com/polarismesh/polaris-go/api" // must import this package for logger init
	"github.com/polarismesh/polaris-go/pkg/config"
	namingpb "github.com/polarismesh/polaris-go/pkg/model/pb/v1"
	"github.com/polarismesh/polaris-go/test/mock"
	"google.golang.org/grpc"
)

const (
	// 测试的默认命名空间
	defaultConsumerNamespace = "Testns"
	// 测试的默认服务名
	defaultConsumerService = "demo"
	// 测试服务器的默认地址
	defaultConsumerIPAddress = "127.0.0.1"
	// 测试服务器的端口
	defaultConsumerPort = 8008
)

const (
	// 直接过滤的实例数
	normalInstances    = 3
	isolatedInstances  = 2
	unhealthyInstances = 1
	allInstances       = normalInstances + isolatedInstances + unhealthyInstances
)

// PolarisMockServer 消费者API测试套
type PolarisMockServer struct {
	conn         *api.ConsumerAPI
	mockServer   mock.NamingServer
	grpcServer   *grpc.Server
	grpcListener net.Listener
	serviceToken string
	testService  *namingpb.Service
	testServices []*namingpb.Service
	serverURL    string
}

// 初始化mock数据
func (m *PolarisMockServer) initMockData() {
	// TODO: synchronize from configuration file
	m.testServices = []*namingpb.Service{
		{
			Name:      &wrappers.StringValue{Value: "polaris-1-2"},
			Namespace: &wrappers.StringValue{Value: "test"},
			Token:     &wrappers.StringValue{Value: m.serviceToken},
		},
		{
			Name:      &wrappers.StringValue{Value: "polaris-3-4"},
			Namespace: &wrappers.StringValue{Value: "test"},
			Token:     &wrappers.StringValue{Value: m.serviceToken},
		},
	}
}

// newServer
func (m *PolarisMockServer) NewServer() {
	grpcOptions := make([]grpc.ServerOption, 0)
	maxStreams := 100000
	grpcOptions = append(grpcOptions, grpc.MaxConcurrentStreams(uint32(maxStreams)))

	// get the grpc server wired up
	grpc.EnableTracing = true

	ipAddr := defaultConsumerIPAddress
	shopPort := defaultConsumerPort
	m.serverURL = fmt.Sprintf("%s:%d", ipAddr, shopPort)
	var err error
	m.grpcServer = grpc.NewServer(grpcOptions...)
	m.serviceToken = uuid.New().String()
	m.mockServer = mock.NewNamingServer()
	token := m.mockServer.RegisterServerService(config.ServerDiscoverService)
	m.mockServer.RegisterServerInstance(ipAddr, shopPort, config.ServerDiscoverService, token, true)
	m.mockServer.RegisterNamespace(&namingpb.Namespace{
		Name:    &wrappers.StringValue{Value: defaultConsumerNamespace},
		Comment: &wrappers.StringValue{Value: "for consumer api test"},
		Owners:  &wrappers.StringValue{Value: "ConsumerAPI"},
	})
	m.mockServer.RegisterServerServices(ipAddr, shopPort)
	m.testService = &namingpb.Service{
		Name:      &wrappers.StringValue{Value: defaultConsumerService},
		Namespace: &wrappers.StringValue{Value: defaultConsumerNamespace},
		Token:     &wrappers.StringValue{Value: m.serviceToken},
	}
	m.mockServer.RegisterService(m.testService)
	m.mockServer.GenTestInstances(m.testService, normalInstances)
	m.mockServer.GenInstancesWithStatus(m.testService, isolatedInstances, mock.IsolatedStatus, 2048)
	m.mockServer.GenInstancesWithStatus(m.testService, unhealthyInstances, mock.UnhealthyStatus, 4096)

	namingpb.RegisterPolarisGRPCServer(m.grpcServer, m.mockServer)
	m.grpcListener, err = net.Listen("tcp", fmt.Sprintf("%s:%d", ipAddr, shopPort))
	if err != nil {
		log.Fatalf("error listening appserver %v", err)
	}
	log.Printf("appserver listening on %s:%d\n", ipAddr, shopPort)
	go func() {
		m.grpcServer.Serve(m.grpcListener)
	}()
}

//stopServer 结束测试套程序
func (m *PolarisMockServer) StopServer() {
	log.Printf("Stopping server")
	m.grpcServer.Stop()
}

func (m *PolarisMockServer) GetGrpcServerURL() string {
	return m.grpcListener.Addr().String()
}

func RunMockServer(stop <-chan struct{}) {
	mockServer := PolarisMockServer{}
	mockServer.NewServer()
	defer mockServer.StopServer()
	<-stop
}

var GlobalPolarisMockServer = PolarisMockServer{}
