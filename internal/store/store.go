package store

import (
	"database/sql"

	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

type Store struct {
	config *Config
	db *sql.DB
	userRepository *UserRepository
}

// Returns new store object
func New(config *Config) *Store {
	return &Store{
		config: config,
	}
}

// Opens database connection
func (s *Store) Open() error{
	db, err := sql.Open("postgres", s.config.DatabaseURL)
	if err != nil {
		return err
	}

	if err := db.Ping(); err != nil {
		return err
	}

	logrus.Info("Connected to database")

	s.db = db
	return nil
}

// Closes connection with DB
func (s *Store) Close() {
	s.db.Close()
}

// Returns the User Repository object
func (s *Store) GetUserRepository() *UserRepository {
	if s.userRepository != nil {
		return s.userRepository
	}

	s.userRepository = &UserRepository{
		store: s,
	}

	return s.userRepository
}
