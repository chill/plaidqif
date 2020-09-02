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
                onSuccess: function(publicToken, metadata) {
                    let insName = document.getElementById('uinsname').value

                    let req = new XMLHttpRequest();
                    let callbackPath = "{{.CallbackPath}}";
                    req.open("POST", callbackPath);
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
	Environment  string
	ClientName   string
	Country      string
	PublicKey    string
	CallbackPath string
}

// TODO update to use link tokens: https://plaid.com/docs/upgrade-to-link-tokens/
func (p *PlaidQIF) LinkInstitution() error {
	const (
		linkPath     = "/link"
		callbackPath = "/linkCallback"
	)

	errs := make(chan error)
	mux := http.NewServeMux()
	mux.HandleFunc(linkPath, p.linkHandler(callbackPath, errs))
	mux.HandleFunc(callbackPath, p.linkCallbackHandler(errs))

	server := &http.Server{Addr: p.listenAddr, Handler: mux}
	go server.ListenAndServe()
	defer server.Close()

	fmt.Printf("Open %s in a web browser to link an institution\n", path.Join(p.listenAddr, linkPath))

	return <-errs
}

func (p *PlaidQIF) linkHandler(callbackPath string, errChan chan<- error) http.HandlerFunc {
	lf := linkFields{
		Environment:  p.plaidEnv,
		ClientName:   p.clientName,
		Country:      p.plaidCountry,
		PublicKey:    p.publicKey,
		CallbackPath: callbackPath,
	}

	return func(rw http.ResponseWriter, _ *http.Request) {
		if err := linkTemplate.Execute(rw, lf); err != nil {
			errChan <- fmt.Errorf("error writing link page: %w", err)
			close(errChan)
			return
		}

		// don't close here, await callback in the other handler
	}
}

type linkCallback struct {
	PublicToken     string
	InstitutionName string
	Metadata        json.RawMessage
}

func (p *PlaidQIF) linkCallbackHandler(errChan chan<- error) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		bs, err := ioutil.ReadAll(req.Body)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			errChan <- fmt.Errorf("unable to read callback body: %w", err)
			close(errChan)
			return
		}

		var callbackReq linkCallback
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
		rw.WriteHeader(http.StatusOK)
	}
}
