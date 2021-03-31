package ini

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/zeebo/assert"
)

func TestRead(t *testing.T) {
	for _, test := range tests {
		var got []Entry
		assert.NoError(t, Read(test.Reader(), func(ent Entry) error {
			got = append(got, ent)
			return nil
		}))
		assert.DeepEqual(t, got, test.Entries)
	}
}

func TestWrite_RoundTrip(t *testing.T) {
	for _, test := range tests {
		var got []Entry
		var buf bytes.Buffer

		assert.NoError(t, Write(&buf, func(emit func(ent Entry)) {
			for _, ent := range test.Entries {
				emit(ent)
			}
		}))
		assert.Equal(t, test.NormalizedData(), strings.TrimSpace(buf.String()))

		assert.NoError(t, Read(&buf, func(ent Entry) error {
			got = append(got, ent)
			return nil
		}))
		assert.DeepEqual(t, got, test.Entries)
	}
}

//
// test cases
//

type testCase struct {
	Data    string
	Entries []Entry
}

func (t testCase) NormalizedData() string {
	data := strings.Split(t.Data, "\n")
	inMultilineComment := false
	for i, v := range data {
		v = strings.TrimPrefix(v, "\t\t")
		if (len(v) > 0 && v[0] == '#') || inMultilineComment {
			inMultilineComment = v[len(v)-1] == '\\'
			v = ""
		}
		data[i] = v
	}
	return strings.TrimSpace(strings.Join(data, "\n"))
}

func (t testCase) Reader() io.Reader {
	// the data has every line prefixed with \t\t to make the
	// test definitions easier to read so we trim them off here
	data := strings.Split(t.Data, "\n")
	for i, v := range data {
		data[i] = strings.TrimPrefix(v, "\t\t")
	}
	return strings.NewReader(strings.Join(data, "\n"))
}

var tests = []testCase{
	{``, nil},

	{`=`, []Entry{{Key: "", Value: ""}}},

	{`
		foo = bar
	`, []Entry{
		{Key: "foo", Value: "bar"},
	}},

	{`
		[table]
		foo = bar
		baz = bif
	`, []Entry{
		{Section: "table", Key: "foo", Value: "bar"},
		{Section: "table", Key: "baz", Value: "bif"},
	}},

	{`
		# a comment
		foo = bar
	`, []Entry{
		{Key: "foo", Value: "bar"},
	}},

	{`
		# multi line \
		comment
		foo = bar
	`, []Entry{
		{Key: "foo", Value: "bar"},
	}},

	{`
		foo = bar\
		multi line
	`, []Entry{
		{Key: "foo", Value: "bar\nmulti line"},
	}},

	{`
		foo = bar\
			multi line with whitespace
	`, []Entry{
		{Key: "foo", Value: "bar\n\tmulti line with whitespace"},
	}},

	{`
		[multi line\
		table]
		foo = bar
	`, []Entry{
		{Section: "multi line\ntable", Key: "foo", Value: "bar"},
	}},

	{`
		# multiple
		# comments
		foo = bar
	`, []Entry{
		{Key: "foo", Value: "bar"},
	}},

	{`
		# empty lines are ignored

		foo = bar

	`, []Entry{
		{Key: "foo", Value: "bar"},
	}},

	{`
		[table1]
		foo = bar

		[table2]
		foo = bar
	`, []Entry{
		{Section: "table1", Key: "foo", Value: "bar"},
		{Section: "table2", Key: "foo", Value: "bar"},
	}},

	{`
		[table1]
		foo = bar

		[]
		foo = reset table
	`, []Entry{
		{Section: "table1", Key: "foo", Value: "bar"},
		{Key: "foo", Value: "reset table"},
	}},
}
