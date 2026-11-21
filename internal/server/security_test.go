package server

import (
	"testing"

	userv1 "apiPos/gen/user/v1"

	"github.com/stretchr/testify/require"
)

func TestToggleUserActiveRequiresUserUpdatePermission(t *testing.T) {
	require.Equal(t, "users.update", endpointPermissions()[userv1.UserService_ToggleUserActive_FullMethodName])
}
