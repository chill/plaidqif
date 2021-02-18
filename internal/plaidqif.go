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
	clientName   string
	userID       string
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
		clientName:   clientName,
		userID:       creds.UserID,
		listenAddr:   listenAddr,
		dateFormat:   dateFormat,
	}, nil
}

// Close writes any updates to institutions that took place during the execution of a command, to disk
func (p *PlaidQIF) Close() error {
	return p.institutions.WriteInstitutions()
}

// getLinkToken returns a link token for use in the link "setup" flow.
func (p *PlaidQIF) getLinkToken() (string, error) {
	return p.createLinkToken(plaid.LinkTokenConfigs{
		User: &plaid.LinkTokenUser{
			ClientUserID: p.userID,
		},
		ClientName:   p.clientName,
		CountryCodes: []string{p.plaidCountry},
		Language:     "en",
		Products:     []string{"transactions"},
	})
}

// getLinkUpdateToken returns a link token for use in the link "update" flow.
func (p *PlaidQIF) getLinkUpdateToken(ins institutions.Institution) (string, error) {
	return p.createLinkToken(plaid.LinkTokenConfigs{
		User: &plaid.LinkTokenUser{
			ClientUserID: p.userID,
		},
		ClientName:   p.clientName,
		CountryCodes: []string{p.plaidCountry},
		Language:     "en",
		AccessToken:  ins.AccessToken,
	})
}

func (p *PlaidQIF) createLinkToken(req plaid.LinkTokenConfigs) (string, error) {
	resp, err := p.client.CreateLinkToken(req)
	if err != nil {
		return "", fmt.Errorf("unable to create link token, request ID: '%s', err: %w",
			resp.RequestID, err)
	}

	return resp.LinkToken, nil
}
