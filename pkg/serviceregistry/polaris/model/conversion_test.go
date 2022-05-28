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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetPolarisInfoFromSEAnnotations(t *testing.T) {
	assert := assert.New(t)
	var tests = []struct {
		annotations map[string]string
		polarisInfo *PolarisInfo
		err         error
	}{
		{map[string]string{
			"aeraki.net/polarisNamespace": "test",
			"aeraki.net/polarisService":   "rating",
		},
			&PolarisInfo{
				PolarisService:   "rating",
				PolarisNamespace: "test",
			}, nil,
		},
		{map[string]string{
			"aeraki.net/polarisService": "rating",
		},
			nil, fmt.Errorf("polaris info should have [annotation]: aeraki.net/polarisNamespace"),
		},
		{map[string]string{
			"aeraki.net/polarisNamespace": "test",
		},
			nil, fmt.Errorf("polaris info should have [annotation]: aeraki.net/polarisService"),
		},
	}
	for _, test := range tests {
		polarisInfo, err := GetPolarisInfoFromSEAnnotations(test.annotations)
		assert.Equal(polarisInfo, test.polarisInfo)
		assert.Equal(err, test.err)
	}
}
