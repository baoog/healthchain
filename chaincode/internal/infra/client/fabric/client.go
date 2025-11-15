package fabric

import (
	"consentcc/config"
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/asn1"
	"encoding/pem"
	"fmt"
	gw "github.com/hyperledger/fabric-gateway/pkg/client"
	"github.com/hyperledger/fabric-gateway/pkg/identity"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"math/big"
	"os"
	"time"
)

type Client struct {
	cfg      *config.Config
	gwConn   *grpc.ClientConn
	gateway  *gw.Gateway
	network  *gw.Network
	contract *gw.Contract
}

type FabricClient interface {
	Contract() *gw.Contract
	ContractEvents(ctx context.Context) (<-chan *gw.ChaincodeEvent, func(), error)
	Close()
}

func NewClient(cfg *config.Config) (*Client, error) {
	c := &Client{cfg: cfg}
	if err := c.connect(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Client) Contract() *gw.Contract { return c.contract }

func (c *Client) connect() error {
	tlsCert, err := os.ReadFile(os.ExpandEnv(c.cfg.FabricConfig.TLSCertPath))
	if err != nil {
		return fmt.Errorf("read tls cert: %w", err)
	}
	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(tlsCert) {
		return fmt.Errorf("failed to add tls cert to pool")
	}
	creds := credentials.NewClientTLSFromCert(certPool, "")

	var clientOption credentials.TransportCredentials
	if c.cfg.FabricConfig.TLSEnable {
		clientOption = creds
	} else {
		clientOption = insecure.NewCredentials()
	}

	conn, err := grpc.Dial(c.cfg.FabricConfig.PeerEndpoint, grpc.WithTransportCredentials(clientOption))
	if err != nil {
		return fmt.Errorf("grpc dial: %w", err)
	}
	c.gwConn = conn

	idCert, err := os.ReadFile(os.ExpandEnv(c.cfg.FabricConfig.CertPath))
	if err != nil {
		return err
	}
	keyPem, err := os.ReadFile(os.ExpandEnv(c.cfg.FabricConfig.KeyPath))
	if err != nil {
		return err
	}

	id, sign, err := identityFromX509(c.cfg.FabricConfig.MSP, idCert, keyPem)
	if err != nil {
		return err
	}

	block, _ := pem.Decode(idCert)
	cert, _ := x509.ParseCertificate(block.Bytes)

	gwClient, err := gw.Connect(
		id,
		gw.WithSign(sign),
		gw.WithClientConnection(conn),
		gw.WithEvaluateTimeout(5*time.Second),
		gw.WithEndorseTimeout(15*time.Second),
		gw.WithSubmitTimeout(15*time.Second),
		gw.WithCommitStatusTimeout(30*time.Second))
	if err != nil {
		return fmt.Errorf("gateway connect: %w", err)
	}

	c.gateway = gwClient
	c.network = gwClient.GetNetwork(c.cfg.FabricConfig.Channel)
	c.contract = c.network.GetContract(c.cfg.FabricConfig.Chaincode)
	log.Printf("[Fabric Identity] MSP=%s CN=%s IssuerCN=%s NotAfter=%s",
		c.cfg.FabricConfig.MSP, cert.Subject.CommonName, cert.Issuer.CommonName, cert.NotAfter)

	return nil
}

func (c *Client) Close() {
	if c.gateway != nil {
		c.gateway.Close()
	}
	if c.gwConn != nil {
		_ = c.gwConn.Close()
	}
}

func (c *Client) Events(ctx context.Context) (<-chan *gw.BlockAndPrivateDataEventsRequest, error) {
	// Use contract events API (all chaincode events)
	// Newer gateway uses contract.Events(ctx)
	return nil, fmt.Errorf("use ContractEvents instead")
}

func (c *Client) ContractEvents(ctx context.Context) (<-chan *gw.ChaincodeEvent, func(), error) {
	// Some Fabric Gateway versions return the channel directly (no iterator/request type)
	// Create a cancellable context so callers can stop the stream via returned cancel()
	ctx2, cancelCtx := context.WithCancel(ctx)

	ch, err := c.network.ChaincodeEvents(ctx2, c.cfg.FabricConfig.Chaincode)
	if err != nil {
		cancelCtx()
		return nil, nil, fmt.Errorf("subscribe chaincode events: %w", err)
	}

	cancel := func() { cancelCtx() }
	return ch, cancel, nil
}

// ---- helpers
func identityFromX509(mspID string, certPEM, keyPEM []byte) (identity.Identity, identity.Sign, error) {
	// 1) Lấy cert
	cert, err := identity.CertificateFromPEM(certPEM)
	if err != nil {
		return nil, nil, fmt.Errorf("parse cert: %w", err)
	}

	// 2) Tạo identity
	id, err := identity.NewX509Identity(mspID, cert)
	if err != nil {
		return nil, nil, fmt.Errorf("new identity: %w", err)
	}

	// 3) Lấy private key từ PEM (SDK sẽ lo PKCS8/EC)
	privKey, err := identity.PrivateKeyFromPEM(keyPEM)
	if err != nil {
		return nil, nil, fmt.Errorf("parse private key: %w", err)
	}

	// 4) Tạo hàm sign đúng chuẩn (không tự hash thêm nữa)
	sign, err := identity.NewPrivateKeySign(privKey)
	if err != nil {
		return nil, nil, fmt.Errorf("new signer: %w", err)
	}

	return id, sign, nil
}

func parseECPrivateKey(pemBytes []byte) (*ecdsa.PrivateKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, fmt.Errorf("no pem data")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err == nil {
		if k, ok := key.(*ecdsa.PrivateKey); ok {
			return k, nil
		}
	}
	k, err := x509.ParseECPrivateKey(block.Bytes)
	return k, err
}

func asn1Sig(r, s interface{}) []byte {
	rr, _ := r.(*big.Int)
	ss, _ := s.(*big.Int)
	type ecdsaSig struct{ R, S *big.Int }
	der, _ := asn1.Marshal(ecdsaSig{R: rr, S: ss})
	return der
}
