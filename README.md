# plaidqif
A tool for downloading transactions from Plaid

Installation and usage:
```
go get github.com/chill/plaidqif

plaidqif --help // usage information, use --help on any command to find out more

plaidqif creds <client_id> <public_key> <secret> // from plaid dashboard
plaidqif setup-ins // repeat for as many institutions you need
plaidqif list-ins // see the institutions you configured
plaidqif list-accounts // see all available accounts for your institutions
plaidqif download <DD/MM/YYYY> // download transactions since the date provided for all accounts
plaidqif update-ins <institution-name> // update consent for an institution you previously configured
```