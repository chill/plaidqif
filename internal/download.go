package internal

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/plaid/plaid-go/plaid"

	"github.com/chill/plaidqif/internal/files"
	"github.com/chill/plaidqif/internal/institutions"
	"github.com/chill/plaidqif/internal/qif"
)

const plaidDateFormat = "2006-01-02"

var (
	plaidToQIFType = map[string]string{
		"credit":     "CCard",
		"depository": "Bank",
	}
	spaceRegex = regexp.MustCompile(`\s+`)
)

func (p *PlaidQIF) DownloadTransactions(institutionNames []string, fr, to, outDir string) error {
	if err := files.IsExistingDir(outDir); err != nil {
		return fmt.Errorf("outdir: %w", err)
	}

	from, err := time.Parse(p.dateFormat, fr)
	if err != nil {
		return fmt.Errorf("cannot parse date to download transactions from '%s': %w", fr, err)
	}

	until, err := time.Parse(p.dateFormat, to)
	if err != nil {
		return fmt.Errorf("cannot parse date to download transactions until '%s': %w", to, err)
	}

	institutions, err := p.institutions.GetInstitutions(institutionNames)
	if err != nil {
		return err
	}

	for _, ins := range institutions {
		if err := p.downloadInstitutionTransactions(ins, from, until, outDir); err != nil {
			return err
		}
	}

	return nil
}

func (p *PlaidQIF) downloadInstitutionTransactions(ins institutions.Institution, from, until time.Time, outDir string) error {

	ins, accounts, err := p.getInstitutionAccounts(ins)
	if err != nil {

		return err
	}

	if ins.ConsentExpires.Before(time.Now().Add(10 * time.Minute)) {
		return fmt.Errorf("institution '%s' consent expires within 10 minutes: %s", ins.Name, ins.ConsentExpires.Format(time.RFC822))
	} else if ins.ConsentExpires.Before(time.Now().Add(24 * time.Hour)) {
		fmt.Printf("Institution '%s' consent expires within 1 day: %s\n", ins.Name, ins.ConsentExpires.Format(time.RFC822))
	} else if ins.ConsentExpires.Before(time.Now().Add(7 * 24 * time.Hour)) {
		fmt.Printf("Institution '%s' consent expires within 1 week: %s\n", ins.Name, ins.ConsentExpires.Format(time.RFC822))
	}

	for _, acct := range accounts {
		if err := p.downloadAccountTransactions(ins.Name, ins.AccessToken, acct, from, until, outDir); err != nil {
			return fmt.Errorf("failed to download transactions for account '%s' from institituon '%s': %w", acct.Name, ins.Name, err)
		}
	}

	return nil
}

func (p *PlaidQIF) downloadAccountTransactions(institution, accessToken string, acct plaid.Account, from, until time.Time, outDir string) error {
	opts := plaid.GetTransactionsOptions{
		StartDate:  from.Format(plaidDateFormat),
		EndDate:    until.Format(plaidDateFormat),
		AccountIDs: []string{acct.AccountID},
		Offset:     0,
		Count:      100,
	}

	resp, err := p.client.GetTransactionsWithOptions(accessToken, opts)
	if err != nil {
		return fmt.Errorf("failed to get transactions from plaid: %w", err)
	}

	total := resp.TotalTransactions
	opts.Offset = len(resp.Transactions)

	if total == 0 {
		return nil
	}

	qifType, ok := plaidToQIFType[acct.Type]
	if !ok {
		return fmt.Errorf("unknown plaid account type '%s'", acct.Type)
	}

	outputPath := filepath.Join(outDir, fmt.Sprintf("%s_%s.qif", institution, acct.Name))
	f, err := files.OpenWriter(outputPath, "qif")
	if err != nil {
		return err
	}
	defer f.Close()

	w := qif.NewWriter(f, acct.Name, qifType, p.dateFormat)
	for {
		if err := appendTransactions(w, resp.Transactions); err != nil {
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

	if err := f.Close(); err != nil {
		return fmt.Errorf("failed to close qif file '%s': %w", outputPath, err)
	}

	return nil
}

func appendTransactions(w *qif.Writer, transactions []plaid.Transaction) error {
	if len(transactions) == 0 {
		return nil
	}

	qifTransactions, err := convertTransactions(transactions)
	if err != nil {
		return err
	}

	if err := w.WriteTransactions(qifTransactions); err != nil {
		return fmt.Errorf("failed to write transactions to qif writer: %w", err)
	}

	return nil
}

func convertTransactions(transactions []plaid.Transaction) ([]qif.Transaction, error) {
	txs := make([]qif.Transaction, 0, len(transactions))
	for _, tx := range transactions {
		payee := tx.Name
		if p := strings.TrimSpace(tx.PaymentMeta.Payee); p != "" {
			payee = p
		}

		date, err := time.Parse(plaidDateFormat, tx.Date)
		if err != nil {
			return nil, fmt.Errorf("failed to parse transaction date for payee '%s' with date string '%s: %w", payee, tx.Date, err)
		}

		txs = append(txs, qif.Transaction{
			Date:   date,
			Payee:  payee,
			Amount: tx.Amount,
		})
	}

	return txs, nil
}
