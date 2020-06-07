package main

import (
	"os"
	"path/filepath"
	"time"

	"github.com/chill/plaidqif/internal"
	"github.com/chill/plaidqif/internal/files"
	"gopkg.in/alecthomas/kingpin.v2"
)

// QIF spec: https://web.archive.org/web/20100222214101/http://web.intuit.com/support/quicken/docs/d_qif.html

var (
	root        = kingpin.New("plaidqif", "Downloads transactions from financial institutions using Plaid, and converts them to QIF files")
	configDir   = root.Flag("confdir", "Directory where plaintext plaidqif configuration is stored").Default(filepath.Join(files.MustHomeDir(), ".plaidqif")).PlaceHolder("$HOME/.plaidqif").String()
	plaidEnv    = root.Flag("environment", "Plaid environment to connect to").Default("development").String()
	clientName  = root.Flag("client", "Name of your client to connect to Plaid with").Default("plaidqif").String()
	countryCode = root.Flag("countrycode", "Plaid countryCode to connect with").Default("GB").String()
	listenPort  = root.Flag("port", "Port to listen on locally, for hosting Plaid Link UI and receiving callbacks from it").Default("8080").Int()

	setupCreds = root.Command("setup-creds", "Set Plaid credentials for plaidqif")
	clientID   = setupCreds.Arg("clientid", "Plaid client_id from the dashboard").Required().String()
	publicKey  = setupCreds.Arg("publickey", "Plaid public_key from the dashboard").Required().String()
	secret     = setupCreds.Arg("secret", "Plaid secret from the dashboard").Required().String()

	institutionSetup = root.Command("setup-ins", "Set up a new institution for plaidqif")
	institutionName  = institutionSetup.Arg("name", "Your friendly name for the institution to set up").Required().String()

	listInstitutions = root.Command("list-ins", "List institutions")

	listAccounts            = root.Command("list-accounts", "List accounts from an institution")
	listAccountInstitutions = listAccounts.Arg("institutions", "Institution to list accounts from").Required().Strings()

	downloadTransactions = root.Command("download", "Download transactions into QIFs")
	downloadUntil        = downloadTransactions.Flag("until", "Date to download transactions up to, inclusive, DD/MM/YYYY").Default(time.Now().Format(internal.DateFormat)).String()
	downloadOutDir       = downloadTransactions.Flag("outdir", "Directory to write QIFs into, defaults to current working dir").Default(files.MustWorkingDir()).ExistingDir()
	downloadFrom         = downloadTransactions.Arg("from", "Date to download transactions from, inclusive, DD/MM/YYYY").Required().String()
	downloadInstitutions = downloadTransactions.Arg("institutions", "Institution(s) to download transactions from, for your configured accounts").Strings()
)

func main() {
	cmd := kingpin.MustParse(root.Parse(os.Args[1:]))

	if cmd == setupCreds.FullCommand() {
		if err := internal.WriteCredentials(*configDir, internal.Credentials{
			ClientID:  *clientID,
			PublicKey: *publicKey,
			Secret:    *secret,
		}); err != nil {
			kingpin.Fatalf("%v", err)
		}

		return
	}

	pq, err := internal.PlaidQif(*configDir, *plaidEnv, *clientName, *countryCode, *listenPort)
	if err != nil {
		kingpin.Fatalf("%v", err)
	}

	switch cmd {
	case institutionSetup.FullCommand():
	case listInstitutions.FullCommand():
		err = pq.ListInstitutions()
	case listAccounts.FullCommand():
		err = pq.ListAccounts(*listAccountInstitutions)
	case downloadTransactions.FullCommand():
		err = pq.DownloadTransactions(*downloadInstitutions, *downloadFrom, *downloadUntil, *downloadOutDir)
	default:
		kingpin.Fatalf("Unknown command ")
	}

	// if we haven't errored out yet, close pq, so that institutions get written to disk, updating err
	// we don't defer the close, because we don't want to write the file if something goes wrong
	if err == nil {
		err = pq.Close()
	}

	if err != nil {
		kingpin.Fatalf("%v", err)
	}
}
