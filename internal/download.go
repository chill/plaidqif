package internal

var plaidToQIFType = map[string]string{
	"credit":     "CCard",
	"depository": "Bank",
}

func (p *PlaidQIF) DownloadTransactions(institutionNames []string, fr, to, outDir string) error {
	return nil
}
