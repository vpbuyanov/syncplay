package model

import (
	"context"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/pkg/errors"
)

//go:generate mockgen -source=model.go -destination model_mock.go -package model MODEL
type storePG interface {
	CreateRoomById(ctx context.Context, id string) error
	DeleteRoomById(ctx context.Context, id string) error
}

type Room struct {
	storePG
}

func NewModelRoom(s storePG) *Room {
	return &Room{
		s,
	}
}

func (r *Room) CreateRoom(ctx context.Context) (string, error) {
	id := uuid.NewString()

	err := r.CreateRoomById(ctx, id)
	if err != nil {
		return "", errors.Wrap(err, "CreateRoom model err")
	}

	return id, nil
}

func (r *Room) DeleteRoom(ctx context.Context, id openapi_types.UUID) error {
	strID := id.String()

	err := r.DeleteRoomById(ctx, strID)
	if err != nil {
		return errors.Wrap(err, "DeleteRoom model err")
	}

	return nil
}
