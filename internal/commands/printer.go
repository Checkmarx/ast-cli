package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/spf13/viper"
)

func Print(w io.Writer, view interface{}) error {
	if IsJSONFormat() {
		viewJSON, err := json.Marshal(view)
		if err != nil {
			return err
		}
		fmt.Fprintln(w, string(viewJSON))
	} else if IsPrettyFormat() {
		entities := toEntities(view)
		printList(w, entities)
	} else if IsTableFormat() {
		entities := toEntities(view)
		printTable(w, entities)
	} else {
		return errors.Errorf("Invalid format %s", viper.GetString(formatFlag))
	}
	return nil
}

func IsJSONFormat() bool {
	return strings.EqualFold(viper.GetString(formatFlag), formatJSON)
}

func IsPrettyFormat() bool {
	return strings.EqualFold(viper.GetString(formatFlag), formatPretty)
}

func IsTableFormat() bool {
	return strings.EqualFold(viper.GetString(formatFlag), formatTable)
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
	fmt.Fprintln(w)
	maxColumn := entities[0].maxKey()
	format := fmt.Sprintf("%%-%ds : %%v\n", maxColumn)
	for _, e := range entities {
		for _, p := range e.Properties {
			fmt.Fprintf(w, format, p.Key, p.Value)
		}
		fmt.Fprintln(w)
	}
}

func printTable(w io.Writer, entities []*entity) {
	if len(entities) == 0 {
		return
	}
	fmt.Fprintln(w)
	colWidth := getColumnWidth(entities)
	// print header
	for i := 0; i < len(colWidth); i++ {
		key := entities[0].Properties[i].Key
		fmt.Fprint(w, key, pad(colWidth[i], key))
	}
	fmt.Fprintln(w)
	// print delimiter
	for i := 0; i < len(colWidth); i++ {
		line := strings.Repeat("-", len(entities[0].Properties[i].Key))
		fmt.Fprint(w, line, pad(colWidth[i], line))
	}
	fmt.Fprintln(w)
	// print rows by columns
	for _, e := range entities {
		for i := 0; i < len(colWidth); i++ {
			val := e.Properties[i].Value
			fmt.Fprint(w, val, pad(colWidth[i], val))
		}
		fmt.Fprintln(w)
	}
	fmt.Fprintln(w)
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
	for i := 0; i < s.NumField(); i++ {
		p, ok := newProperty(s, i)
		if ok {
			e.Properties = append(e.Properties, p)
		}
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
		p.Value = t.Format(timeFmt)
	}
}

func parseName(name string) func(*property, interface{}) {
	keyName := name[len("name:"):]
	return func(p *property, _ interface{}) {
		p.Key = keyName
	}
}
