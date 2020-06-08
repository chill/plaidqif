package internal

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"text/template"

	"github.com/chill/plaidqif/internal/institutions"
)

const linkTempl = `<html>
    <body>
        <input type="text" name="uinsname" id="uinsname" placeholder="Your institution name">
        <button id='linkButton'>Plaid Link: Institution Select</button>
        <script src="https://cdn.plaid.com/link/v2/stable/link-initialize.js"></script>
        <script>
            var linkHandler = Plaid.create({
                env: '{{.Environment}}',
                clientName: '{{.ClientName}}',
                countryCodes: ['{{.Country}}'],
                key: '{{.PublicKey}}',
                product: 'transactions',
                apiVersion: 'v2',
                {{if .PublicToken}}
                // update mode
                // from plaid.com/docs/#creating-public-tokens: POST /item/public_token/create, client_id and secret from dashboard, access token from institution
                token: '{{.PublicToken}}',
                {{end}}
                onSuccess: function(publicToken, metadata) {
                    let insName = document.getElementById('uinsname').value

                    let req = new XMLHttpRequest();
                    let callbackURL = "{{.CallbackURL}}";
                    req.open("POST", callbackURL);
                    req.setRequestHeader("Content-Type", "application/json;charset=UTF-8");

                    req.send(JSON.stringify({"publicToken": publicToken, "institutionName": insName, "metadata": JSON.stringify(metadata)}));
                },
                onExit: function(err, metadata) {
                    if (err === null) {
                        return
                    }

                    console.log('error: ' + err)
                    console.log('metadata: ' + JSON.stringify(metadata))
                }
            });

            document.getElementById('linkButton').onclick = function() {
                if (document.getElementById('uinsname').value === "") {
                    window.alert("Provide a friendly name for the institution you are about to link")
                    return
                }
                linkHandler.open();
            };
        </script>
    </body>
</html>`

var linkTemplate = template.Must(template.New("link").Parse(linkTempl))

type linkFields struct {
	Environment string
	ClientName  string
	Country     string
	PublicKey   string
	PublicToken string // optional, set for token refresh (plaid link update mode)
	CallbackURL string
}

func (p *PlaidQIF) LinkInstitution() error {
	const callbackPath = "/linkCallback"
	callbackURL := path.Join(p.listenAddr, callbackPath)

	errs := make(chan error)
	mux := http.NewServeMux()
	mux.HandleFunc("/link", p.linkHandler(callbackURL, errs))
	mux.HandleFunc(callbackPath, p.linkCallbackHandler(errs))

	if err := http.ListenAndServe(p.listenAddr, mux); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return <-errs
}

func (p *PlaidQIF) linkHandler(callbackURL string, errChan chan<- error) func(http.ResponseWriter, *http.Request) {
	lf := linkFields{
		Environment: p.plaidEnv,
		ClientName:  p.clientName,
		Country:     p.plaidCountry,
		PublicKey:   p.publicKey,
		CallbackURL: callbackURL,
	}

	return func(rw http.ResponseWriter, _ *http.Request) {
		if err := linkTemplate.Execute(rw, lf); err != nil {
			errChan <- fmt.Errorf("error writing link page: %w", err)
			close(errChan)
			return
		}

		// don't close here, await callback
	}
}

type callback struct {
	PublicToken     string
	InstitutionName string
	Metadata        json.RawMessage
}

func (p *PlaidQIF) linkCallbackHandler(errChan chan<- error) func(http.ResponseWriter, *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		bs, err := ioutil.ReadAll(req.Body)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			errChan <- fmt.Errorf("unable to read callback body: %w", err)
			close(errChan)
			return
		}

		var callbackReq callback
		if err := json.Unmarshal(bs, &callbackReq); err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			errChan <- fmt.Errorf("unable to unmarshal callback body: %w", err)
			close(errChan)
			return
		}

		tokResp, err := p.client.ExchangePublicToken(callbackReq.PublicToken)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			errChan <- fmt.Errorf("error exchanging public token with plaid: %w", err)
			close(errChan)
			return
		}

		itemResp, err := p.client.GetItem(tokResp.AccessToken)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			errChan <- fmt.Errorf("error looking up item with plaid using access token: %w", err)
			close(errChan)
			return
		}

		institution := institutions.Institution{
			Name:           callbackReq.InstitutionName,
			AccessToken:    tokResp.AccessToken,
			ItemID:         tokResp.ItemID,
			ConsentExpires: &itemResp.Item.ConsentExpirationTime,
		}

		if err := p.institutions.AddInstitution(institution); err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			fmt.Printf("%s\n", err)
			// only error here is name already exists, but we randomise and write anyway, so don't return it
			close(errChan)
			return
		}

		close(errChan)
	}
}
