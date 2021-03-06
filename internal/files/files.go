package files

import (
	"encoding/json"
	"fmt"
	"os"
)

// DirExists will create a directory at path (including parents where necessary) if it does not exist.
// if path does exist and is not a directory, it will error.
func DirExists(path, kind string) error {
	err := IsExistingDir(path)
	if err == nil {
		return nil
	}

	if !os.IsNotExist(err) {
		return fmt.Errorf("%s: %w", kind, err)
	}

	if err := os.MkdirAll(path, 0700); err != nil {
		return fmt.Errorf("failed to create %s at '%s': %w", kind, path, err)
	}

	return nil
}

func IsExistingDir(path string) error {
	s, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat '%s': %w", path, err)
	}

	if !s.IsDir() {
		return fmt.Errorf("'%s' is not a directory", path)
	}

	return nil
}

func Unmarshal(path, kind string, v interface{}) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open %s file '%s' for reading: %w", kind, path, err)
	}
	defer f.Close()

	if err := json.NewDecoder(f).Decode(v); err != nil {
		return fmt.Errorf("failed to unmarshal %s file '%s': %w", kind, path, err)
	}

	return nil
}

func MarshalFile(path, kind string, v interface{}) error {
	f, err := OpenWriter(path, kind)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "\t")

	if err := enc.Encode(v); err != nil {
		return fmt.Errorf("failed to marshal %s file '%s': %w", kind, path, err)
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("failed to close %s file '%s': %w", kind, path, err)
	}

	return nil
}

func OpenWriter(path, kind string) (*os.File, error) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s file '%s' for writing: %w", kind, path, err)
	}

	return f, nil
}

