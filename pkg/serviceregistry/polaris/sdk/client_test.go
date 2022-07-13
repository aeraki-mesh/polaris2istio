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
	"testing"

	mock "github.com/aeraki-mesh/polaris2istio/pkg/serviceregistry/polaris/mock"
	"github.com/polarismesh/polaris-go/api"
)

func TestClient(t *testing.T) {
	mock.GlobalPolarisMockServer.NewServer()
	defer mock.GlobalPolarisMockServer.StopServer()
	// assert := assert.New(t)
	var tests = []struct {
		polarisNamespace string
		polarisService   string
		err              error
	}{
		{"Testns", "demo", nil},
	}
	polarisclient, err := NewPolarisClient(mock.GlobalPolarisMockServer.GetGrpcServerURL())
	if err != nil {
		t.Errorf("failed to new polaris client consumer client: %v", err)
	}

	for _, test := range tests {
		req := &api.GetAllInstancesRequest{}
		req.Namespace = test.polarisNamespace
		req.Service = test.polarisService
		_, err := polarisclient.GetConn().GetAllInstances(req)
		if err != nil {
			t.Errorf("GetAllInstances failed: %v", err)
		}
		// assert.Equal(t, err, test.err)
	}

}
