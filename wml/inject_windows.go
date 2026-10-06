package wml

import (
	"fmt"
	"reflect"
	"strings"
)

// Inject fills the exported fields of the struct that into points to. A field
// with the tag `wml:"Name"` receives the window or control with that Name:
//
//	type form struct {
//	    Window *wui.Window   `wml:"main"`
//	    User   *wui.EditLine `wml:"user"`
//	    Note   *wui.Label    `wml:"note,optional"`
//	}
//
// The field type must fit the element: a *wui.Button field cannot receive a
// Label. A field of type wui.Control or interface{} accepts any control.
// Fields without a tag are left alone. A named element that does not exist is
// an error unless the tag ends with ",optional".
//
// All problems are reported together in one error.
func (v *View) Inject(into interface{}) error {
	rv := reflect.ValueOf(into)
	if rv.Kind() != reflect.Ptr || rv.IsNil() || rv.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("wml: Inject needs a non-nil pointer to a struct, got %T", into)
	}
	sv := rv.Elem()
	st := sv.Type()

	var problems []string
	for i := 0; i < st.NumField(); i++ {
		f := st.Field(i)
		tag, ok := f.Tag.Lookup("wml")
		if !ok || tag == "" || tag == "-" {
			continue
		}
		name := tag
		optional := false
		if idx := strings.Index(tag, ","); idx >= 0 {
			name = tag[:idx]
			optional = strings.TrimSpace(tag[idx+1:]) == "optional"
		}

		if f.PkgPath != "" {
			problems = append(problems, fmt.Sprintf("field %s is unexported and cannot be set", f.Name))
			continue
		}
		obj, found := v.byName[name]
		if !found {
			if !optional {
				problems = append(problems, fmt.Sprintf("field %s: there is no element with Name %q", f.Name, name))
			}
			continue
		}
		ov := reflect.ValueOf(obj)
		if !ov.Type().AssignableTo(f.Type) {
			problems = append(problems, fmt.Sprintf("field %s has type %s but %q is a %s", f.Name, f.Type, name, ov.Type()))
			continue
		}
		sv.Field(i).Set(ov)
	}

	if len(problems) > 0 {
		return fmt.Errorf("wml: cannot fill %s: %s", st, strings.Join(problems, "; "))
	}
	return nil
}
