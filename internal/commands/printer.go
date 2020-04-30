package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"time"
)

func Print(w io.Writer, view interface{}) error {
	if IsJSONFormat() {
		viewJson, err := json.Marshal(view)
		if err != nil {
			return err
		}
		fmt.Fprintln(w, string(viewJson))
	} else if IsPrettyFormat() {
		entities := toEntities(view)
		printList(w, entities)
	}
	return nil
}

type entity struct {
	Properties []property
}

type property struct {
	Key   string
	Value interface{}
}

func printList(w io.Writer, entities []*entity) {
	maxColumn := entities[0].MaxKey()
	format := fmt.Sprintf("%%-%ds : %%v\n", maxColumn)
	for _, e := range entities {
		for _, p := range e.Properties {
			fmt.Fprintf(w, format, p.Key, p.Value)
		}
		fmt.Fprintln(w)
	}
}

func (e entity) MaxKey() int {
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
		field := s.Type().Field(i)
		format := field.Tag.Get("format")
		if format == "-" {
			continue
		}
		p := property{
			Key:   field.Name,
			Value: s.Field(i).Interface(),
		}
		if format != "" {
			for _, f := range strings.Split(format, ";") {
				formatter := getFormatter(f)
				formatter(&p)
			}
		}
		e.Properties = append(e.Properties, p)
	}
	return &e
}

func getFormatter(name string) func(*property) {
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

func parseMaxlen(name string) func(*property) {
	mlStr := name[len("maxlen:"):]
	mlVal, err := strconv.Atoi(mlStr)
	if err != nil {
		panic("bad format tag " + name)
	}
	return func(p *property) {
		val := fmt.Sprint(p.Value)
		if len(val) > mlVal {
			p.Value = val[:mlVal]
		}
	}
}

func parseTime(name string) func(*property) {
	timeFmt := name[len("time:"):]
	return func(p *property) {
		t, ok := p.Value.(time.Time)
		if !ok {
			tp, ok := p.Value.(*time.Time)
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

func parseName(name string) func(*property) {
	keyName := name[len("name:"):]
	return func(p *property) {
		p.Key = keyName
	}
}
