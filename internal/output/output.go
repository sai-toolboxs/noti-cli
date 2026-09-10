package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"text/tabwriter"
)

// Format represents an output format.
type Format string

const (
	FormatText  Format = "text"
	FormatJSON  Format = "json"
	FormatMD    Format = "md"
	FormatTable Format = "table"
)

// Writer handles output in different formats.
type Writer struct {
	format    Format
	jsonMode  bool
	stdout    io.Writer
	stderr    io.Writer
	tabWriter *tabwriter.Writer
}

// New creates a new Writer.
func New(format Format) *Writer {
	w := &Writer{
		format:   format,
		jsonMode: format == FormatJSON,
		stdout:   os.Stdout,
		stderr:   os.Stderr,
	}
	if format == FormatTable {
		w.tabWriter = tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	}
	return w
}

// NewWithWriters creates a Writer with custom output writers.
func NewWithWriters(format Format, stdout, stderr io.Writer) *Writer {
	w := &Writer{
		format:   format,
		jsonMode: format == FormatJSON,
		stdout:   stdout,
		stderr:   stderr,
	}
	if format == FormatTable {
		w.tabWriter = tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	}
	return w
}

// Print writes a line to stdout (unless in JSON mode and it's diagnostic).
func (w *Writer) Print(args ...interface{}) {
	if w.jsonMode {
		return
	}
	if w.tabWriter != nil {
		fmt.Fprintln(w.tabWriter, args...)
		return
	}
	fmt.Fprintln(w.stdout, args...)
}

// Printf writes a formatted line to stdout.
func (w *Writer) Printf(format string, args ...interface{}) {
	if w.jsonMode {
		return
	}
	if w.tabWriter != nil {
		fmt.Fprintf(w.tabWriter, format, args...)
		return
	}
	fmt.Fprintf(w.stdout, format, args...)
}

// PrintJSON writes a JSON object to stdout.
func (w *Writer) PrintJSON(v interface{}) {
	enc := json.NewEncoder(w.stdout)
	enc.SetIndent("", "  ")
	enc.Encode(v)
}

// PrintRawJSON writes raw JSON bytes to stdout.
func (w *Writer) PrintRawJSON(data []byte) {
	w.stdout.Write(data)
	w.stdout.Write([]byte("\n"))
}

// Diag writes a diagnostic message to stderr.
func (w *Writer) Diag(args ...interface{}) {
	fmt.Fprintln(w.stderr, args...)
}

// Diagf writes a formatted diagnostic to stderr.
func (w *Writer) Diagf(format string, args ...interface{}) {
	fmt.Fprintf(w.stderr, format, args...)
}

// Flush flushes the tab writer if active.
func (w *Writer) Flush() {
	if w.tabWriter != nil {
		w.tabWriter.Flush()
	}
}

// Format returns the current output format.
func (w *Writer) Format() Format {
	return w.format
}

// IsJSON reports whether the output format is JSON.
func (w *Writer) IsJSON() bool {
	return w.jsonMode
}

// PrintTableHeader writes a header row for table output.
func (w *Writer) PrintTableHeader(headers ...string) {
	if w.jsonMode {
		return
	}
	for i, h := range headers {
		if i > 0 {
			fmt.Fprint(w.stdout, "\t")
		}
		fmt.Fprint(w.stdout, h)
	}
	fmt.Fprintln(w.stdout)
}

// PrintTableRow writes a data row for table output.
func (w *Writer) PrintTableRow(values ...string) {
	if w.jsonMode {
		return
	}
	if w.tabWriter != nil {
		for i, v := range values {
			if i > 0 {
				fmt.Fprint(w.tabWriter, "\t")
			}
			fmt.Fprint(w.tabWriter, v)
		}
		fmt.Fprintln(w.tabWriter)
		return
	}
	for i, v := range values {
		if i > 0 {
			fmt.Fprint(w.stdout, "\t")
		}
		fmt.Fprint(w.stdout, v)
	}
	fmt.Fprintln(w.stdout)
}
