package apiserver

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	pb_v "gowallet/cmd/vaultworker/proto"
	"gowallet/interfaces"

	v "github.com/hashicorp/vault/api"
	"github.com/oklog/run"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

// ApiVaultServer...
type ApiVaultServer struct {
	interfaces.Server
	pb_v.UnimplementedVaultWorkerServer

	logger *logrus.Logger
	config *VaultConfig
	vaultClient *v.Client
	server *grpc.Server
}

// NewVaultServer...
func NewVaultServer(config *VaultConfig) (*ApiVaultServer, error) {
	client, err := initVaultClient(config)
	if err != nil {
		return nil, err
	}
	
	return &ApiVaultServer{
		logger: logrus.New(),
		config: config,
		server: grpc.NewServer(),
		vaultClient: client,
	}, nil
}

// === PRIVATE METHODS
// InitVaultClient...
func initVaultClient(vc *VaultConfig) (*v.Client, error) {
	config := v.DefaultConfig()
	config.Address = vc.BindAddr

	client, err := v.NewClient(config)
	if err != nil {
		return nil, err
	}

	client.SetToken(vc.Token)
	return client, nil
}

func (s *ApiVaultServer) Start() {
	if err := s.configureVaultLogger(); err != nil {
		log.Fatalln(err)
	}
	
	pb_v.RegisterVaultWorkerServer(s.server, s)

	s.logger.Info("Server initialized...")

	var g run.Group
	{
		listen, err := net.Listen("tcp", fmt.Sprintf(":%d", s.config.BindPort))
		if err != nil {
			s.logger.Error(err)
		}
		g.Add(func() error {
			s.logger.Info("Server listening at port: ", listen.Addr())
			return s.server.Serve(listen)
		}, func(error){
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
			s.vaultClient.ClearToken()
			s.server.GracefulStop()
		})
	}

	if err := g.Run(); err != nil {
		s.logger.Error(err)
	}
}

/* ----------------------- Encryption ----------------------- */

// returns empty result if ok
func (s *ApiVaultServer) Encrypt(
	ctx context.Context, 
	in *pb_v.VaultWorkerDefaultEncryptRequest) (*pb_v.VaultWorkerDefaultEncryptResponse, error) {
	privateKey := map[string]interface{}{
		"private_key": in.GetPrivateKey(),
	}

	walletAddr := map[string]interface{}{
		"wallet_addr": in.GetAddr(),
	}

	pswd := map[string]interface{}{
		"password": in.GetPassword(),
	}

	_, err := s.vaultClient.KVv2(
		fmt.Sprintf("secret/data/wallets/%v/address",
	 in.GetUsername())).Put(ctx, s.config.SecretName, privateKey)
	if err != nil {
		logrus.Error(err)
		return &pb_v.VaultWorkerDefaultEncryptResponse{}, err
	}

	_, err = s.vaultClient.KVv2(
		fmt.Sprintf("secret/data/wallets/%v/pk",
	 in.GetUsername())).Put(ctx, s.config.SecretName, walletAddr)
	if err != nil {
		logrus.Error(err)
		return &pb_v.VaultWorkerDefaultEncryptResponse{}, err
	}

	_, err = s.vaultClient.KVv2(
		fmt.Sprintf("secret/data/wallets/%v/pswd",
	 in.GetUsername())).Put(ctx, s.config.SecretName, pswd)
	if err != nil {
		logrus.Error(err)
		return &pb_v.VaultWorkerDefaultEncryptResponse{}, err
	}

	return &pb_v.VaultWorkerDefaultEncryptResponse{}, nil
}

func (s *ApiVaultServer) EncryptPassword(
	ctx context.Context, 
	in *pb_v.VaultWorkerPasswordEncryptRequest) (*pb_v.VaultWorkerDefaultEncryptResponse, error) {
	pswd := map[string]interface{}{
		"password": in.GetPassword(),
	}

	_, err := s.vaultClient.KVv2(fmt.Sprintf("secret/data/wallets/%v/pswd", in.GetUsername())).Put(ctx, s.config.SecretName, pswd)
	if err != nil {
		logrus.Error(err)
		return &pb_v.VaultWorkerDefaultEncryptResponse{}, err
	}

	return &pb_v.VaultWorkerDefaultEncryptResponse{}, nil
}

