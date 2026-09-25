package basic

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/evaxio/arcella-lib/utils"

	"gopkg.in/yaml.v3"
)

const (
	defaultFileName = "config.yml"
	cfgPrefix       = "cfg"
	argCfgFile      = cfgPrefix + "="
	//
	tagDefault = "default"
	tagEnv     = "env"
	//
	encodedPrefix  = "ENC("
	encodedPostfix = ")"
)

type ConfigManagerDecoder interface {
	Decode(plainText string) (string, error)
}

type BasicConfig struct {
	decoder ConfigManagerDecoder
	cfg     interface{}
}

func NewBasicConfigWithDecoder(cfg any, inDecoder ConfigManagerDecoder) *BasicConfig {
	return &BasicConfig{cfg: cfg, decoder: inDecoder}
}

func NewBasicConfig(cfg any) *BasicConfig {
	return &BasicConfig{cfg: cfg, decoder: NewDecoder()}
}

// Load loading config from file or env:
//  1. Searching config file in app arguments and current folder
//  2. Setting default values
//  3. Loading values from env
func (bc *BasicConfig) Load() error {
	var err error

	// Load config from file
	var fileName string
	if fileName = bc.getEnvCfgVariable(); fileName == "" {
		fileName = bc.getDefaultDirConfigFileName()
	}
	if fileName != "" && utils.FileExists(fileName) {
		if err = bc.loadYaml(fileName); err != nil {
			return err
		}
	}

	// Process values
	bc.modifyStruct()

	// return results
	return err
}

///////////////////////////////////////////////////////////////////////

func (bc *BasicConfig) loadYaml(fileName string) error {
	if yamlFile, err := os.ReadFile(fileName); err == nil {
		return yaml.Unmarshal(yamlFile, bc.cfg)
	} else {
		return err
	}
}

func (bc *BasicConfig) getEnvCfgVariable() string {
	for _, arg := range os.Args[1:] {
		if strings.HasPrefix(arg, argCfgFile) {
			return arg[len(argCfgFile):]
		}
	}
	return os.Getenv(cfgPrefix) // load from ENV
}

func (bc *BasicConfig) getDefaultDirConfigFileName() string {
	if ex, err := os.Executable(); err == nil {
		return filepath.Dir(ex) + filepath.Join(string(os.PathSeparator), defaultFileName)
	} else {
		panic("getDefaultDirConfigFileName: " + err.Error())
	}
	return ""
}

// Work with data ///////////////////////////////////////////////////

func (bc *BasicConfig) modifyStruct() {
	inputValue := reflect.ValueOf(bc.cfg)
	if inputValue.Kind() != reflect.Ptr {
		panic(fmt.Sprintf("ModifyStruct requires a pointer to struct: (got %v)", inputValue.Kind()))
	}
	elem := inputValue.Elem()

	if elem.Kind() != reflect.Struct {
		panic(fmt.Sprintf("ModifyStruct requires a pointer to struct: (got %v)", elem.Kind().String()))
	}

	modifiedValue := reflect.New(elem.Type()).Elem()

	// Copy & Modify
	for i := 0; i < elem.NumField(); i++ {
		field := elem.Field(i)
		copyField := modifiedValue.Field(i)
		structField := elem.Type().Field(i)

		modifiedFieldValue := bc.modifyValue(field, structField.Tag)

		if modifiedFieldValue.IsValid() && modifiedFieldValue.Type().AssignableTo(copyField.Type()) {
			copyField.Set(modifiedFieldValue)
		}
	}

	elem.Set(modifiedValue) // replace the original structure with a modified one
}

func (bc *BasicConfig) modifyValue(value reflect.Value, stag reflect.StructTag) reflect.Value {
	switch value.Kind() {
	case reflect.String:
		return reflect.ValueOf(bc.processString(value, stag))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return reflect.ValueOf(bc.processInt(value, stag)).Convert(value.Type())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return reflect.ValueOf(bc.processUInt(value, stag)).Convert(value.Type())
	case reflect.Float32, reflect.Float64:
		return reflect.ValueOf(bc.processFloat(value, stag)).Convert(value.Type())
	case reflect.Bool:
		return reflect.ValueOf(bc.processBool(value, stag))

	case reflect.Ptr:
		if value.IsNil() {
			// Если указатель nil, возвращаем nil
			return value
		}
		elem := value.Elem()
		modifiedElem := bc.modifyValue(elem, stag)
		if modifiedElem.IsValid() {
			newPtr := reflect.New(modifiedElem.Type())
			newPtr.Elem().Set(modifiedElem)
			return newPtr
		}
		return value

	case reflect.Slice:
		if value.IsNil() {
			return value
		}
		newSlice := reflect.MakeSlice(value.Type(), value.Len(), value.Cap())
		for i := 0; i < value.Len(); i++ {
			elem := value.Index(i)
			modifiedElem := bc.modifyValue(elem, stag)
			if modifiedElem.IsValid() {
				newSlice.Index(i).Set(modifiedElem)
			}
		}
		return newSlice

	case reflect.Array:
		newArray := reflect.New(value.Type()).Elem()
		for i := 0; i < value.Len(); i++ {
			elem := value.Index(i)
			modifiedElem := bc.modifyValue(elem, stag)
			if modifiedElem.IsValid() {
				newArray.Index(i).Set(modifiedElem)
			}
		}
		return newArray

	case reflect.Map:
		if value.IsNil() {
			return value
		}
		newMap := reflect.MakeMap(value.Type())
		iter := value.MapRange()
		for iter.Next() {
			key := iter.Key()
			val := iter.Value()

			// do we need modify key ?
			// modifiedKey := bc.modifyValue(key, stag)
			// or only value ?
			modifiedVal := bc.modifyValue(val, stag)

			//if modifiedKey.IsValid() && modifiedVal.IsValid() {
			if key.IsValid() && modifiedVal.IsValid() {
				// newMap.SetMapIndex(modifiedKey, modifiedVal)
				newMap.SetMapIndex(key, modifiedVal)
			}
		}
		return newMap

	case reflect.Struct:
		newStruct := reflect.New(value.Type()).Elem()
		for i := 0; i < value.NumField(); i++ {
			field := value.Field(i)
			structField := value.Type().Field(i)
			modifiedField := bc.modifyValue(field, structField.Tag)
			if modifiedField.IsValid() {
				// fmt.Printf("set %v\n", structField.Name)
				newStruct.Field(i).Set(modifiedField)
			}
		}
		return newStruct

	default: // as is by default
		return value
	}
}

