package teststore

import (
	psql_pb "gowallet/cmd/psg_worker/proto"
	"gowallet/internal/model"

	"gowallet/internal/store"
)

// UserRepository ...
type UserRepository struct {
	store *Store
	users map[int]*psql_pb.UserModel
}

// Create ...
func (r *UserRepository) Create(u *psql_pb.UserModel) error {
	if err := model.Validate(u); err != nil {
		return err
	}

	return nil
}

// FindByEmail ...
func (r *UserRepository) FindByEmail(email string) (*psql_pb.UserModel, error) {
	for _, u := range r.users {
		if u.Email == email {
			return u, nil
		}
	}

	return nil, store.ErrRecordNotFound
}