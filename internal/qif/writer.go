package qif

import (
	"io"
	"strconv"
	"text/template"
	"time"
)

// QIF spec: https://web.archive.org/web/20100222214101/http://web.intuit.com/support/quicken/docs/d_qif.html

// Writer is not safe for concurrent use
type Writer struct {
	w           io.Writer
	dateFormat  string
	header      header
	wroteHeader bool
	err         error
}

type header struct {
	Name string
	Type string
}

type Transaction struct {
	Date   time.Time
	Payee  string
	Amount float64
}

type transaction struct {
	Date   string
	Payee  string
	Amount string
}

const headerFmt = `!Account
N{{.Name}}
T{{.Type}}
^
!Type:{{.Type}}`

// txFmt intentionally starts with a newline
const txFmt = `
D{{.Date}}
P{{.Payee}}
T{{.Amount}}
^`

var (
	headerTemplate = template.Must(template.New("headerFmt").Parse(headerFmt))
	txTemplate     = template.Must(template.New("txFmt").Parse(txFmt))
)

// NewWriter returns a Writer which is not safe for concurrent use
func NewWriter(w io.Writer, accountName, accountQIFType, dateFormat string) *Writer {
	return &Writer{
		w:          w,
		dateFormat: dateFormat,
		header: header{
			Name: accountName,
			Type: accountQIFType,
		},
	}
}

func (w *Writer) writeHeader() error {
	if w.wroteHeader {
		return nil
	}

	w.wroteHeader = true
	if err := headerTemplate.Execute(w.w, w.header); err != nil {
		w.err = err
		return w.err
	}

	return nil
}

func (w *Writer) WriteTransactions(transactions []Transaction) error {
	if w.err != nil {
		return w.err
	}

	for _, tx := range transactions {
		if err := w.WriteTransaction(tx); err != nil {
			return err
		}
	}

	return nil
}

func (w *Writer) WriteTransaction(tx Transaction) error {
	if w.err != nil {
		return w.err
	}

	if err := w.writeHeader(); err != nil {
		return err
	}

	return w.writeTransaction(tx)
}

func (w *Writer) writeTransaction(tx Transaction) error {
	transaction := transaction{
		Date:   tx.Date.Format(w.dateFormat),
		Payee:  tx.Payee,
		Amount: strconv.FormatFloat(-tx.Amount, 'f', 2, 64),
	}

	if err := txTemplate.Execute(w.w, transaction); err != nil {
		w.err = err
		return err
	}

	return nil
}
