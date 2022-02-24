package printer

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
)

const (
	FormatJSON           = "json"
	FormatSarif          = "sarif"
	FormatSonar          = "sonar"
	FormatSummary        = "summaryHTML"
	FormatSummaryJSON    = "summaryJSON"
	FormatSummaryConsole = "summaryConsole"
	FormatList           = "list"
	FormatTable          = "table"
	FormatHTML           = "html"
)

func Print(w io.Writer, view interface{}, format string) error {
	if IsFormat(format, FormatJSON) {
		viewJSON, err := json.Marshal(view)
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintln(w, string(viewJSON))
	} else if IsFormat(format, FormatList) {
		entities := toEntities(view)
		printList(w, entities)
	} else if IsFormat(format, FormatTable) {
		entities := toEntities(view)
		printTable(w, entities)
	} else {
		return errors.Errorf("Invalid format %s", format)
	}
	return nil
}

func IsFormat(val, format string) bool {
	return strings.EqualFold(val, format)
}

type entity struct {
	Properties []property
}

type property struct {
	Key   string
	Value string
}

func printList(w io.Writer, entities []*entity) {
	if len(entities) == 0 {
		return
	}
	columnReformat(entities)
	_, _ = fmt.Fprintln(w)
	maxColumn := entities[0].maxKey()
	format := fmt.Sprintf("%%-%ds : %%v\n", maxColumn)
	for _, e := range entities {
		for _, p := range e.Properties {
			if columnFilters(p.Key) {
				_, _ = fmt.Fprintf(w, format, p.Key, p.Value)
			}
		}
		_, _ = fmt.Fprintln(w)
	}
}

func columnFilters(key string) bool {
	return key != "Updated at"
}

func columnReformat(entities []*entity) {
	// Dates should just not contain HH:MM:SS
	for _, e := range entities {
		for i := 0; i < len(e.Properties); i++ {
			key := e.Properties[i].Key
			if key == "Created at" {
				e.Properties[i].Value = e.Properties[i].Value[0:8]
			}
		}
	}
}

func printTable(w io.Writer, entities []*entity) {
	if len(entities) == 0 {
		return
	}
	columnReformat(entities)
	_, _ = fmt.Fprintln(w)
	colWidth := getColumnWidth(entities)
	// print header
	for i := 0; i < len(colWidth); i++ {
		key := entities[0].Properties[i].Key
		if columnFilters(key) {
			_, _ = fmt.Fprint(w, key, pad(colWidth[i], key))
		}
	}
	_, _ = fmt.Fprintln(w)
	// print delimiter
	for i := 0; i < len(colWidth); i++ {
		key := entities[0].Properties[i].Key
		line := strings.Repeat("-", len(entities[0].Properties[i].Key))
		if columnFilters(key) {
			_, _ = fmt.Fprint(w, line, pad(colWidth[i], line))
		}
	}
	_, _ = fmt.Fprintln(w)
	// print rows by columns
	for _, e := range entities {
		for i := 0; i < len(colWidth); i++ {
			val := e.Properties[i].Value
			key := entities[0].Properties[i].Key
			if columnFilters(key) {
				_, _ = fmt.Fprint(w, val, pad(colWidth[i], val))
			}
		}
		_, _ = fmt.Fprintln(w)
	}
	_, _ = fmt.Fprintln(w)
}

func pad(width int, key string) string {
	padLen := width - len(key) + 1
	const nonBreakingSpace = string('\u00A0')
	return strings.Repeat(nonBreakingSpace, padLen)
}

func getColumnWidth(entities []*entity) []int {
	result := make([]int, len(entities[0].Properties))
	for i := range result {
		result[i] = len(entities[0].Properties[i].Key)
		for _, e := range entities {
			disp := e.Properties[i].Value
			if len(disp) > result[i] {
				result[i] = len(disp)
			}
		}
	}
	return result
}

func (e entity) maxKey() int {
	max := 0
	for _, p := range e.Properties {
		if len(p.Key) > max {
			max = len(p.Key)
		}
	}
	return max
}

func toEntities(view interface{}) []*entity {
	var entities []*entity
	viewVal := reflect.ValueOf(view)
	if viewVal.Kind() == reflect.Slice {
		for i := 0; i < viewVal.Len(); i++ {
			e := newEntity(viewVal.Index(i))
			entities = append(entities, e)
		}
	} else {
		e := newEntity(viewVal)
		entities = append(entities, e)
	}
	return entities
}

func newEntity(v reflect.Value) *entity {
	s := reflect.Indirect(v)
	e := entity{}
	if s.Kind() == reflect.Struct {
		for i := 0; i < s.NumField(); i++ {
			p, ok := newProperty(s, i)
			if ok {
				e.Properties = append(e.Properties, p)
			}
		}
	} else {
		e.Properties = append(
			e.Properties, property{
				Key:   "---",
				Value: fmt.Sprint(v.String()),
			},
		)
	}

	return &e
}

func newProperty(s reflect.Value, i int) (property, bool) {
	typeField := s.Type().Field(i)
	format := typeField.Tag.Get("format")
	if format == "-" {
		return property{}, false
	}
	valueField := s.Field(i)

	p := property{
		Key:   typeField.Name,
		Value: fmt.Sprint(valueField.Interface()),
	}

	if valueField.Kind() == reflect.Map {
		p.Value = p.Value[3:] // remove 'map'
	}

	if format != "" {
		for _, f := range strings.Split(format, ";") {
			if f == "omitempty" {
				if valueField.IsZero() {
					return property{}, false
				}

				continue
			}

			format := getFormatter(f)
			format(&p, valueField.Interface())
		}
	}
	return p, true
}

func getFormatter(name string) func(*property, interface{}) {
	if strings.HasPrefix(name, "maxlen:") {
		return parseMaxlen(name)
	}

	if strings.HasPrefix(name, "time:") {
		return parseTime(name)
	}

	if strings.HasPrefix(name, "name:") {
		return parseName(name)
	}

	panic("unknown format " + name)
}

func parseMaxlen(name string) func(*property, interface{}) {
	mlStr := name[len("maxlen:"):]
	mlVal, err := strconv.Atoi(mlStr)
	if err != nil {
		panic("bad format tag " + name)
	}
	return func(p *property, raw interface{}) {
		if len(p.Value) > mlVal {
			p.Value = p.Value[:mlVal]
		}
	}
}

func parseTime(name string) func(*property, interface{}) {
	timeFmt := name[len("time:"):]
	return func(p *property, raw interface{}) {
		t, ok := raw.(time.Time)
		if !ok {
			tp, ok := raw.(*time.Time)
			if !ok {
				panic("time tag can be applied only to time.Time or *time.Time")
			}
			if tp == nil {
				p.Value = ""
				return
			}
			t = *tp
		}

		localTime, err := time.LoadLocation("Local")
		if err == nil {
			t = t.In(localTime)
		}

		p.Value = t.Format(timeFmt)
	}
}

func parseName(name string) func(*property, interface{}) {
	keyName := name[len("name:"):]
	return func(p *property, _ interface{}) {
		p.Key = keyName
	}
}
