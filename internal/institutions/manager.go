package institutions

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/chill/plaidqif/internal/files"
)

type institutions map[string]Institution

type Institution struct {
	Name           string
	AccessToken    string
	ItemID         string
	ConsentExpires time.Time
}

// InstitutionManager is not safe for concurrent use
type InstitutionManager struct {
	path         string
	institutions institutions
}

// NewInstitutionManager assumed confDir already exists.
// The returned InstitutionManager is not safe for concurrent use.
func NewInstitutionManager(confDir string) (*InstitutionManager, error) {
	path := filepath.Join(confDir, "institutions.json")

	var institutions institutions
	err := files.Unmarshal(path, "institutions", &institutions)
	if err != nil && !errors.Is(err, os.ErrNotExist) { // ignore ErrNotExist
		return nil, err
	}

	// if there was no file, unmarshal failed, but that's fine:
	// we would only have an empty institutions map anyway, so just continue
	return &InstitutionManager{
		path:         path,
		institutions: institutions,
	}, nil
}

func (m *InstitutionManager) GetInstitution(name string) (Institution, error) {
	ins, ok := m.institutions[name]
	if !ok {
		return Institution{}, fmt.Errorf("institution '%s' not yet configured", name)
	}

	return ins, nil
}

func (m *InstitutionManager) AddInstitution(ins Institution) error {
	var err error
	if _, ok := m.institutions[ins.Name]; ok {
		var newName string
		for ok {
			newName := fmt.Sprintf("%s_%s", ins.Name, strconv.Itoa(rand.Int()))
			_, ok = m.institutions[newName]
		}

		err = fmt.Errorf("institution '%s' already exists, adding as '%s'", ins.Name, newName)
		ins.Name = newName
	}

	m.institutions[ins.Name] = ins
	return err
}

func (m *InstitutionManager) List() []Institution {
	ordered := make([]Institution, 0, len(m.institutions))
	for _, ins := range m.institutions {
		ordered = append(ordered, ins)
	}

	sort.Slice(ordered, func(i, j int) bool {
		return ordered[i].Name < ordered[j].Name
	})

	return ordered
}

func (m *InstitutionManager) UpdateConsentExpiry(name string, newExpiry time.Time) (Institution, error) {
	ins, ok := m.institutions[name]
	if !ok {
		return Institution{}, fmt.Errorf("institution '%s' not yet configured", name)
	}

	// noop if expiry already set and matches provided expiry
	if !ins.ConsentExpires.IsZero() && ins.ConsentExpires.Equal(newExpiry) {
		return ins, nil
	}

	expiry := newExpiry.UTC()
	ins.ConsentExpires = expiry
	m.institutions[name] = ins
	return ins, nil
}

func (m *InstitutionManager) WriteInstitutions() error {
	return files.MarshalFile(m.path, "institutions", m.institutions)
}

// GetInstitutions returns all institutions if len(names) == 0
func (m *InstitutionManager) GetInstitutions(names []string) ([]Institution, error) {
	is := make([]Institution, 0, len(names))

	for _, name := range names {
		ins, err := m.GetInstitution(name)
		if err != nil {
			return nil, err
		}

		is = append(is, ins)
	}

	if len(is) == 0 {
		is = m.List()
	}

	return is, nil
}
