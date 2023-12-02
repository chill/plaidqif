package internal

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/chill/plaidqif/internal/institutions"
	"github.com/plaid/plaid-go/plaid"
)

func (p *PlaidQIF) ListInstitutions() error {
	institutions := p.institutions.List()

	tw := tabwriter.NewWriter(os.Stdout, 0, 8, 2, '\t', 0)
	defer tw.Flush()

	fmt.Fprintln(tw, "Configured Institutions:")
	fmt.Fprintln(tw, "Payee\tPlaid Access Token\tPlaid Item ID\tConsent Expires\t")
	fmt.Fprintln(tw, "----\t------------------\t-------------\t---------------\t")

	for _, ins := range institutions {
		if err := p.printInstitutionDetails(tw, ins); err != nil {
			return err
		}
	}

	return nil
}

func (p *PlaidQIF) printInstitutionDetails(tw *tabwriter.Writer, ins institutions.Institution) error {
	itemGet := p.client.ItemGet(context.TODO())
	itemGet = itemGet.ItemGetRequest(plaid.ItemGetRequest{AccessToken: ins.AccessToken})

	resp, _, err := itemGet.Execute()
	if err != nil {
		return fmt.Errorf("unable to get institution details from plaid for institution '%s': %w",
			ins.Name, err)
	}

	expiry := resp.Item.ConsentExpirationTime.Get()
	if expiry == nil {
		// some items can have no expiry we can get at, let's set those 100 years into the future...
		future := time.Now().AddDate(100, 0, 0)
		expiry = &future
	}

	ins, err = p.institutions.UpdateConsentExpiry(ins.Name, *expiry)
	if err != nil {
		// this should never happen since we already got the institution above
		panic(fmt.Errorf("failed to update consent expiry for existing institution '%s': %w", ins.Name, err))
	}

	// could also add the last transaction update time? fine for now
	fmt.Fprintln(tw, fmt.Sprintf("%s\t%s\t%s\t%s\t",
		ins.Name, ins.AccessToken, ins.ItemID, ins.ConsentExpires.Format(time.RFC822)))
	return nil
}
