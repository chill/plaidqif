package internal

import (
	"path/filepath"
)

type Credentials struct {
	ClientID  string
	PublicKey string
	Secret    string
}

func WriteCredentials(confDir string, creds Credentials) error {
	if err := confdirExists(confDir); err != nil {
		return err
	}

	return marshalFile(credPath(confDir), "credential", creds)
}

func credPath(dir string) string {
	const filename = "creds.json"
	return filepath.Join(dir, filename)
}

func readCreds(confDir string) (Credentials, error) {
	var creds Credentials
	if err := unmarshalFile(credPath(confDir), "credential", &creds); err != nil {
		return Credentials{}, err
	}

	return creds, nil
}
