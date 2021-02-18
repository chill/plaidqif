package qif

import (
	"bytes"
	"testing"
	"time"
)

func TestWriteHeader(t *testing.T) {
	var out bytes.Buffer
	w := NewWriter(&out, "testAcct", "CCard", "02/01/2006")

	if err := w.writeHeader(); err != nil {
		t.Fatal(err)
	}

	expect := `!Account
NtestAcct
TCCard
^
!Type:CCard`

	if got := out.String(); got != expect {
		t.Fatalf("expected:\n%s\n\ngot:\n%s", expect, got)
	}
}

func TestWriteTransaction(t *testing.T) {
	tests := []struct {
		Name   string
		Tx     Transaction
		Expect string
	}{
		{
			Name: "WithMemo",
			Tx: Transaction{
				Date:   time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
				Payee:  "testPayee",
				Amount: 10.26,
				Memo:   "abcdef",
			},
			Expect: `
D01/01/2020
PtestPayee
T-10.26
Mabcdef
^`,
		},
		{
			Name: "WithoutMemo",
			Tx: Transaction{
				Date:   time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
				Payee:  "testPayee",
				Amount: 10.26,
			},
			Expect: `
D01/01/2020
PtestPayee
T-10.26
^`,
		},
	}

	for _, tst := range tests {
		tst := tst
		t.Run(tst.Name, func(t *testing.T) {
			var out bytes.Buffer
			w := NewWriter(&out, "testAcct", "CCard", "02/01/2006")

			if err := w.writeTransaction(tst.Tx); err != nil {
				t.Fatal(err)
			}

			if got := out.String(); got != tst.Expect {
				t.Fatalf("expected:\n%s\n\ngot:\n%s", tst.Expect, got)
			}
		})
	}
}

func TestWriteTransactions(t *testing.T) {
	var out bytes.Buffer
	w := NewWriter(&out, "testAcct", "CCard", "02/01/2006")

	err := w.WriteTransactions([]Transaction{
		{
			Date:   time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
			Payee:  "testPayee1",
			Amount: 10.26,
			Memo:   "testMemo",
		},
		{
			Date:   time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC),
			Payee:  "testPayee2",
			Amount: -5001.67,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	expect := `!Account
NtestAcct
TCCard
^
!Type:CCard
D01/01/2020
PtestPayee1
T-10.26
MtestMemo
^
D02/01/2020
PtestPayee2
T5001.67
^`

	if got := out.String(); got != expect {
		t.Fatalf("expected:\n%s\n\ngot:\n%s", expect, got)
	}
}
