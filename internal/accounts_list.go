package internal

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"
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

	resp, err := p.client.GetAccounts(ins.AccessToken)
	if err != nil {
		return fmt.Errorf("failed to get institution '%s' accounts from plaid: %w", insName, err)
	}

	ins, err = p.institutions.UpdateConsentExpiry(ins.Name, resp.Item.ConsentExpirationTime)
	if err != nil {
		// this should never happen since we already got the institution above
		panic(fmt.Errorf("failed to update consent expiry for existing institution '%s': %w", ins.Name, err))
	}

	for _, plaidAcct := range resp.Accounts {
		// we'll get empty string if this is unknown, that's fine
		qifType := plaidToQIFType[plaidAcct.Type]

		fmt.Fprintln(tw, fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%s\t",
			ins.Name, plaidAcct.Name, plaidAcct.Type, qifType, plaidAcct.AccountID, ins.ConsentExpires.Format(time.RFC822)))
	}

	return nil
}
