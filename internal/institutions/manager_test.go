package institutions

import (
	"reflect"
	"testing"
	"time"
)

func TestInstitutionManager_GetInstitutions(t *testing.T) {
	im, err := NewInstitutionManager("./", "test_institutions.json")
	if err != nil {
		t.Fatalf("failed to setup institution manager: %v", err)
	}

	ins, err := im.GetInstitutions([]string{})
	if err != nil {
		t.Fatalf("failed to get institutions: %v", err)
	}

	// we expect the output to be sorted even if the input wasn't
	expect := []Institution{
		{
			Name:           "regular",
			AccessToken:    "test-key-regular",
			ItemID:         "abcdef-regular",
			ConsentExpires: time.Date(2024, 2, 27, 18, 47, 04, 0, time.UTC),
		},
		{
			Name:           "regular-three",
			AccessToken:    "test-key-regular-three",
			ItemID:         "abcdef-regular-three",
			ConsentExpires: time.Date(2024, 2, 4, 9, 50, 59, 0, time.UTC),
		},
		{
			Name:           "regular-two",
			AccessToken:    "test-key-regular-two",
			ItemID:         "abcdef-regular-two",
			ConsentExpires: time.Date(2024, 2, 27, 18, 48, 32, 0, time.UTC),
		},
		{
			Name:           "test-with\u0026s",
			AccessToken:    "test-key-test-with\u0026s",
			ItemID:         "abcdef-test-with\u0026s",
			ConsentExpires: time.Date(2024, 2, 27, 18, 45, 27, 0, time.UTC),
		},
	}

	if !reflect.DeepEqual(ins, expect) {
		t.Fatalf("mismatch in institutions expected\nhave: %+v\nwant: %+v", ins, expect)
	}
}
