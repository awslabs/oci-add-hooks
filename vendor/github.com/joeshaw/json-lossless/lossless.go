// json-lossless populates structs from JSON and allows serialization
// back to JSON without losing fields that are not explicitly defined
// in the struct.
//
// Source code: https://github.com/joeshaw/json-lossless
package lossless

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"

	"github.com/bitly/go-simplejson"
)

// The JSON type contains the state of the decoded data.  Embed this
// type in your type and implement MarshalJSON and UnmarshalJSON
// methods to add lossless encoding and decoding.
//
// Example:
//
//    type Person struct {
//        lossless.JSON `json:"-"`
//
//        Name    string
//        Age     int
//        Address string
//    }
//
//    func (p *Person) UnmarshalJSON(data []byte) error {
//        return p.JSON.UnmarshalJSON(p, data)
//    }
//
//    func (p Person) MarshalJSON() ([]byte, error) {
//        return p.JSON.MarshalJSON(p)
//    }
type JSON struct {
	json *simplejson.Json
}

func (js *JSON) maybeInit() {
	if js.json == nil {
		js.json, _ = simplejson.NewJson([]byte("{}"))
	}
}

// Sets a JSON value not represented in the struct type.  The
// argument list is a set of strings referring to the JSON path,
// with the value to be set as the last value.
//
// Example:
//
//    // This sets {"Phone": {"Mobile": "614-555-1212"}} in the JSON
//    p.Set("Phone", "Mobile", "614-555-1212")
func (js *JSON) Set(args ...interface{}) error {
	js.maybeInit()

	if len(args) < 2 {
		return errors.New("rs must contain a path and value")
	}

	v := args[len(args)-1]
	key, ok := args[len(args)-2].(string)
	if !ok {
		return errors.New("all args except last must be strings")
	}

	j := js.json
	for _, p := range args[:len(args)-2] {
		strp, ok := p.(string)
		if !ok {
			return errors.New("all args except last must be strings")
		}

		newj, ok := j.CheckGet(strp)
		if !ok {
			j.Set(strp, make(map[string]interface{}))
			j = j.Get(strp)
		} else {
			j = newj
		}
	}

	j.Set(key, v)

	return nil
}

// Unmarshals JSON data into the given destination.  Users should
// call this from their type's UnmarshalJSON method.
//
// Example:
//
//     func (p *Person) UnmarshalJSON(data []byte) error {
//         return p.JSON.UnmarshalJSON(p, data)
//     }
func (js *JSON) UnmarshalJSON(dest interface{}, data []byte) error {
	j, err := simplejson.NewJson(data)
	if err != nil {
		return err
	}

	js.json = j
	return syncToStruct(dest, j)
}

// Marshals the given source into JSON data.  Users should
// call this from their type's MarshalJSON method.
//
// Example:
//
//     func (p Person) MarshalJSON() ([]byte, error) {
//         return p.JSON.MarshalJSON(p)
//     }
func (js *JSON) MarshalJSON(src interface{}) ([]byte, error) {
	js.maybeInit()
	err := syncFromStruct(src, js.json)
	if err != nil {
		return nil, err
	}

	return json.Marshal(js.json)
}

func syncToStruct(dest interface{}, j *simplejson.Json) error {
	dv := reflect.Indirect(reflect.ValueOf(dest))
	dt := dv.Type()

	// Probably a good candidate for future caching
	tagmap := make(map[string]string)
	for i := 0; i < dt.NumField(); i++ {
		sf := dt.Field(i)
		tag := sf.Tag.Get("json")
		if tag == "-" {
			continue
		}

		tagmap[sf.Name] = sf.Name
		if tag == "" {
			tagmap[strings.ToLower(sf.Name)] = sf.Name
		} else {
			tagmap[tag] = sf.Name
		}
	}

	m, err := j.Map()
	if err != nil {
		return err
	}

	for k, v := range m {
		name, ok := tagmap[k]
		if !ok {
			continue
		}

		f := dv.FieldByName(name)
		if !f.IsValid() {
			continue
		}

		if reflect.TypeOf(v) == f.Type() {
			f.Set(reflect.ValueOf(v))
		} else {
			// If the default encoding/json decoded type does
			// not match our target type -- for instance, a
			// time.Time that was parsed as a string but we
			// want to store it in a time.Time field --
			// re-marshal and unmarshal it into the target
			// type.  Gross, yes.
			marsh, err := json.Marshal(v)
			if err != nil {
				return err
			}
			fv := f.Addr().Interface()
			err = json.Unmarshal(marsh, fv)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func syncFromStruct(src interface{}, j *simplejson.Json) error {
	dv := reflect.Indirect(reflect.ValueOf(src))
	dt := dv.Type()

	// This skips the encoding/json "json" tag's "omitempty"
	// value.
	for i := 0; i < dt.NumField(); i++ {
		sf := dt.Field(i)
		tag := sf.Tag.Get("json")
		if tag == "-" {
			continue
		}

		var name string
		if tag == "" {
			name = sf.Name
		} else {
			name = tag
		}

		f := dv.Field(i)
		j.Set(name, f.Interface())
	}

	return nil
}
