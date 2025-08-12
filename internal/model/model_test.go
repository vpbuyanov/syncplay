package model

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestRoom_CreateRoom(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := NewMockstorePG(ctrl)
	r := NewModelRoom(mockStore)

	t.Run("success", func(t *testing.T) {
		mockStore.
			EXPECT().
			CreateRoomById(ctx, gomock.Any()).
			Return(nil)

		id, err := r.CreateRoom(ctx)
		require.NoError(t, err)
		require.NotEmpty(t, id)

		_, parseErr := uuid.Parse(id)
		require.NoError(t, parseErr)
	})

	t.Run("store error", func(t *testing.T) {
		mockStore.
			EXPECT().
			CreateRoomById(ctx, gomock.Any()).
			Return(errors.New("db failure"))

		id, err := r.CreateRoom(ctx)
		require.Empty(t, id)
		require.Error(t, err)
		require.True(t, strings.Contains(err.Error(), "CreateRoom model err"))
	})
}

func TestRoom_DeleteRoom(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := NewMockstorePG(ctrl)
	r := NewModelRoom(mockStore)

	rawID := uuid.NewString()
	apiID := openapi_types.UUID{}
	err := apiID.Scan(rawID)
	assert.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		mockStore.
			EXPECT().
			DeleteRoomById(ctx, rawID).
			Return(nil)

		err = r.DeleteRoom(ctx, apiID)
		require.NoError(t, err)
	})

	t.Run("store error", func(t *testing.T) {
		mockStore.
			EXPECT().
			DeleteRoomById(ctx, rawID).
			Return(errors.New("not found"))

		err = r.DeleteRoom(ctx, apiID)
		require.Error(t, err)
		require.True(t, strings.Contains(err.Error(), "DeleteRoom model err"))
	})
}

func TestRoom_RoomExistsUUID(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := NewMockstorePG(ctrl)
	r := NewModelRoom(mockStore)

	rawID := uuid.NewString()
	var apiID openapi_types.UUID
	require.NoError(t, apiID.Scan(rawID))

	t.Run("exists true", func(t *testing.T) {
		mockStore.
			EXPECT().
			RoomExists(ctx, rawID).
			Return(true, nil)

		ok, err := r.RoomExistsUUID(ctx, apiID)
		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("exists false", func(t *testing.T) {
		mockStore.
			EXPECT().
			RoomExists(ctx, rawID).
			Return(false, nil)

		ok, err := r.RoomExistsUUID(ctx, apiID)
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("store error", func(t *testing.T) {
		mockStore.
			EXPECT().
			RoomExists(ctx, rawID).
			Return(false, assert.AnError)

		ok, err := r.RoomExistsUUID(ctx, apiID)
		require.Error(t, err)
		assert.False(t, ok)
		assert.True(t, strings.Contains(err.Error(), "room id not found"))
	})
}