func (bc *BasicConfig) processString(value reflect.Value, stag reflect.StructTag) string {
	val := value.String()
	if defValue, ok := stag.Lookup(tagDefault); ok && bc.isZero(value) { // default values
		val = defValue
	}
	if envName, ok := stag.Lookup(tagEnv); ok { // values from env
		if envValue := os.Getenv(envName); envValue != "" {
			val = envValue
		}
	}
	if strings.HasPrefix(val, encodedPrefix) && strings.HasSuffix(val, ")") {
		if decVal, err := bc.decoder.Decode(val[len(encodedPrefix) : len(val)-len(encodedPostfix)]); err != nil {
			fmt.Printf("Decode value %s, error: %s\n", val, err.Error())
		} else {
			val = decVal
		}
	}
	return val
}

func (bc *BasicConfig) processInt(value reflect.Value, stag reflect.StructTag) any {
	val := value.Int()
	if defValue, ok := stag.Lookup(tagDefault); ok && bc.isZero(value) { // default values
		if valInt, err := strconv.ParseInt(defValue, 10, 64); err == nil {
			val = valInt
		} else if valDur, err := time.ParseDuration(defValue); err == nil { // duration case ?
			val = valDur.Nanoseconds()
		}
	}
	if envName, ok := stag.Lookup(tagEnv); ok { // values from env
		if envValue := os.Getenv(envName); envValue != "" {
			if valInt, err := strconv.ParseInt(envValue, 10, 64); err == nil {
				val = valInt
			}
		}
	}
	return val
}

func (bc *BasicConfig) processUInt(value reflect.Value, stag reflect.StructTag) any {
	val := value.Uint()
	if defValue, ok := stag.Lookup(tagDefault); ok && bc.isZero(value) { // default values
		if valInt, err := strconv.ParseUint(defValue, 10, 64); err == nil {
			val = valInt
		}
	}
	if envName, ok := stag.Lookup(tagEnv); ok { // values from env
		if envValue := os.Getenv(envName); envValue != "" {
			if valInt, err := strconv.ParseUint(envValue, 10, 64); err == nil {
				val = valInt
			}
		}
	}
	return val
}

func (bc *BasicConfig) processFloat(value reflect.Value, stag reflect.StructTag) any {
	val := value.Float()
	if defValue, ok := stag.Lookup(tagDefault); ok && bc.isZero(value) { // default values
		if valF64, err := strconv.ParseFloat(defValue, 64); err == nil {
			val = valF64
		}
	}
	if envName, ok := stag.Lookup(tagEnv); ok { // values from env
		if envValue := os.Getenv(envName); envValue != "" {
			if valF64, err := strconv.ParseFloat(envValue, 64); err == nil {
				val = valF64
			}
		}
	}
	return val
}

func (bc *BasicConfig) processBool(value reflect.Value, stag reflect.StructTag) any {
	val := value.Bool()
	if defValue, ok := stag.Lookup(tagDefault); ok && bc.isZero(value) { // default values
		if valBool, err := strconv.ParseBool(defValue); err == nil {
			val = valBool
		}
	}
	if envName, ok := stag.Lookup(tagEnv); ok { // values from env
		if envValue := os.Getenv(envName); envValue != "" {
			if valBool, err := strconv.ParseBool(envValue); err == nil {
				val = valBool
			}
		}
	}
	return val
}

// isZero reports whether v is its zero value for its type.
func (bc *BasicConfig) isZero(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	case reflect.Slice, reflect.Array:
		return v.Len() == 0
	case reflect.Struct:
		if t, ok := v.Interface().(time.Time); ok {
			return t.IsZero()
		}
		return false
	case reflect.Invalid:
		return true
	default:
		return v.IsZero()
	}
}
