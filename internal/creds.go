package internal

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/chill/plaidqif/internal/files"
)

type Credentials struct {
	ClientID  string
	PublicKey string
	Secret    string
}

func WriteCredentials(confDir string, creds Credentials) error {
	if err := files.DirExists(confDir, "confdir"); err != nil {
		return err
	}

	return files.MarshalFile(credPath(confDir), "credential", creds)
}

func credPath(dir string) string {
	const filename = "creds.json"
	return filepath.Join(dir, filename)
}

func readCreds(confDir string) (Credentials, error) {
	path := credPath(confDir)
	var creds Credentials
	if err := files.Unmarshal(path, "credential", &creds); err != nil {
		return Credentials{}, err
	}

	var missing []string
	if creds.ClientID == "" {
		missing = append(missing, "ClientID")
	}

	if creds.PublicKey == "" {
		missing = append(missing, "PublicKey")
	}

	if creds.Secret == "" {
		missing = append(missing, "Secret")
	}

	if len(missing) != 0 {
		return creds, fmt.Errorf("missing fields [%s] from %s", strings.Join(missing, ", "), path)
	}

	return creds, nil
}
