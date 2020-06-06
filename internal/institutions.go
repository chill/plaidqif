package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"text/tabwriter"
)

type Institutions map[string]Institution

type Institution struct {
	AccessToken string
	ItemID      string
}

func (p *PlaidQIF) ListInstitutions() error {
	institutions, err := readInstitutions(p.confDir)
	if err != nil {
		return err
	}

	type named struct {
		Name string
		Institution
	}

	ordered := make([]named, 0, len(institutions))
	for name, ins := range institutions {
		ordered = append(ordered, named{
			Name:        name,
			Institution: ins,
		})
	}

	sort.Slice(ordered, func(i, j int) bool {
		return ordered[i].Name < ordered[j].Name
	})

	tw := tabwriter.NewWriter(os.Stdout, 0, 8, 2, '\t', 0)
	defer tw.Flush()

	fmt.Fprintln(tw, "Institutions:")
	fmt.Fprintln(tw, "Name\tPlaid Access Token\tPlaid Item ID\t")
	fmt.Fprintln(tw, "----\t------------------\t-------------\t")

	for _, ins := range ordered {
		fmt.Fprintln(tw, fmt.Sprintf("%s\t%s\t%s\t", ins.Name, ins.AccessToken, ins.ItemID))
	}

	return nil
}

func institutionsPath(dir string) string {
	const filename = "institutions.json"
	return filepath.Join(dir, filename)
}

func readInstitutions(confDir string) (Institutions, error) {
	var institutions Institutions
	if err := unmarshalFile(institutionsPath(confDir), "institutions", &institutions); err != nil {
		return nil, err
	}

	return institutions, nil
}

func writeInstitutions(confDir string, institutions Institutions) error {
	if err := confdirExists(confDir); err != nil {
		return err
	}

	return marshalFile(credPath(confDir), "institutions", institutions)
}
