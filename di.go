package injection

import (
	"fmt"
	"log"
	"reflect"
)

// DIContainer Dependency injection container, not thread safe
type DIContainer struct {
	depGraph   map[string]*componentShell
	components map[string]interface{}
}

// NewDIContainer create new DI container
func NewDIContainer() *DIContainer {
	return &DIContainer{
		depGraph:   make(map[string]*componentShell),
		components: make(map[string]interface{}),
	}
}

// Register register component to DI container by its type name
func (dic *DIContainer) Register(shell interface{}) {
	ptrVal := reflect.ValueOf(shell)
	realVal := reflect.Indirect(ptrVal)
	realType := realVal.Type()
	typeName := getQualifiedTypeName(realType)

	defer func() {
		if err := recover(); err != nil {
			log.Printf("Error occurred while registering component '%s': %v", typeName, err)
			panic(err)
		}
	}()

	// check availablity
	if ptrVal.Kind() != reflect.Ptr {
		panic(fmt.Errorf("shell parameter must be of pointer type"))
	}
	if realType.Kind() != reflect.Struct {
		panic(fmt.Errorf("component must be of Struct type"))
	}

	cShell := newComponentShell(typeName, realVal, realType, shell)
	n := realType.NumField()
	for i := 0; i < n; i++ {
		// field to get tag data
		sField := realType.Field(i)
		// field to set field value
		fieldVal := realVal.Field(i)
		if depName := getFieldDepName(sField); depName != "" {
			if !fieldVal.CanSet() {
				panic(fmt.Errorf("field '%s' should be exported", sField.Name))
			}
			if !isInterfaceType(sField.Type) {
				// if field type is concrete type, register it recursively
				eType := sField.Type.Elem()
				dic.Register(reflect.New(eType).Interface())
			}
			tf := &tagField{name: depName, fType: sField.Type, fVal: fieldVal}
			cShell.fields = append(cShell.fields, tf)
		}
	}
	// if component has no dependency, initialize it first
	if len(cShell.fields) == 0 {
		initComponent(typeName, dic.depGraph, dic.components, make(map[string]bool))
	}
	dic.depGraph[typeName] = cShell
}

// Get return component from DI container by qualified type name, initialization may be needed
func (dic *DIContainer) Get(name string) (interface{}, error) {
	components := dic.components
	if c, ok := components[name]; ok {
		return c, nil
	}
	// record initialization path
	pathMap := make(map[string]bool)
	pathMap[name] = true
	c, err := initComponent(name, dic.depGraph, dic.components, pathMap)
	if err != nil {
		return nil, fmt.Errorf("Error occurred while injecting dependency of '%s':\n  %v", name, err)
	}
	return c, nil
}
