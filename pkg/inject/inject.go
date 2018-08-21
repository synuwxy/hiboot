// Copyright 2018 John Deng (hi.devops.io@gmail.com).
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package inject

import (
	"reflect"
	"errors"
	"strings"
	"github.com/hidevopsio/hiboot/pkg/log"
	"github.com/hidevopsio/hiboot/pkg/starter"
	"github.com/hidevopsio/hiboot/pkg/utils/reflector"
	"github.com/hidevopsio/hiboot/pkg/utils/str"
	"github.com/hidevopsio/hiboot/pkg/utils/io"
)


const (
	injectIdentifier = "inject"
	valueIdentifier = "value"
	initMethodName = "Init"
)

var (
	autoConfiguration   starter.Factory
	NotImplementedError = errors.New("[inject] interface is not implemented")
	InvalidObjectError      = errors.New("[inject] invalid object")
	UnsupportedInjectionTypeError      = errors.New("[inject] unsupported injection type")
	IllegalArgumentError = errors.New("[inject] input argument type can not be the same as receiver")
	TagIsAlreadyExistError = errors.New("[inject] tag is already exist")
	TagIsNilError = errors.New("[inject] tag is nil")
	InvalidTagNameError = errors.New("[inject] invalid tag name, e.g. exampleTag")

	tagsContainer map[string]Tag
)

func init() {
	autoConfiguration = starter.GetFactory()
	tagsContainer = make(map[string]Tag)
}

// AddTag
func AddTag(tag Tag) error {
	name := reflector.ParseObjectName(tag, "Tag")
	if name == "" {
		return InvalidTagNameError
	}

	t := tagsContainer[name]
	if t != nil {
		return TagIsAlreadyExistError
	}
	if tag != nil {
		tagsContainer[name] = tag
	}
	return nil
}

func getInstanceByName(instances map[string]interface{}, name string, instType reflect.Type) (inst interface{}) {
	name = str.LowerFirst(name)
	inst = instances[name]

	// TODO: we should pro load all candidates into instances for improving performance.
	// if inst is nil, and the object type is an interface
	// then try to find the instance that embedded with the interface
	if inst == nil && instType.Kind() == reflect.Interface {
		for _, ist := range instances {
			//log.Debug(n)
			if ist != nil && reflector.HasEmbeddedField(ist, instType.Name()) {
				inst = ist
				break
			}
		}
	}
	return
}

func saveInstance(instances map[string]interface{}, name string, inst interface{}) {
	name = str.LowerFirst(name)
	instances[name] = inst
	return
}

// IntoObject injects instance into the tagged field with `inject:"instanceName"`
func IntoObject(object reflect.Value) error {
    var err error

	obj := reflector.Indirect(object)
	if obj.Kind() != reflect.Struct {
		log.Errorf("object: %v", object)
		return InvalidObjectError
	}

	instances := autoConfiguration.Instances()

	// field injection
	for _, f := range reflector.DeepFields(object.Type()) {
		//log.Debugf("parent: %v, name: %v, type: %v, tag: %v", obj.Type(), f.Name, f.Type, f.Tag)
		// check if object has value field to be injected
		var injectedObject interface{}

		ft := f.Type
		if f.Type.Kind() == reflect.Ptr {
			ft = ft.Elem()
		}

		// set field object
		var fieldObj reflect.Value
		if obj.IsValid() && obj.Kind() == reflect.Struct {
			fieldObj = obj.FieldByName(f.Name)
		}

		// TODO: assume that the f.Name of value and inject tag is not the same
		injectedObject = getInstanceByName(instances, f.Name, f.Type)
		if injectedObject == nil {
			for tagName, tagObject := range tagsContainer {
				tag, ok := f.Tag.Lookup(tagName)
				if ok {
					injectedObject = tagObject.Decode(object, f, tag)
					if injectedObject != nil {
						if tagObject.IsSingleton() {
							saveInstance(instances, f.Name, injectedObject)
						}
						// ONLY one tag should be used for dependency injection
						break
					}
				}
			}
		}

		if injectedObject != nil && fieldObj.CanSet() {
			fov := reflect.ValueOf(injectedObject)
			fieldObj.Set(fov)
			log.Debugf("Injected %v.(%v) into %v.%v", injectedObject, fov.Type(), obj.Type(), f.Name)
		}

		//log.Debugf("- kind: %v, %v, %v, %v", obj.Kind(), object.Type(), fieldObj.Type(), f.Name)
		//log.Debugf("isValid: %v, canSet: %v", fieldObj.IsValid(), fieldObj.CanSet())
		filedObject := reflect.Indirect(fieldObj)
		filedKind := filedObject.Kind()
		canNested := filedKind == reflect.Struct
		if canNested && fieldObj.IsValid() && fieldObj.CanSet() && filedObject.Type() != obj.Type() {
			err = IntoObject(fieldObj)
		}
	}

	// method injection
	// Init, Setter
	method, ok := object.Type().MethodByName(initMethodName)
	if ok {
		numIn := method.Type.NumIn()
		inputs := make([]reflect.Value, numIn)
		inputs[0] = obj.Addr()
		injectByMethod := true
		for i := 1; i < numIn; i++ {
			inType := reflector.IndirectType(method.Type.In(i))
			var paramValue reflect.Value
			inTypeName := inType.Name()
			pkgName := io.DirName(inType.PkgPath())
			//log.Debugf("pkg: %v", pkgName)
			inst := getInstanceByName(instances, inTypeName, inType)
			if inst == nil {
				alternativeName := strings.Title(pkgName) + inTypeName
				inst = getInstanceByName(instances, alternativeName, inType)
			}
			if inst == nil {
				if inType.Kind() == reflect.Interface {
					injectByMethod = false
					break
				} else {
					paramValue = reflect.New(inType)
					inst = paramValue.Interface()
					saveInstance(instances, inTypeName, inst)
				}
			}

			if inst != nil {
				paramValue = reflect.ValueOf(inst)
			}
			inputs[i] = paramValue

			//log.Debugf("inType: %v, name: %v, instance: %v", inType, inTypeName, inst)
			//log.Debugf("kind: %v == %v, %v, %v ", obj.Kind(), reflect.Struct, paramValue.IsValid(), paramValue.CanSet())
			paramObject := reflect.Indirect(paramValue)
			if paramObject.Type() != obj.Type() && paramObject.Kind() == reflect.Struct && paramValue.IsValid() {
				err = IntoObject(paramValue)
			}
		}
		// finally call Init method to inject
		if injectByMethod {
			method.Func.Call(inputs)
		}
	}

	return err
}

