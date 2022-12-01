package internal

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/plaid/plaid-go/plaid"

	"github.com/chill/plaidqif/internal/institutions"
)

func (p *PlaidQIF) ListAccounts(names []string) error {
	tw := tabwriter.NewWriter(os.Stdout, 0, 8, 2, '\t', 0)
	defer tw.Flush()

	fmt.Fprintln(tw, "Accounts:")
	fmt.Fprintln(tw, "Institution\tName\tPlaid Type\tQIF Type\tPlaid Account ID\tConsent Expires\t")
	fmt.Fprintln(tw, "-----------\t----\t----------\t--------\t----------------\t---------------\t")

	institutions, err := p.institutions.GetInstitutions(names)
	if err != nil {
		return err
	}

	for _, ins := range institutions {
		if err := p.listInstitutionAccounts(tw, ins); err != nil {
			return err
		}
	}

	return nil
}

func (p *PlaidQIF) listInstitutionAccounts(tw *tabwriter.Writer, ins institutions.Institution) error {
	ins, accounts, err := p.getInstitutionAccounts(ins)
	if err != nil {
		return err
	}

	for _, acct := range accounts {
		// we'll get empty string if this is unknown, that's fine
		qifType := plaidToQIFType[acct.Type]

		fmt.Fprintln(tw, fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%s\t",
			ins.Name, acct.Name, acct.Type, qifType, acct.AccountId, ins.ConsentExpires.Format(time.RFC822)))
	}

	return nil
}

func (p *PlaidQIF) getInstitutionAccounts(ins institutions.Institution) (institutions.Institution, []plaid.AccountBase, error) {
	req := p.client.AccountsGet(context.TODO())
	req = req.AccountsGetRequest(plaid.AccountsGetRequest{
		AccessToken: ins.AccessToken,
	})

	resp, _, err := req.Execute()
	if err != nil {
		return ins, nil, fmt.Errorf("failed to get institution '%s' accounts from plaid: %w", ins.Name, err)
	}

	expiry := resp.Item.ConsentExpirationTime.Get()
	if expiry == nil {
		panic(fmt.Errorf("error getting instutiton details for '%s', no consent expiry", ins.Name))
	}

	ins, err = p.institutions.UpdateConsentExpiry(ins.Name, *expiry)
	if err != nil {
		return ins, nil, fmt.Errorf("failed to update consent expiry for existing institution '%s': %w", ins.Name, err)
	}

	return ins, resp.Accounts, nil
}
