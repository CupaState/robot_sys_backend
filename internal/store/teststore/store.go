package teststore

import psql_pb "gowallet/cmd/psg_worker/proto"

// Store...
type Store struct {
	userRepository *UserRepository
}

// New...
func New() *Store {
	return &Store{}
}

// User...
func (s *Store) User() UserRepository{
	if s.userRepository != nil {
		return *s.userRepository
	}

	s.userRepository = &UserRepository{
		store: s,
		users: make(map[int]*psql_pb.UserModel),
	}
	return *s.userRepository
}
