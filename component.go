package injection

import (
	"fmt"
	"path"
	"reflect"
	"strings"
)

type tagField struct {
	name  string
	fType reflect.Type
	fVal  reflect.Value
}

type componentShell struct {
	name      string
	realType  reflect.Type  // underlying type
	realValue reflect.Value // underlying value
	template  interface{}
	fields    []*tagField
}

func newComponentShell(name string, realVal reflect.Value, realType reflect.Type, template interface{}) *componentShell {
	return &componentShell{
		name:      name,
		realType:  realType,
		realValue: realVal,
		template:  template,
	}
}

var zeroValue = reflect.Value{}

// getQualifiedTypeName get full type name of stub, including the full package path
func getQualifiedTypeName(stub interface{}) string {
	var t reflect.Type
	switch stub.(type) {
	case reflect.StructField:
		t = stub.(reflect.StructField).Type
	case reflect.Type:
		t = stub.(reflect.Type)
	case reflect.Value:
		rv := stub.(reflect.Value)
		uv := reflect.Indirect(stub.(reflect.Value))
		if uv.IsValid() {
			t = uv.Type()
		} else {
			// stub is zero value
			t = rv.Type()
		}
	default:
		panic(fmt.Errorf("unsupported stub type '%s', expected reflect.Type, reflect.StructField or reflect.Value", reflect.TypeOf(stub)))
	}
	if t.Kind() == reflect.Ptr {
		// if t is pointer type, return its underlying type
		// and if t is the zero value of a type, this call won't panic
		t = t.Elem()
	}
	pkg := t.PkgPath()
	parts := strings.Split(t.String(), ".")
	return path.Join(pkg, parts[len(parts)-1])
}

func getFieldDepName(field reflect.StructField) string {
	v, ok := field.Tag.Lookup("dep")
	if !ok {
		return ""
	}
	if v != "" {
		return v
	}
	return getQualifiedTypeName(field)
}

// isInterfaceType check if the field type is interface
func isInterfaceType(field reflect.Type) bool {
	return field.Kind() == reflect.Interface
}

func createComponentInstance(cShell *componentShell) (componentPtr interface{}, err error) {
	defer func() {
		if e := recover(); e != nil {
			componentPtr = nil
			err = fmt.Errorf("%v", e)
		}
	}()
	realVal := cShell.realValue
	constructor := realVal.MethodByName("Constructor")
	if constructor == zeroValue {
		return cShell.template, nil
	}
	retVals := constructor.Call(nil)
	if count := len(retVals); count > 1 {
		return nil, fmt.Errorf("too many return values, expect: %d, actual: %d", 1, count)
	}
	return retVals[0].Interface(), nil
}

func initInterfaceComponent(
	t reflect.Type,
	dic *DIContainer,
	pathMap map[string]bool,
) (interface{}, error) {
	var impls []*componentShell

	depGraph := dic.depGraph
	for _, cShell := range depGraph {
		if reflect.PtrTo(cShell.realType).Implements(t) {
			impls = append(impls, cShell)
		}
	}
	if impls == nil {
		return nil, fmt.Errorf("couldn't find implementation for interface '%s'", getQualifiedTypeName(t))
	} else if len(impls) > 1 {
		names := make([]string, len(impls))
		for i, cShell := range impls {
			names[i] = cShell.name
		}
		return nil, fmt.Errorf("multiple implementations for interface '%s':\n  %s\n"+
			"(you may not use embedded fields in struct to solve this problem)",
			getQualifiedTypeName(t),
			strings.Join(names, "\n  "))
	}

	impl := impls[0]
	if pathMap[impl.name] {
		return nil, fmt.Errorf("cycle dependency detected, '%s' and '%s' are depend on each other", impl.name, impl.name)
	}
	return initComponent(impl.name, dic, pathMap)
}

func initComponent(
	name string,
	dic *DIContainer,
	pathMap map[string]bool,
) (interface{}, error) {
	depGraph := dic.depGraph
	components := dic.components
	cShell := depGraph[name]

	pathMap[name] = true
	for _, tf := range cShell.fields {
		depName := tf.name
		if _, ok := components[depName]; !ok {
			if pathMap[depName] { // cycle detected
				return nil, fmt.Errorf("cycle dependency detected, '%s' and '%s' are depend on each other", name, depName)
			}
			var componentPtr interface{}
			var err error
			if isInterfaceType(tf.fType) {
				componentPtr, err = initInterfaceComponent(tf.fType, dic, pathMap)
			} else if depGraph[depName] == nil {
				eType := tf.fType.Elem()
				shell := reflect.New(eType).Interface()
				dic.Register(shell)
				componentPtr, err = initComponent(depName, dic, pathMap)
			} else {
				componentPtr, err = initComponent(depName, dic, pathMap)
			}
			if err != nil {
				return nil, err
			}
			components[depName] = componentPtr
		}
		componentPtr := components[depName]
		if depVal := reflect.ValueOf(componentPtr); !depVal.Type().AssignableTo(tf.fType) {
			return nil, fmt.Errorf("'%s' is not assignable to '%s'", getQualifiedTypeName(depVal), getQualifiedTypeName(tf.fType))
		}
		tf.fVal.Set(reflect.ValueOf(componentPtr))
	}
	// Constructor is value type receiver's to call
	componentPtr, err := createComponentInstance(cShell)
	if err != nil {
		return nil, fmt.Errorf("failed to call Constructor of '%s': %v", name, err)
	}
	components[name] = componentPtr
	pathMap[name] = false
	return componentPtr, nil
}
