package algorithm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// Values holds parsed parameter values by name.
type Values map[string]any

// holds reports whether a condition, like Param.OnlyIf, holds for v. An
// empty condition always holds.
func (v Values) holds(condition string) bool {
	if condition == "" {
		return true
	}
	if name, value, ok := strings.Cut(condition, "="); ok {
		return v.Choice(name) == value
	}
	return v.Bool(strings.TrimPrefix(condition, "!")) != strings.HasPrefix(condition, "!")
}

// ParamValues lists parameter values in order. It encodes as a JSON
// object, keeping the order.
type ParamValues []ParamValue

// ParamValue is the value of one parameter.
type ParamValue struct {
	Name  string
	Value any
}

func (pv ParamValues) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, p := range pv {
		if i > 0 {
			buf.WriteByte(',')
		}
		name, err := json.Marshal(p.Name)
		if err != nil {
			return nil, err
		}
		value, err := json.Marshal(p.Value)
		if err != nil {
			return nil, err
		}
		buf.Write(name)
		buf.WriteByte(':')
		buf.Write(value)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// UnmarshalJSON reads parameter values from a JSON object, keeping their
// order. Numbers come out as float64s.
func (pv *ParamValues) UnmarshalJSON(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	if tok, err := dec.Token(); err != nil || tok != json.Delim('{') {
		return fmt.Errorf("parameter values must be a JSON object")
	}
	*pv = ParamValues{}
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return err
		}
		var value any
		if err := dec.Decode(&value); err != nil {
			return err
		}
		*pv = append(*pv, ParamValue{Name: tok.(string), Value: value})
	}
	_, err := dec.Token()
	return err
}

func (v Values) Int(name string) int       { return v[name].(int) }
func (v Values) Float(name string) float64 { return v[name].(float64) }
func (v Values) Bool(name string) bool     { return v[name].(bool) }
func (v Values) Choice(name string) string { return v[name].(string) }
