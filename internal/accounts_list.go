package internal

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"
)

type Accounts map[string]Account

type Account struct {
	Type    string
	PlaidID string
}

func (p *PlaidQIF) ListAccounts(institutions []string) error {
	tw := tabwriter.NewWriter(os.Stdout, 0, 8, 2, '\t', 0)
	defer tw.Flush()

	for i, ins := range institutions {
		if err := p.listInstitutionAccounts(tw, ins); err != nil {
			return err
		}

		if i != len(institutions)-1 {
			fmt.Fprintln(tw, "")
		}
	}

	return nil
}

func (p *PlaidQIF) listInstitutionAccounts(tw *tabwriter.Writer, insName string) error {
	institutions, err := readInstitutions(p.confDir)
	if err != nil {
		return err
	}

	ins, ok := institutions[insName]
	if !ok {
		return fmt.Errorf("institution '%s' not yet configured", insName)
	}

	resp, err := p.client.GetAccounts(ins.AccessToken)
	if err != nil {
		return fmt.Errorf("failed to get institution '%s' accounts from plaid: %w", insName, err)
	}

	// if a consent expiry was not already set, or it differs to what plaid reports, set/update it in the file
	if ins.ConsentExpires == nil || !ins.ConsentExpires.Equal(resp.Item.ConsentExpirationTime) {
		expiry := resp.Item.ConsentExpirationTime.UTC()
		ins.ConsentExpires = &expiry

		institutions[insName] = ins

		if err := writeInstitutions(p.confDir, institutions); err != nil {
			return fmt.Errorf("failed to update institution '%s' consent expiry: %w", insName, err)
		}
	}

	fmt.Fprintln(tw, fmt.Sprintf("Accounts for Institution '%s':", insName))
	fmt.Fprintln(tw, "Your Name\tYour Type\tPlaid Name\tOfficial Name\tPlaid Type\tPlaid Account ID\t")
	fmt.Fprintln(tw, "---------\t---------\t----------\t-------------\t----------\t----------------\t")

	// get any configured accounts from the file by their IDs, so we can print them alongside plaid ones
	byID := accountsByID(ins.Accounts)
	for _, plaidAcct := range resp.Accounts {
		var configuredAcct namedAccount
		if got, ok := byID[plaidAcct.AccountID]; ok {
			// grab any matching existing account from the file, remove it from the byID mapping
			configuredAcct = got
			delete(byID, plaidAcct.AccountID)
		}

		// configuredAcct will be full of 0-vals if we didn't find it, so "Your Name/Type" will be empty
		fmt.Fprintln(tw, fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%s\t",
			configuredAcct.Name, configuredAcct.Type,
			plaidAcct.Name, plaidAcct.OfficialName, plaidAcct.Type, plaidAcct.AccountID))
	}

	// anything left in the byID mapping was not in plaid (who knows why), print them out too anyway
	orderedRemainder := orderAccounts(byID)
	for _, acct := range orderedRemainder {
		fmt.Fprintln(tw, fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%s\t",
			acct.Name, acct.Type, "", "", "", ""))
	}

	fmt.Fprintln(tw, "")
	fmt.Fprintln(tw, fmt.Sprintf("Institution '%s' consent expires on %s at %s",
		insName, ins.ConsentExpires.Format(DateFormat), ins.ConsentExpires.Format(timeFormat)))

	return nil
}

type namedAccount struct {
	Name string
	Account
}

func accountsByID(accounts Accounts) map[string]namedAccount {
	byID := make(map[string]namedAccount, len(accounts))
	for name, account := range accounts {
		byID[account.PlaidID] = namedAccount{
			Name:    name,
			Account: account,
		}
	}

	return byID
}

func orderAccounts(accounts map[string]namedAccount) []namedAccount {
	ordered := make([]namedAccount, 0, len(accounts))
	for _, account := range accounts {
		ordered = append(ordered, namedAccount{
			Name:    account.Name,
			Account: account.Account,
		})
	}

	sort.Slice(ordered, func(i, j int) bool {
		return ordered[i].Name < ordered[j].Name
	})

	return ordered
}
