package internal

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/chill/plaidqif/internal/institutions"
)

func (p *PlaidQIF) ListInstitutions() error {
	institutions := p.institutions.List()

	tw := tabwriter.NewWriter(os.Stdout, 0, 8, 2, '\t', 0)
	defer tw.Flush()

	fmt.Fprintln(tw, "Configured Institutions:")
	fmt.Fprintln(tw, "Name\tPlaid Access Token\tPlaid Item ID\tConsent Expires\t")
	fmt.Fprintln(tw, "----\t------------------\t-------------\t---------------\t")

	for _, ins := range institutions {
		if err := p.printInstitutionDetails(tw, ins); err != nil {
			return err
		}
	}

	return nil
}

func (p *PlaidQIF) printInstitutionDetails(tw *tabwriter.Writer, ins institutions.Institution) error {
	resp, err := p.client.GetItem(ins.AccessToken)
	if err != nil {
		return fmt.Errorf("unable to get institution details from plaid for institution '%s': %w",
			ins.Name, err)
	}

	ins, err = p.institutions.UpdateConsentExpiry(ins.Name, resp.Item.ConsentExpirationTime)
	if err != nil {
		// this should never happen since we already got the institution above
		panic(fmt.Errorf("failed to update consent expiry for existing institution '%s': %w", ins.Name, err))
	}

	// could also add the last transaction update time? fine for now
	fmt.Fprintln(tw, fmt.Sprintf("%s\t%s\t%s\t%s\t",
		ins.Name, ins.AccessToken, ins.ItemID, ins.ConsentExpires.Format(time.RFC822)))
	return nil
}
