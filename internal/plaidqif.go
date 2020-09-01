package internal

import (
	"fmt"
	"net"
	"strconv"

	"github.com/plaid/plaid-go/plaid"

	"github.com/chill/plaidqif/internal/files"
	"github.com/chill/plaidqif/internal/institutions"
)

type PlaidQIF struct {
	institutions *institutions.InstitutionManager
	client       *plaid.Client
	plaidCountry string
	plaidEnv     string
	publicKey    string
	clientName   string
	listenAddr   string
	dateFormat   string
}

var validPlaidEnvs = map[string]plaid.Environment{
	"sandbox":     plaid.Sandbox,
	"development": plaid.Development,
	"production":  plaid.Production,
}

func PlaidQif(confDir, plaidEnv, clientName, country, dateFormat string, listenPort int) (*PlaidQIF, error) {
	if err := files.DirExists(confDir, "confdir"); err != nil {
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

	institutionMgr, err := institutions.NewInstitutionManager(confDir)
	if err != nil {
		return nil, err
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
		institutions: institutionMgr,
		client:       client,
		plaidCountry: country,
		plaidEnv:     plaidEnv,
		publicKey:    creds.PublicKey,
		clientName:   clientName,
		listenAddr:   listenAddr,
		dateFormat:   dateFormat,
	}, nil
}

// Close writes any updates to institutions that took place during the execution of a command, to disk
func (p *PlaidQIF) Close() error {
	return p.institutions.WriteInstitutions()
}
