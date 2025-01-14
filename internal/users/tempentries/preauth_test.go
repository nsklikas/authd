package tempentries

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/ubuntu/authd/internal/testutils/golden"
	"github.com/ubuntu/authd/internal/users/idgenerator"
	"github.com/ubuntu/authd/internal/users/types"
)

func TestPreAuthUser(t *testing.T) {
	t.Parallel()

	loginName := "test"
	uidToGenerate := uint32(12345)

	tests := map[string]struct {
		maxUsers       bool
		uidsToGenerate []uint32
		registerTwice  bool

		wantErr bool
	}{
		"Successfully register a pre-auth user": {},
		"Successfully register a pre-auth user if the first generated UID is already in use": {
			uidsToGenerate: []uint32{0, uidToGenerate}, // UID 0 (root) always exists
		},
		"No error when registering a pre-auth user with the same name": {registerTwice: true},

		"Error when maximum number of pre-auth users is reached": {maxUsers: true, wantErr: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if tc.uidsToGenerate == nil {
				tc.uidsToGenerate = []uint32{uidToGenerate}
			}

			idGeneratorMock := &idgenerator.IDGeneratorMock{UIDsToGenerate: tc.uidsToGenerate}
			records := newPreAuthUserRecords(idGeneratorMock)

			if tc.maxUsers {
				records.numUsers = MaxPreAuthUsers
			}

			uid, err := records.RegisterPreAuthUser(loginName)
			if tc.wantErr {
				require.Error(t, err, "RegisterPreAuthUser should return an error, but did not")
				return
			}
			require.NoError(t, err, "RegisterPreAuthUser should not return an error, but did")
			require.Equal(t, uidToGenerate, uid, "UID should be the one generated by the IDGenerator")
			require.Equal(t, records.numUsers, 1, "Number of pre-auth users should be 1")

			if tc.registerTwice {
				uid, err = records.RegisterPreAuthUser(loginName)
				require.NoError(t, err, "RegisterPreAuthUser should not return an error, but did")
				require.Equal(t, uidToGenerate, uid, "UID should be the one generated by the IDGenerator")
				require.Equal(t, records.numUsers, 1, "Number of pre-auth users should be 1")
			}

			// Check that the user was registered
			user, err := records.userByLogin(loginName)
			require.NoError(t, err, "UserByID should not return an error, but did")
			checkPreAuthUser(t, user)

			// Remove the user
			records.deletePreAuthUser(uidToGenerate)
			require.Equal(t, records.numUsers, 0, "Number of pre-auth users should be 0")

			// Check that the user was removed
			_, err = records.userByLogin(loginName)
			require.Error(t, err, "UserByID should return an error, but did not")
		})
	}
}

func TestPreAuthUserByIDAndName(t *testing.T) {
	t.Parallel()

	loginName := "test"
	uidToGenerate := uint32(12345)

	tests := map[string]struct {
		registerUser       bool
		userAlreadyRemoved bool

		wantErr bool
	}{
		"Successfully get a user by ID and name": {registerUser: true},

		"Error when user is not registered":  {wantErr: true},
		"Error when user is already removed": {registerUser: true, userAlreadyRemoved: true, wantErr: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			idGeneratorMock := &idgenerator.IDGeneratorMock{UIDsToGenerate: []uint32{uidToGenerate}}
			records := newPreAuthUserRecords(idGeneratorMock)

			if tc.registerUser {
				uid, err := records.RegisterPreAuthUser(loginName)
				require.NoError(t, err, "RegisterPreAuthUser should not return an error, but did")
				require.Equal(t, uidToGenerate, uid, "UID should be the one generated by the IDGenerator")
			}

			if tc.userAlreadyRemoved {
				records.deletePreAuthUser(uidToGenerate)
			} else {
				defer records.deletePreAuthUser(uidToGenerate)
			}

			user, err := records.userByID(uidToGenerate)

			if tc.wantErr {
				require.Error(t, err, "UserByID should return an error, but did not")
				return
			}
			require.NoError(t, err, "UserByID should not return an error, but did")
			checkPreAuthUser(t, user)

			user, err = records.userByName(user.Name)
			if tc.wantErr {
				require.Error(t, err, "UserByName should return an error, but did not")
				return
			}
			require.NoError(t, err, "UserByName should not return an error, but did")
			checkPreAuthUser(t, user)
		})
	}
}

func checkPreAuthUser(t *testing.T, user types.UserEntry) {
	t.Helper()

	// The name field is randomly generated, so unset it before comparing the user with the golden file.
	require.NotEmpty(t, user.Name, "Name should not be empty")
	user.Name = ""

	golden.CheckOrUpdateYAML(t, user)
}