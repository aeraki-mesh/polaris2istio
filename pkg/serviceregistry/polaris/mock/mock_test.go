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
	"testing"

	polarissdk "github.com/aeraki-mesh/polaris2istio/pkg/serviceregistry/polaris/sdk"
	"github.com/polarismesh/polaris-go/api"

	"github.com/stretchr/testify/assert"
)

func TestGetAllInstance(t *testing.T) {
	GlobalPolarisMockServer.NewServer()
	defer GlobalPolarisMockServer.StopServer()
	polarisclient, err := polarissdk.NewPolarisClient(GlobalPolarisMockServer.grpcListener.Addr().String())
	if err != nil {
		t.Errorf("NewPolarisClient failed: %v", err)
	}
	req := &api.GetAllInstancesRequest{}
	req.Namespace = "Polaris"
	req.Service = "polaris.discover"
	rsp, err := polarisclient.GetConn().GetAllInstances(req)
	if err != nil {
		t.Errorf("GetAllInstances failed: %v", err)
	}
	t.Logf("Polaris Service Instances: %+v", rsp.Instances[0].GetHost())
	assert.Equal(t, rsp.Instances[0].GetHost(), "127.0.0.1")
	assert.Equal(t, rsp.Instances[0].GetPort(), uint32(8008))
}
