package main

import (
	"os"
	"path/filepath"
	"time"

	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/chill/plaidqif/internal"
	"github.com/chill/plaidqif/internal/files"
)

const defaultDateFmt = "02/01/2006"

var (
	root        = kingpin.New("plaidqif", "Downloads transactions from financial institutions using Plaid, and converts them to QIF files")
	configDir   = root.Flag("confdir", "Directory where plaintext plaidqif configuration is stored").Default(filepath.Join(files.MustHomeDir(), ".plaidqif")).PlaceHolder("$HOME/.plaidqif").String()
	plaidEnv    = root.Flag("environment", "Plaid environment to connect to").Default("development").String()
	clientName  = root.Flag("client", "Payee of your client to connect to Plaid with").Default("plaidqif").String()
	countryCode = root.Flag("countrycode", "Plaid countryCode to connect with").Default("GB").String()
	dateFormat  = root.Flag("dateformat", "Format to use for parsing and writing dates, must be a string representing 2nd Jan 2006").Default(defaultDateFmt).String()
	listenPort  = root.Flag("port", "Port to listen on locally, for hosting Plaid Link UI and receiving callbacks from it").Default("8080").Int()

	setupCreds = root.Command("setup-creds", "Set Plaid credentials for plaidqif")
	clientID   = setupCreds.Arg("clientid", "Plaid client_id from the dashboard").Required().String()
	publicKey  = setupCreds.Arg("publickey", "Plaid public_key from the dashboard").Required().String()
	secret     = setupCreds.Arg("secret", "Plaid secret from the dashboard").Required().String()

	institutionSetup = root.Command("setup-ins", "Set up institutions")

	updateInstitution     = root.Command("update-ins", "Update an institution's consent")
	updateInstitutionName = updateInstitution.Arg("institution", "Institution to update").Required().String()

	listInstitutions = root.Command("list-ins", "List institutions")

	listAccounts            = root.Command("list-accounts", "List accounts from an institution")
	listAccountInstitutions = listAccounts.Arg("institutions", "Institution to list accounts from, defaults to all").Strings()

	downloadTransactions = root.Command("download", "Download transactions into QIFs")
	downloadUntil        = downloadTransactions.Flag("until", "Date to download transactions up to, inclusive, defaults to today").Default(time.Now().Format(defaultDateFmt)).String()
	downloadOutDir       = downloadTransactions.Flag("outdir", "Directory to write QIFs into, defaults to current working dir").Default(files.MustWorkingDir()).PlaceHolder("<workdir>").ExistingDir()
	downloadFrom         = downloadTransactions.Arg("from", "Date to download transactions from, inclusive").Required().String()
	downloadInstitutions = downloadTransactions.Arg("institutions", "Institution(s) to download transactions from, for your configured accounts, defaults to all").Strings()
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

	pq, err := internal.PlaidQif(*configDir, *plaidEnv, *clientName, *countryCode, *dateFormat, *listenPort)
	if err != nil {
		kingpin.Fatalf("%v", err)
	}

	switch cmd {
	case institutionSetup.FullCommand():
		err = pq.LinkInstitution()
	case updateInstitution.FullCommand():
		err = pq.UpdateInstitution(*updateInstitutionName)
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
