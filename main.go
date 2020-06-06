package main

import (
	"os"
	"path/filepath"
	"time"

	"github.com/chill/plaidqif/internal"
	"gopkg.in/alecthomas/kingpin.v2"
)

// QIF spec: https://web.archive.org/web/20100222214101/http://web.intuit.com/support/quicken/docs/d_qif.html

var (
	root        = kingpin.New("plaidqif", "Downloads transactions from financial institutions using Plaid, and converts them to QIF files")
	configDir   = root.Flag("confdir", "Directory where plaintext plaidqif configuration is stored").Default(filepath.Join(mustHomeDir(), ".plaidqif")).PlaceHolder("$HOME/.plaidqif").String()
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

	listAccounts    = root.Command("list-accounts", "List accounts from an institution")
	listInstitution = listAccounts.Arg("institution", "Institution to list accounts from").Required().String()

	configureAccount   = root.Command("configure-account", "Configure an account with a friendly name and account type")
	accountName        = configureAccount.Arg("name", "Your friendly name for the account, also used as !Account header in QIF").Required().String()
	accountType        = configureAccount.Arg("type", "Type of account, used as !Type header in QIF, probably just Bank or CCard").Required().String()
	accountInstitution = configureAccount.Arg("institution", "Institution account is with").Required().String()
	accountID          = configureAccount.Arg("id", "Plaid Account ID, can be determined using plaidqif list-accounts").Required().String()

	downloadTransactions = root.Command("download", "Download transactions into QIFs")
	downloadTo           = downloadTransactions.Flag("until", "Date to download transactions up to, inclusive, any Go-supported date format").Default(time.Now().Format("02/01/2006")).String()
	downloadOutDir       = downloadTransactions.Flag("outdir", "Directory to write QIFs into, defaults to current working dir").Default(mustWorkingDir()).ExistingDir()
	downloadFrom         = downloadTransactions.Arg("from", "Date to download transactions from, inclusive, any Go-supported date format").Required().String()
	downloadAccounts     = downloadTransactions.Arg("accounts", "Account(s) to download, by their friendly name you configured, defaults to all accounts").Strings()
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
	case configureAccount.FullCommand():
	case downloadTransactions.FullCommand():
	default:
		kingpin.Fatalf("Unknown command ")
	}

	if err != nil {
		kingpin.Fatalf("%v", err)
	}
}

func mustHomeDir() string {
	return mustDir(os.UserHomeDir, "home")
}

func mustWorkingDir() string {
	return mustDir(os.Getwd, "working")
}

func mustDir(fn func() (string, error), kind string) string {
	dir, err := fn()
	if err != nil {
		kingpin.Fatalf("could not determine %s directory: %v", kind, err)
	}

	return dir
}
