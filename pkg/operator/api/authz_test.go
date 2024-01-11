/*
Copyright 2021 The Dapr Authors
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

package api

import (
	"testing"

	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/dapr/dapr/tests/util"
	"github.com/dapr/dapr/utils"
)

func Test_authzRequest(t *testing.T) {
	appID := spiffeid.RequireFromString("spiffe://example.org/ns/ns1/app1")
	serverID := spiffeid.RequireFromString("spiffe://example.org/ns/dapr-system/dapr-operator")
	pki := util.GenPKI(t, util.PKIOptions{LeafID: serverID, ClientID: appID})
	spiffeID, _ := utils.GetSpiffeIDFromContext(pki.ClientGRPCCtx(t))

	t.Run("different namespace should error", func(t *testing.T) {
		err := new(apiServer).authzRequest(spiffeID, "ns2")
		require.Error(t, err)
		assert.Equal(t, codes.PermissionDenied, status.Code(err))
	})

	t.Run("empty namespace should error", func(t *testing.T) {
		err := new(apiServer).authzRequest(spiffeID, "")
		require.Error(t, err)
		assert.Equal(t, codes.PermissionDenied, status.Code(err))
	})

	t.Run("invalid SPIFFE path should error", func(t *testing.T) {
		appID := spiffeid.RequireFromString("spiffe://example.org/foo/bar")
		pki2 := util.GenPKI(t, util.PKIOptions{LeafID: serverID, ClientID: appID})
		spiffeID2, _ := utils.GetSpiffeIDFromContext(pki2.ClientGRPCCtx(t))
		err := new(apiServer).authzRequest(spiffeID2, "ns1")
		require.Error(t, err)
		assert.Equal(t, codes.PermissionDenied, status.Code(err))
	})

	t.Run("same namespace should no error", func(t *testing.T) {
		err := new(apiServer).authzRequest(spiffeID, "ns1")
		require.NoError(t, err)
	})
}
