package ers

import (
	"fmt"
	"strings"

	"github.com/davecgh/go-spew/spew"
)

var (
	// T 関数は, NewTrace 関数のエイリアス.
	T = NewTrace
)

type Trace struct {
	Text   string
	Values []any
}

func NewTrace(src any) *Trace {
	switch v := src.(type) {
	case string:
		return &Trace{Text: v}
	case []byte:
		return &Trace{Text: string(v)}
	case error:
		return &Trace{Text: v.Error()}
	case *Trace:
		if v != nil {
			return &Trace{Text: v.Text, Values: v.Values}
		}
	case Trace:
		return &v
	}
	return &Trace{Text: fmt.Sprintf("%s", src)}
}

func (t *Trace) Dump() string {
	elems := []string{t.Text}
	if t.Values != nil {
		dump := (&spew.ConfigState{
			MaxDepth:                2,
			Indent:                  "  ",
			DisableMethods:          true,
			DisablePointerMethods:   true,
			DisableCapacities:       true,
			DisablePointerAddresses: true,
		}).Sdump(t.Values...)
		elems = append(elems, dump)
	}
	return strings.Join(elems, ": ")
}
