package apiserver

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	wc_pb "gowallet/cmd/walletcreator/proto"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/oklog/run"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

// ApiWalletCreatorServer...
type ApiWalletCreatorServer struct {
	wc_pb.UnimplementedWalletCreatorServer
	config *WallerCreatorConfig
	logger *logrus.Logger
	server *grpc.Server
}

// NewApiWalletCreatorServer...
func NewApiWalletCreatorServer(config *WallerCreatorConfig) *ApiWalletCreatorServer {
	return &ApiWalletCreatorServer{
		config: config,
		logger: logrus.New(),
		server: grpc.NewServer(),
	}
}

// Start...
func (s *ApiWalletCreatorServer) Start() {
	if err := s.configureWalletCreatorLogger(); err != nil {
		s.logger.Error(err)
	}
	s.logger.Info("Server is initialized...")
	
	wc_pb.RegisterWalletCreatorServer(s.server, s)

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

// === API METHODS

// NewWallet creates wallet and returns private_key and address of wallet
func (s *ApiWalletCreatorServer) NewWallet(
	ctx context.Context,
	in *wc_pb.WalletCreatorRequest) (*wc_pb.WalletCreatorResponse, error) {
	var private, walletAddress string

	privateKey, err := crypto.GenerateKey()
	if err != nil {
		
		return &wc_pb.WalletCreatorResponse{}, err
	}

	privateKeyBytes := crypto.FromECDSA(privateKey)
	private = hexutil.Encode(privateKeyBytes) // private key to return

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatalf("value type assertion failed: %T %#v", publicKey, publicKey)
	}

	// TODO: print for debug
	//publicKeyBytes := crypto.FromECDSAPub(publicKeyECDSA)
	//fmt.Println("Public Key: ", hexutil.Encode(publicKeyBytes))

	walletAddress = crypto.PubkeyToAddress(*publicKeyECDSA).Hex()
	if err != nil {
		return &wc_pb.WalletCreatorResponse{}, err
	}

	return &wc_pb.WalletCreatorResponse{
		WalletAddr: walletAddress,
		PrivateKey: private,
	}, nil
}
// === API METHODS

// ============= CONFIGURATION METHODS
// Configurates logger with config
func (s *ApiWalletCreatorServer) configureWalletCreatorLogger() error {
	level, err := logrus.ParseLevel(s.config.LogLevel)
	if err != nil {
		return err
	}

	s.logger.SetLevel(level)
	return nil
}