package osutil

import (
	"os"
	"os/user"

	"gopkg.in/alecthomas/kingpin.v2"
)

func MustHomeDir() string {
	return mustDir(os.UserHomeDir, "home")
}

func MustWorkingDir() string {
	return mustDir(os.Getwd, "working")
}

func mustDir(fn func() (string, error), kind string) string {
	dir, err := fn()
	if err != nil {
		kingpin.Fatalf("could not determine %s directory: %v", kind, err)
	}

	return dir
}

func MustUsername() string {
	usr, err := user.Current()
	if err != nil {
		kingpin.Fatalf("could not retrieve current user info: %v", err)
	}

	return usr.Username
}
