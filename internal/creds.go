package internal

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
)

type Credentials struct {
	ClientID  string
	PublicKey string
	Secret    string
}

const credFilename = "creds.json"

func WriteCredentials(dir string, creds Credentials) error {
	if err := confdirExists(dir); err != nil {
		return err
	}

	bs, err := json.Marshal(creds)
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	credFile := filepath.Join(dir, credFilename)
	if err := ioutil.WriteFile(credFile, bs, 0600); err != nil {
		return fmt.Errorf("failed to write credentials to '%s': %w", credFile, err)
	}

	return nil
}

func readCreds(dir string) (Credentials, error) {
	credFile := filepath.Join(dir, credFilename)
	bs, err := ioutil.ReadFile(credFile)
	if err != nil {
		return Credentials{}, fmt.Errorf("failed to read credential file '%s': %w", credFile, err)
	}

	var creds Credentials
	if err := json.Unmarshal(bs, &creds); err != nil {
		return Credentials{}, fmt.Errorf("failed to unmarshal credential file '%s': %w", credFile, err)
	}

	return creds, nil
}
