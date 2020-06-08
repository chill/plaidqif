package internal

const linkTempl = `<html>
	<body>
		<button id='linkButton'>Open Plaid Link - Institution Select</button>
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
				onSuccess: function(public_token, metadata) {
					{{if not .PublicToken}}
					// Send the public_token to your app server here. No need for this func if in update mode
					// The metadata object contains info about the institution the
					// user selected and the account ID, if selectAccount is enabled.
					console.log('public_token: '+public_token+', metadata: '+JSON.stringify(metadata));
					{{end}}
				},
				onExit: function(err, metadata) {
					if (err === null) {
						return
					}

					console.log('error: ' + error)
					console.log('metadata: ' + JSON.stringify(metadata))
				}
			});
			
			document.getElementById('linkButton').onclick = function() {
				linkHandler.open();
			};
		</script>
	</body>
</html>`
