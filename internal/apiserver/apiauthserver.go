package apiserver

import (
	"context"
	"fmt"
	apiauthserver "gowallet/cmd/authenticator/proto"
	apiwalletcreatorserver "gowallet/cmd/walletcreator/proto"
	"gowallet/internal/model"
	"net"
	"os"
	"os/signal"
	"syscall"

	apipsqlserver "gowallet/cmd/psg_worker/proto"
	apivaultserver "gowallet/cmd/vaultworker/proto"

	"github.com/oklog/run"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ApiAuthServer...
type ApiAuthServer struct {
	apiauthserver.UnimplementedApiAuthServerServer
	logger *logrus.Logger
	config *AuthConfig
	server *grpc.Server

	// clients:
	vaultClient apivaultserver.VaultWorkerClient
	walletCreatorClient apiwalletcreatorserver.WalletCreatorClient
	psqlClient apipsqlserver.ApiPSQLServerClient

	// connections:
	vaultConn *grpc.ClientConn
	walletCreatorConn *grpc.ClientConn
	psqlConn *grpc.ClientConn
}

// NewApiAuthServer...
func NewApiAuthServer(config *AuthConfig) *ApiAuthServer{
	return &ApiAuthServer{
		logger: logrus.New(),
		config: config,
		server: grpc.NewServer(),
	}
}

// === PUBLIC METHODS
// Starts server
func (s *ApiAuthServer) Start() {
	if err := s.configureAuthLogger(); err != nil {
		s.logger.Error(err)
	}

	if err := s.initVaultClient(); err != nil {
		s.logger.Error(err)
	}
	s.logger.Info("VaultWorker Client is initialized")
	
	if err := s.initWalletCreatorClient(); err != nil {
		s.logger.Error(err)
	}
	s.logger.Info("WalletCreator Client is initialized")
	
	if err := s.initPSQLClient(); err != nil {
		s.logger.Error(err)
	}
	s.logger.Info("PSQL Client is initialized")

	s.logger.Info("Server is initialized...")
	
	apiauthserver.RegisterApiAuthServerServer(s.server, s)

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
			s.psqlConn.Close()
			s.vaultConn.Close()
			s.walletCreatorConn.Close()
		})
	}

	if err := g.Run(); err != nil {
		s.logger.Error(err)
	}
}

// Registration...
func (s *ApiAuthServer) Registration(
	ctx context.Context, 
	in *apiauthserver.RegistrationRequest) (*apiauthserver.RegistrationResponse, error) {
	u := &apipsqlserver.UserModel{UserName: in.GetUsername(), Password: in.GetPassword()}
	model.Validate(u)

	ctx_cancel, cancel := context.WithCancel(ctx)
	defer cancel()

	// Vault
	_, err := s.vaultClient.EncryptPassword(
		ctx_cancel,
		&apivaultserver.VaultWorkerPasswordEncryptRequest{
			Username: u.UserName,
			Password: u.Password,
		},
	)

	if err != nil {
		return nil, err
	}

	u.Password = "" // crearing password string
	s.logger.Info("Password putted to vault")

	// WalletCreator
	r, err := s.walletCreatorClient.NewWallet(
		ctx_cancel,
		&apiwalletcreatorserver.WalletCreatorRequest{
			UserName: u.UserName,
		},
	)

	if err != nil {
		s.logger.Error(err)
	}

	_, err = s.vaultClient.EncryptInternalWalletAddr(
		ctx_cancel, 
		&apivaultserver.VaultWorkerInternalWalletAddrEncryptRequest{
			Username: u.UserName, 
			WalletAddr: r.WalletAddr,
		})
	
	if err != nil {
		s.logger.Error(err)
	}

	_, err = s.vaultClient.EncryptPrivateKey(
		ctx_cancel, 
		&apivaultserver.VaultWorkerPrivateKeyEncryptRequest{
			Username: u.UserName, 
			PrivateKey: r.PrivateKey,
		})
	
	if err != nil {
		s.logger.Error(err)
	}

	s.logger.Info("Internal Wallet Address and Private Key has been putted to vault")
	
	_, err = s.psqlClient.CreateUser(
		ctx_cancel,
		&apipsqlserver.UserCreateRequest{
			U: u,
		},
	)
	s.logger.Info("User putted to database")

	return &apiauthserver.RegistrationResponse{}, nil
}

// === PRIVATE METHODS
func (s *ApiAuthServer) initVaultClient() error {
	//Set up connection to the vault server
	conn, err := grpc.Dial(
		fmt.Sprintf(":%d", s.config.VaultPort), 
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	
	if err != nil {
		return err
	}

	s.vaultConn = conn
	s.vaultClient = apivaultserver.NewVaultWorkerClient(conn)
	return nil
}

func (s *ApiAuthServer) initWalletCreatorClient() error {
	//Set up connection to the walletcreator server
	conn, err := grpc.Dial(
		fmt.Sprintf(":%d", s.config.WalletCreatorPort), 
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	
	if err != nil {
		return err
	}

	s.walletCreatorConn = conn
	s.walletCreatorClient = apiwalletcreatorserver.NewWalletCreatorClient(conn)
	return nil
}

func (s *ApiAuthServer) initPSQLClient() error {
	//Set up connection to the psql server
	conn, err := grpc.Dial(
		fmt.Sprintf(":%d", s.config.PSQLPort), 
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	
	if err != nil {
		return err
	}

	s.psqlConn = conn
	s.psqlClient = apipsqlserver.NewApiPSQLServerClient(conn)
	return nil
}

// ============= CONFIGURATION METHODS
// Configurates logger with config
func (s *ApiAuthServer) configureAuthLogger() error {
	level, err := logrus.ParseLevel(s.config.LogLevel)
	if err != nil {
		return err
	}

	s.logger.SetLevel(level)
	return nil
}
