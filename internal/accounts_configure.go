package internal

import (
	"fmt"
	"time"

	"github.com/plaid/plaid-go/plaid"
)

var plaidToQIFType = map[string]string{
	"credit":     "CCard",
	"depository": "Bank",
}

func (p *PlaidQIF) ConfigureAccount(acctName, insName, acctID string) error {
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

	expiry := resp.Item.ConsentExpirationTime
	ins, err = p.updateConsentExpiry(insName, ins, institutions, expiry, false)
	if err != nil {
		return err
	}

	// get any configured accounts from the file by their IDs, so we can print them alongside plaid ones
	byID := plaidAcctsByID(resp.Accounts)
	plaidAcct, ok := byID[acctID]
	if !ok {
		return fmt.Errorf("account with plaid id '%s' not found for institution '%s'", acctID, insName)
	}

	if len(ins.Accounts) == 0 {
		ins.Accounts = make(map[string]Account)
	}

	acctType, ok := plaidToQIFType[plaidAcct.Type]
	if !ok {
		return fmt.Errorf("unknown plaid account type '%s' for account with id '%s', from institition '%s'",
			plaidAcct.Type, acctID, insName)
	}

	ins.Accounts[acctName] = Account{
		Type:    acctType,
		PlaidID: acctID,
	}

	institutions[insName] = ins

	if err := writeInstitutions(p.confDir, institutions); err != nil {
		return err
	}

	fmt.Println("Account configured")
	fmt.Printf("Institution '%s' consent expires on %s at %s\n",
		insName, ins.ConsentExpires.Format(DateFormat), ins.ConsentExpires.Format(timeFormat))

	return nil

}

func plaidAcctsByID(accounts []plaid.Account) map[string]plaid.Account {
	byID := make(map[string]plaid.Account, len(accounts))
	for _, acct := range accounts {
		byID[acct.AccountID] = acct
	}

	return byID
}

func (p *PlaidQIF) updateConsentExpiry(name string, ins Institution, institutions Institutions, newExpiry time.Time, writeFile bool) (Institution, error) {
	// if a consent expiry was not already set, or it differs to what plaid reports, set/update it in the file
	if ins.ConsentExpires != nil && ins.ConsentExpires.Equal(newExpiry) {
		return ins, nil
	}

	expiry := newExpiry.UTC()
	ins.ConsentExpires = &expiry

	institutions[name] = ins

	if !writeFile {
		return ins, nil
	}

	if err := writeInstitutions(p.confDir, institutions); err != nil {
		return ins, fmt.Errorf("failed to update institution '%s' consent expiry: %w", name, err)
	}

	return ins, nil
}
