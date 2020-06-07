package internal

import (
	"fmt"
	"regexp"
	"time"

	"github.com/chill/plaidqif/internal/files"
	"github.com/plaid/plaid-go/plaid"
)

var (
	plaidToQIFType = map[string]string{
		"credit":     "CCard",
		"depository": "Bank",
	}
	spaceRegex = regexp.MustCompile(`\s+`)
)

func (p *PlaidQIF) DownloadTransactions(institutions []string, fr, to, outDir string) error {
	if err := files.IsExistingDir(outDir); err != nil {
		return fmt.Errorf("outdir: %w", err)
	}

	from, err := time.Parse(DateFormat, fr)
	if err != nil {
		return fmt.Errorf("cannot parse date to download transactions from '%s': %w", fr, err)
	}

	until, err := time.Parse(DateFormat, to)
	if err != nil {
		return fmt.Errorf("cannot parse date to download transactions until '%s': %w", to, err)
	}

	for _, name := range institutions {
		if err := p.downloadInstitutionTransactions(name, from, until, outDir); err != nil {
			return err
		}
	}

	return nil
}

func (p *PlaidQIF) downloadInstitutionTransactions(name string, from, until time.Time, outDir string) error {
	ins, err := p.institutions.GetInstitution(name)
	if err != nil {
		return err
	}

	ins, accounts, err := p.getInstitutionAccounts(ins)
	if err != nil {
		return err
	}

	if ins.ConsentExpires.Before(time.Now().Add(10 * time.Minute)) {
		return fmt.Errorf("institution '%s' consent expires within 10 minutes: %s", name, ins.ConsentExpires.Format(time.RFC822))
	} else if ins.ConsentExpires.Before(time.Now().Add(24 * time.Hour)) {
		fmt.Printf("Institution '%s' consent expires within 1 day: %s\n", name, ins.ConsentExpires.Format(time.RFC822))
	} else if ins.ConsentExpires.Before(time.Now().Add(7 * 24 * time.Hour)) {
		fmt.Printf("Institution '%s' consent expires within 1 week: %s\n", name, ins.ConsentExpires.Format(time.RFC822))
	}

	for _, acct := range accounts {
		// TODO pass through a QIF writer for filepath.Join(outDir, fmt.Sprintf("%s_%s.qif", ins.Name, acct.Name))

		if err := p.downloadAccountTransactions(ins.AccessToken, acct, from, until, outDir); err != nil {
			return fmt.Errorf("failed to download transactions for account '%s' from institituon '%s': %w", acct.Name, ins.Name, err)
		}
	}

	return nil
}

func (p *PlaidQIF) downloadAccountTransactions(accessToken string, acct plaid.Account, from, until time.Time, outDir string) error {
	const plaidDateFormat = "2006-01-02"

	opts := plaid.GetTransactionsOptions{
		StartDate:  from.Format(plaidDateFormat),
		EndDate:    until.Format(plaidDateFormat),
		AccountIDs: []string{acct.AccountID},
		Offset:     0,
	}

	resp, err := p.client.GetTransactionsWithOptions(accessToken, opts)
	if err != nil {
		return fmt.Errorf("failed to get transactions from plaid: %w", err)
	}

	total := resp.TotalTransactions
	opts.Offset = len(resp.Transactions)

	for {
		// TODO pass through a QIF writer
		if err := appendTransactions(resp.Transactions); err != nil {
			return err
		}

		if opts.Offset >= total {
			break
		}

		resp, err = p.client.GetTransactionsWithOptions(accessToken, opts)
		if err != nil {
			return fmt.Errorf("failed to get transactions from plaid: %w", err)
		}

		opts.Offset += len(resp.Transactions)
	}

	// TODO flush a QIF writer
	return nil
}

func appendTransactions(transactions []plaid.Transaction) error {
	if len(transactions) == 0 {
		return nil
	}

	return nil
}
