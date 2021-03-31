// Package ini is a minimal markup language for config files
//
// Specification:
//
// Entries:
//
// 0. an entry contains three fields
//    a. section of type string
//    a. key of type string
//    b. value of type string
//    c. comment of type string
//
// Parser state:
//
// 0. a section of type string, initially empty
// 1. a comment of type string, initially empty
//
// Parser semantics:
//
// 0. the byte stream is broken up into lines
//    a. lines are split by the separator regex '\r?\n'
//    b. a separator may be escaped with '\' causing it not to split
//    c. an escaping '\' is removed from the contents of the line
//    d. the line is always joined with '\n'
//
// 1. lines beginning with '#' are comments and are ignored
//
// 2. empty space trimmed lines are valid and ignored
//
// 3. lines beginning with '[' and ending with ']' are section declarations
//    a. the line is invalid if the contents contain '[', ']', '\', '=', or '#'
//    b. the contents between the '[' and ']' become the section
//    c. the comment state is reset to empty
//
// 4. lines containing the string "=" are entries
//    a. the entry key is the space trimmed portion before the first "="
//    b. the entry value is the space trimmed portion after the first "="
//    c. the comment state has the final '\n' removed, if it exists
//    d. entries are immediately emitted
//    e. when an entry is emitted, the comment state is reset to empty
//
// 5. anything else is an invalid line
//    a. invalid lines causes Read to return an error
package ini

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/zeebo/errs/v2"
)

type Entry struct {
	Section string
	Key     string
	Value   string
}

func Read(r io.Reader, cb func(ent Entry) error) error {
	var linebuf []byte = make([]byte, 0, 64)
	var ent Entry

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		linebuf = append(linebuf, scanner.Bytes()...)

		if len(linebuf) == 0 || len(bytes.TrimSpace(linebuf)) == 0 {
			continue
		}

		if linebuf[len(linebuf)-1] == '\\' {
			linebuf[len(linebuf)-1] = '\n'
			continue
		}

		if linebuf[0] == '#' {
			linebuf = linebuf[:0]
			continue
		}

		if linebuf[0] == '[' && linebuf[len(linebuf)-1] == ']' {
			ent.Section = string(linebuf[1 : len(linebuf)-1])
			linebuf = linebuf[:0]
			continue
		}

		if idx := bytes.IndexByte(linebuf, '='); idx >= 0 {
			ent.Key = string(bytes.TrimSpace(linebuf[:idx]))
			ent.Value = string(bytes.TrimSpace(linebuf[idx+1:]))
			if err := cb(ent); err != nil {
				return err
			}
			linebuf = linebuf[:0]
			continue
		}

		return errs.Tag("invalid line").Errorf("%q", linebuf)
	}

	return scanner.Err()
}

type errWriter struct {
	err error
	w   io.Writer
}

func (e *errWriter) Write(p []byte) (n int, err error) {
	if e.err != nil {
		return 0, e.err
	}
	n, e.err = e.w.Write(p)
	return n, e.err
}

func Write(w io.Writer, cb func(emit func(ent Entry))) error {
	var section string
	var wrote bool
	ew := &errWriter{w: w}

	cb(func(ent Entry) {
		if ent.Section != section {
			if wrote {
				fmt.Fprintln(ew)
			}
			fmt.Fprintf(ew, "[%s]\n", escape(ent.Section))
			section = ent.Section
		}
		if len(ent.Key) > 0 {
			fmt.Fprintf(ew, "%s ", escape(ent.Key))
		}
		fmt.Fprint(ew, "=")
		if len(ent.Value) > 0 {
			fmt.Fprintf(ew, " %s", escape(ent.Value))
		}
		fmt.Fprint(ew, "\n")

		wrote = true
	})

	return ew.err
}

func escape(x string) string {
	return strings.ReplaceAll(x, "\n", "\\\n")
}
