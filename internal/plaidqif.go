package internal

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"github.com/plaid/plaid-go/plaid"

	"github.com/chill/plaidqif/internal/files"
	"github.com/chill/plaidqif/internal/institutions"
)

type PlaidQIF struct {
	institutions *institutions.InstitutionManager
	client       *plaid.PlaidApiService
	plaidCountry plaid.CountryCode
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

	countryCode, err := plaid.NewCountryCodeFromValue(country)
	if err != nil {
		return nil, fmt.Errorf("invalid plaid country code '%s': %w", country, err)
	}

	listenAddr := net.JoinHostPort("127.0.0.1", strconv.Itoa(listenPort))
	if _, err := net.ResolveTCPAddr("tcp", listenAddr); err != nil {
		return nil, fmt.Errorf("unable to resolve listen address '%s': %w", listenAddr, err)
	}

	institutionMgr, err := institutions.NewInstitutionManager(confDir)
	if err != nil {
		return nil, err
	}

	return &PlaidQIF{
		institutions: institutionMgr,
		client:       newPlaidClient(creds, env).PlaidApi,
		plaidCountry: *countryCode,
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
	products := []plaid.Products{plaid.PRODUCTS_TRANSACTIONS}

	return p.createLinkToken(plaid.LinkTokenCreateRequest{
		User: plaid.LinkTokenCreateRequestUser{
			ClientUserId: p.userID,
		},
		ClientName:   p.clientName,
		CountryCodes: []plaid.CountryCode{p.plaidCountry},
		Language:     "en",
		Products:     &products,
	})
}

// getLinkUpdateToken returns a link token for use in the link "update" flow.
func (p *PlaidQIF) getLinkUpdateToken(ins institutions.Institution) (string, error) {
	return p.createLinkToken(plaid.LinkTokenCreateRequest{
		User: plaid.LinkTokenCreateRequestUser{
			ClientUserId: p.userID,
		},
		ClientName:   p.clientName,
		CountryCodes: []plaid.CountryCode{p.plaidCountry},
		Language:     "en",
		AccessToken:  &ins.AccessToken,
	})
}

func (p *PlaidQIF) createLinkToken(req plaid.LinkTokenCreateRequest) (string, error) {
	r := p.client.LinkTokenCreate(context.TODO())
	r = r.LinkTokenCreateRequest(req)
	resp, _, err := r.Execute()
	if err != nil {
		return "", fmt.Errorf("unable to create link token, request ID: '%s', err: %w",
			resp.RequestId, err)
	}

	return resp.LinkToken, nil
}

func newPlaidClient(creds Credentials, env plaid.Environment) *plaid.APIClient {
	configuration := plaid.NewConfiguration()
	configuration.AddDefaultHeader("PLAID-CLIENT-ID", creds.ClientID)
	configuration.AddDefaultHeader("PLAID-SECRET", creds.Secret)
	configuration.UseEnvironment(env)
	return plaid.NewAPIClient(configuration)

}
