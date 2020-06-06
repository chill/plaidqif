package internal

import (
	"fmt"
	"net"
	"os"
	"strconv"

	"github.com/plaid/plaid-go/plaid"
)

type PlaidQIF struct {
	confDir      string
	client       *plaid.Client
	plaidCountry string
	clientName   string
	listenAddr   string
}

var validPlaidEnvs = map[string]plaid.Environment{
	"sandbox":     plaid.Sandbox,
	"development": plaid.Development,
	"production":  plaid.Production,
}

func PlaidQif(confDir, plaidEnv, country, clientName string, listenPort int) (*PlaidQIF, error) {
	if err := confdirExists(confDir); err != nil {
		return nil, err
	}

	creds, err := readCreds(confDir)
	if err != nil {
		return nil, err
	}

	env, ok := validPlaidEnvs[plaidEnv]
	if !ok {
		return nil, fmt.Errorf("unknown plaid environment '%s'", plaidEnv)
	}

	listenAddr := net.JoinHostPort("127.0.0.1", strconv.Itoa(listenPort))
	if _, err := net.ResolveTCPAddr("tcp", listenAddr); err != nil {
		return nil, fmt.Errorf("unable to resolve listen address '%s': %w", listenAddr, err)
	}

	client, err := plaid.NewClient(plaid.ClientOptions{
		ClientID:    creds.ClientID,
		Secret:      creds.Secret,
		PublicKey:   creds.PublicKey,
		Environment: env,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create plaid client: %w", err)
	}

	return &PlaidQIF{
		confDir:      confDir,
		client:       client,
		plaidCountry: country,
		clientName:   clientName,
		listenAddr:   listenAddr,
	}, nil
}

// confdirExists will create a directory at path (including parents where necessary) if it does not exist.
// if path does exist and is not a directory, it will error.
func confdirExists(path string) error {
	s, err := os.Stat(path)
	if os.IsNotExist(err) {
		fmt.Printf("confdir '%s' does not exist, creating\n", path)
		if err := os.MkdirAll(path, 0700); err != nil {
			return fmt.Errorf("failed to create confdir at '%s': %w", path, err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to stat confdir '%s': %w", path, err)
	}

	if !s.IsDir() {
		return fmt.Errorf("confdir '%s' is not a directory", path)
	}

	return nil
}
