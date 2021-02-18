package internal

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"text/template"
	"time"

	"github.com/chill/plaidqif/internal/institutions"
)

const updateTempl = `<html>
    <body>
        <button id='linkButton'>Plaid Link Update: {{.Institution}}</button>
        <script src="https://cdn.plaid.com/link/v2/stable/link-initialize.js"></script>
        <script>
            var linkHandler = Plaid.create({
                token: '{{.LinkToken}}',
                onSuccess: function(publicToken, metadata) {
                    let insName = "{{.Institution}}";

                    let req = new XMLHttpRequest();
                    let callbackPath = "{{.CallbackPath}}";
                    req.open("POST", callbackPath);
                    req.setRequestHeader("Content-Type", "application/json;charset=UTF-8");

                    console.log('metadata: ' + JSON.stringify(metadata))
                    req.send(JSON.stringify({"institutionName": insName}));
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
                linkHandler.open();
            };
        </script>
    </body>
</html>`

var updateTemplate = template.Must(template.New("update").Parse(updateTempl))

type updateFields struct {
	Environment  string
	ClientName   string
	Country      string
	Institution  string
	CallbackPath string
	LinkToken    string
}

func (p *PlaidQIF) UpdateInstitution(insName string) error {
	const (
		updatePath   = "/update"
		callbackPath = "/updateCallback"
	)

	ins, err := p.institutions.GetInstitution(insName)
	if err != nil {
		return err
	}

	linkToken, err := p.getLinkUpdateToken(ins)
	if err != nil {
		return err
	}

	errs := make(chan error)
	updateHandler, err := p.updateHandler(callbackPath, linkToken, ins.Name, errs)
	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.HandleFunc(updatePath, updateHandler)
	mux.HandleFunc(callbackPath, p.updateCallbackHandler(ins, errs))

	server := &http.Server{Addr: p.listenAddr, Handler: mux}
	go server.ListenAndServe()
	defer server.Close()

	fmt.Printf("Open %s in a web browser to update %s\n", path.Join(p.listenAddr, updatePath), insName)

	return <-errs
}

func (p *PlaidQIF) updateHandler(callbackPath, linkToken, institution string, errChan chan<- error) (http.HandlerFunc, error) {
	lf := updateFields{
		Environment:  p.plaidEnv,
		ClientName:   p.clientName,
		Country:      p.plaidCountry,
		Institution:  institution,
		CallbackPath: callbackPath,
		LinkToken:    linkToken,
	}

	return func(rw http.ResponseWriter, _ *http.Request) {
		if err := updateTemplate.Execute(rw, lf); err != nil {
			errChan <- fmt.Errorf("error writing link page: %w", err)
			close(errChan)
			return
		}

		// don't close here, await callback in the other handler
	}, nil
}

type updateCallback struct {
	InstitutionName string
}

func (p *PlaidQIF) updateCallbackHandler(ins institutions.Institution, errChan chan<- error) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		bs, err := ioutil.ReadAll(req.Body)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			errChan <- fmt.Errorf("unable to read callback body: %w", err)
			close(errChan)
			return
		}

		var callbackReq updateCallback
		if err := json.Unmarshal(bs, &callbackReq); err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			errChan <- fmt.Errorf("unable to unmarshal callback body: %w", err)
			close(errChan)
			return
		}

		if callbackReq.InstitutionName != ins.Name {
			rw.WriteHeader(http.StatusBadRequest)
			errChan <- fmt.Errorf("received institution name '%s' but expected '%s'",
				callbackReq.InstitutionName, ins.Name)
			close(errChan)
			return
		}

		// we don't have to rotate the access token, we just need the updated consent expiry

		itemResp, err := p.client.GetItem(ins.AccessToken)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			errChan <- fmt.Errorf("error looking up item with plaid using access token: %w", err)
			close(errChan)
			return
		}

		ins, err = p.institutions.UpdateConsentExpiry(ins.Name, itemResp.Item.ConsentExpirationTime)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			errChan <- fmt.Errorf("error updating instutiton '%s' consent expiry to %s: %w",
				ins.Name, itemResp.Item.ConsentExpirationTime.Format(time.RFC822), err)
			close(errChan)
			return
		}

		fmt.Printf("updated instutiton '%s' consent expiry to %s\n", ins.Name, itemResp.Item.ConsentExpirationTime.Format(time.RFC822))
		close(errChan)
		rw.WriteHeader(http.StatusOK)
	}
}
