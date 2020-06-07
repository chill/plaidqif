package internal

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/chill/plaidqif/internal/institutions"
	"github.com/plaid/plaid-go/plaid"
)

func (p *PlaidQIF) ListAccounts(institutions []string) error {
	tw := tabwriter.NewWriter(os.Stdout, 0, 8, 2, '\t', 0)
	defer tw.Flush()

	fmt.Fprintln(tw, "Accounts:")
	fmt.Fprintln(tw, "Institution\tName\tPlaid Type\tQIF Type\tPlaid Account ID\tConsent Expires\t")
	fmt.Fprintln(tw, "-----------\t----\t----------\t--------\t----------------\t---------------\t")

	for _, ins := range institutions {
		if err := p.listInstitutionAccounts(tw, ins); err != nil {
			return err
		}
	}

	return nil
}

func (p *PlaidQIF) listInstitutionAccounts(tw *tabwriter.Writer, insName string) error {
	ins, err := p.institutions.GetInstitution(insName)
	if err != nil {
		return err
	}

	ins, accounts, err := p.getInstitutionAccounts(ins)
	if err != nil {
		return err
	}

	for _, acct := range accounts {
		// we'll get empty string if this is unknown, that's fine
		qifType := plaidToQIFType[acct.Type]

		fmt.Fprintln(tw, fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%s\t",
			ins.Name, acct.Name, acct.Type, qifType, acct.AccountID, ins.ConsentExpires.Format(time.RFC822)))
	}

	return nil
}

func (p *PlaidQIF) getInstitutionAccounts(ins institutions.Institution) (institutions.Institution, []plaid.Account, error) {
	resp, err := p.client.GetAccounts(ins.AccessToken)
	if err != nil {
		return ins, nil, fmt.Errorf("failed to get institution '%s' accounts from plaid: %w", ins.Name, err)
	}

	ins, err = p.institutions.UpdateConsentExpiry(ins.Name, resp.Item.ConsentExpirationTime)
	if err != nil {
		return ins, nil, fmt.Errorf("failed to update consent expiry for existing institution '%s': %w", ins.Name, err)
	}

	return ins, resp.Accounts, nil
}
