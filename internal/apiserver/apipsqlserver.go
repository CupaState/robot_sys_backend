package apiserver

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	pb_psql "gowallet/cmd/psg_worker/proto"
	"gowallet/internal/store"

	"github.com/oklog/run"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

// Postgres server struct
type ApiPSQLServer struct {
	pb_psql.UnimplementedApiPSQLServerServer
	config *PSQLConfig
	logger *logrus.Logger
	store  *store.Store
	server *grpc.Server
}

// New ApiSQLServer ...
func NewPSQLServer(config *PSQLConfig) *ApiPSQLServer {
	return &ApiPSQLServer{
		config: config,
		logger: logrus.New(),
		store: store.New(config.Store),
		server: grpc.NewServer(),
	}
}

// Start...
func (s *ApiPSQLServer) Start() {
	if err := s.init(); err != nil {
		log.Fatalln(err)
	}
	s.logger.Info("Server initialized...")
	pb_psql.RegisterApiPSQLServerServer(s.server, s)

	var g run.Group
	{
		listen, err := net.Listen("tcp", fmt.Sprintf(":%d", s.config.BindAddr))
		if err != nil {
			s.logger.Error(err)
		}
		g.Add(func() error{
			s.logger.Info("Server listening at port: ", listen.Addr())
			return s.server.Serve(listen)
		}, func(error) {
			listen.Close()
		})
	}
	{
		cancelInterrupt := make(chan struct{})
		g.Add(func() error {
			ch := make(chan os.Signal, 1)
			signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
			select {
			case sig := <-ch:
				s.logger.Info(sig)
				return fmt.Errorf("received signal %v", sig)
			case <-cancelInterrupt:
				return nil
			}
		}, func (err error){
			defer close(cancelInterrupt)
			s.logger.Error(err)

			s.logger.Exit(0)
			s.server.GracefulStop()
		})
	}

	if err := g.Run(); err != nil {
		s.logger.Error(err)
	}
}

func (s *ApiPSQLServer) init() error {
	if err := s.configurePSQLLogger(); err != nil {
		return err
	}

	if err := s.configureStore(); err != nil {
		return err
	}

	return nil
}

// ============= API METHODS
// CreateUser...
func (s *ApiPSQLServer) CreateUser(
	ctx context.Context,
	in *pb_psql.UserCreateRequest,
) (*pb_psql.UserCreateResponse, error) {
	r := s.store.GetUserRepository()
	u := in.GetU()
	
	err := r.Create(u)
	if err != nil {
		return &pb_psql.UserCreateResponse{}, err
	}

	s.logger.Info("User with UserID ", u.UserId, " has been created")


	return &pb_psql.UserCreateResponse{}, nil
}

// FindUserByEmail...
func (s *ApiPSQLServer) FindUserByEmail(
	ctx context.Context,
	in *pb_psql.FindUserByEmailRequest,
) (*pb_psql.FindUserByEmailResponse, error) {
	r := s.store.GetUserRepository()

	u, err := r.FindByEmail(in.GetEmail())
	if err != nil {
		return &pb_psql.FindUserByEmailResponse{}, err
	}

	return &pb_psql.FindUserByEmailResponse{U: u}, nil
}
// ============= API METHODS

// ============= CONFIGURATION METHODS
// Configurates logger with config
func (s *ApiPSQLServer) configurePSQLLogger() error {
	level, err := logrus.ParseLevel(s.config.LogLevel)
	if err != nil {
		return err
	}

	s.logger.SetLevel(level)
	return nil
}

// Configurates store with config
// configureStore...
func (s *ApiPSQLServer) configureStore() error {
	st := store.New(s.config.Store)

	if err := st.Open(); err != nil {
		return err
	}

	s.store = st
	return nil
}
