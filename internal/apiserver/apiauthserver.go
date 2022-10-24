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

	u, err := s.putPasswordToVault(u)
	if err != nil {
		s.logger.Error(err)
	}
	s.logger.Info("Password putted to vault")

	
	walletAddr, privateKey, err := s.createWallet(u)
	if err != nil {
		s.logger.Error(err)
	}
	err = s.putWalletDataToVault(u, walletAddr, privateKey)
	s.logger.Info("Internal Wallet Address and Private Key has been putted to vault")
	
	err = s.putToDatabase(u)
	s.logger.Info("User putted to database")

	return &apiauthserver.RegistrationResponse{}, nil
}

// === PRIVATE METHODS
// putPasswordToVault...
func (s *ApiAuthServer) putPasswordToVault(u *apipsqlserver.UserModel) (*apipsqlserver.UserModel, error) {
	//Set up connection to the vault server
	connVault, err := grpc.Dial(
		fmt.Sprintf(":%d", s.config.VaultPort), 
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	
	if err != nil {
		return nil, err
	}
	defer connVault.Close()

	client := apivaultserver.NewVaultWorkerClient(connVault)
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, err = client.EncryptPassword(
		ctx,
		&apivaultserver.VaultWorkerPasswordEncryptRequest{
			Username: u.UserName,
			Password: u.Password,
		},
	)

	if err != nil {
		return nil, err
	}

	u.Password = ""
	return u, nil
}

// putToDatabase...
func (s *ApiAuthServer) putToDatabase(u *apipsqlserver.UserModel) (error) {
	connPSQL, err := grpc.Dial(
		fmt.Sprintf(":%d", s.config.PSQLPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return err
	}
	defer connPSQL.Close()

	client := apipsqlserver.NewApiPSQLServerClient(connPSQL)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, err = client.CreateUser(
		ctx,
		&apipsqlserver.UserCreateRequest{
			U: u,
		},
	)

	return nil
}

// putWalletDataToVault...
func (s *ApiAuthServer) putWalletDataToVault(u *apipsqlserver.UserModel, walletAddr, privateKey string) error {
	connVault, err := grpc.Dial(
		fmt.Sprintf(":%d", s.config.VaultPort), 
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	
	if err != nil {
		return err
	}
	defer connVault.Close()

	client := apivaultserver.NewVaultWorkerClient(connVault)
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	/*  encrypt data  */
	_, err = client.EncryptInternalWalletAddr(
		ctx, 
		&apivaultserver.VaultWorkerInternalWalletAddrEncryptRequest{
			Username: u.UserName, 
			WalletAddr: walletAddr,
		})
	
		if err != nil {
		return err
	}

	_, err = client.EncryptPrivateKey(
		ctx, 
		&apivaultserver.VaultWorkerPrivateKeyEncryptRequest{
			Username: u.UserName, 
			PrivateKey: privateKey,
		})
	if err != nil {
		return err
	}

	if err != nil {
		return err
	}

	return nil
}

// createWallet...
func (s *ApiAuthServer) createWallet(u *apipsqlserver.UserModel) (string, string, error) {
	connWallet, err := grpc.Dial(
		fmt.Sprintf(":%d", s.config.WalletCreatorPort), 
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	
	if err != nil {
		return "", "", err
	}
	defer connWallet.Close()

	client := apiwalletcreatorserver.NewWalletCreatorClient(connWallet)
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	r, err := client.NewWallet(
		ctx,
		&apiwalletcreatorserver.WalletCreatorRequest{
			UserName: u.UserName,
		},
	)

	if err != nil {
		return "", "", err
	}

	return r.WalletAddr, r.PrivateKey, nil
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