// EncryptInternalWalletAddr...
func (s *ApiVaultServer) EncryptInternalWalletAddr(
	ctx context.Context, 
	in *pb_v.VaultWorkerInternalWalletAddrEncryptRequest) (*pb_v.VaultWorkerDefaultEncryptResponse, error) {
	iwa := map[string]interface{}{
		"wallet_addr": in.GetWalletAddr(),
	}

	_, err := s.vaultClient.KVv2(
		fmt.Sprintf(
			"secret/data/wallets/%v/address", 
			in.GetUsername())).Put(ctx, s.config.SecretName, iwa)
	if err != nil {
		logrus.Error(err)
		return &pb_v.VaultWorkerDefaultEncryptResponse{}, err
	}
	
	return &pb_v.VaultWorkerDefaultEncryptResponse{}, nil
}

func (s *ApiVaultServer) EncryptPrivateKey (
	ctx context.Context, 
	in *pb_v.VaultWorkerPrivateKeyEncryptRequest) (*pb_v.VaultWorkerDefaultEncryptResponse, error) {
	pk := map[string]interface{} {
		"private_key": in.GetPrivateKey(),
	}

	_, err := s.vaultClient.KVv2(fmt.Sprintf("secret/data/wallets/%v/pk", in.GetUsername())).Put(ctx, s.config.SecretName, pk)
	if err != nil {
		logrus.Error(err)
		return &pb_v.VaultWorkerDefaultEncryptResponse{}, err
	}

	return &pb_v.VaultWorkerDefaultEncryptResponse{}, nil
}

/* ----------------------- Decryption ----------------------- */

// DecryptPassword returns decrypted password
// DecryptPassword
func (s *ApiVaultServer) DecryptPassword(
	ctx context.Context, 
	in *pb_v.VaultWorkerPasswordDecryptRequest) (*pb_v.VaultWorkerPasswordDecryptResponse, error) {
	secretPassword, err := s.vaultClient.KVv2(fmt.Sprintf("secret/data/wallets/%v/pswd", in.GetUsername())).Get(ctx, s.config.SecretName)
	if err != nil {
		s.logger.Error(err)
		return &pb_v.VaultWorkerPasswordDecryptResponse{}, err
	}

	pswd, ok := secretPassword.Data["password"].(string)
	if !ok {
		log.Fatalf("value type assertion failed: %T %#v", secretPassword.Data["password"], secretPassword.Data["password"])
	}

	return &pb_v.VaultWorkerPasswordDecryptResponse{Password: pswd}, nil
}

// returns decrypted internal wallet address
func (s *ApiVaultServer) DecryptInternalWalletAddr (
	ctx context.Context,
	in *pb_v.VaultWorkerInternalWalletAddrDecryptRequest) (*pb_v.VaultWorkerInternalWalletAddrDecryptRespone, error) {
	secretWalletAddr, err := s.vaultClient.KVv2(
		fmt.Sprintf(
			"secret/data/wallets/%v/address", 
		in.GetUsername())).Get(ctx, s.config.SecretName)
	if err != nil {
		s.logger.Error(err)
		return &pb_v.VaultWorkerInternalWalletAddrDecryptRespone{}, err
	}

	wa, ok := secretWalletAddr.Data["wallet_addr"].(string)
	if !ok {
		log.Fatalf("value type assertion failed: %T %#v", 
		secretWalletAddr.Data["wallet_addr"],
		secretWalletAddr.Data["wallet_addr"])
	}
	
	return &pb_v.VaultWorkerInternalWalletAddrDecryptRespone{WalletAddr: wa}, nil
}

// returns decrypted private key
func (s *ApiVaultServer) DecryptPrivateKey (
	ctx context.Context, 
	in *pb_v.VaultWorkerPrivateKeyDecryptRequest) (*pb_v.VaultWorkerPrivateKeyDecryptResponse, error) {
	secretPK, err := s.vaultClient.KVv2(
		fmt.Sprintf("secret/data/wallets/%v/address", 
		in.GetUsername())).Get(ctx, s.config.SecretName)
	if err != nil {
		s.logger.Error(err)
		return &pb_v.VaultWorkerPrivateKeyDecryptResponse{}, err
	}

	pk, ok := secretPK.Data["wallet_addr"].(string)
	if !ok {
		log.Fatalf("value type assertion failed: %T %#v", 
		secretPK.Data["wallet_addr"],
		secretPK.Data["wallet_addr"])
	}
	
	return &pb_v.VaultWorkerPrivateKeyDecryptResponse{Pk: pk}, nil
}

// ============= CONFIGURATION METHODS
// Configurates logger with config
func (s *ApiVaultServer) configureVaultLogger() error {
	level, err := logrus.ParseLevel(s.config.LogLevel)
	if err != nil {
		return err
	}

	s.logger.SetLevel(level)
	return nil
}
