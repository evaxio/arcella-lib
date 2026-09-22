package basic

import (
	"axgit.vixiv.ru/snake/arcella-lib/utils"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"log/slog"

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
	decoder      ConfigManagerDecoder
	cfg          interface{}
	decodeErrors []error
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
		if fileName, err = bc.getDefaultDirConfigFileName(); err != nil {
			return err
		}
	}
	if fileName != "" && utils.FileExists(fileName) {
		if err = bc.loadYaml(fileName); err != nil {
			return err
		}
	}

	// Process values
	bc.modifyStruct()

	// return results
	return errors.Join(bc.decodeErrors...)
}

// File Config ///////////////////////////////////////////////////

func (bc *BasicConfig) loadYaml(fileName string) error {
	yamlFile, err := os.ReadFile(fileName)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(yamlFile, bc.cfg)
}

func (bc *BasicConfig) getEnvCfgVariable() string {
	for _, arg := range os.Args[1:] {
		if strings.HasPrefix(arg, argCfgFile) {
			return arg[len(argCfgFile):]
		}
	}
	return os.Getenv(cfgPrefix) // load from ENV
}

func (bc *BasicConfig) getDefaultDirConfigFileName() (string, error) {
	ex, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(ex), defaultFileName), nil
}

// Work with data ///////////////////////////////////////////////////

func (bc *BasicConfig) modifyStruct() {
	inputValue := reflect.ValueOf(bc.cfg)

	if inputValue.Kind() != reflect.Ptr {
		panic("ModifyStruct requires a pointer to struct")
	}
	elem := inputValue.Elem()
	if elem.Kind() != reflect.Struct {
		panic("ModifyStruct requires a pointer to struct")
	}

	// Modify in place: preserves unexported fields and avoids deep-copying the whole tree
	for i := 0; i < elem.NumField(); i++ {
		field := elem.Field(i)
		if !field.CanSet() {
			continue
		}
		structField := elem.Type().Field(i)

		modifiedFieldValue := bc.modifyValue(field, structField.Tag)

		if modifiedFieldValue.IsValid() && modifiedFieldValue.Type().AssignableTo(field.Type()) {
			field.Set(modifiedFieldValue)
		}
	}
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
		_ = bc.modifyValue(elem, stag)
		return value

	case reflect.Slice:
		if value.IsNil() {
			return value
		}
		for i := 0; i < value.Len(); i++ {
			elem := value.Index(i)
			if !elem.CanSet() {
				continue
			}
			modifiedElem := bc.modifyValue(elem, stag)
			if modifiedElem.IsValid() {
				elem.Set(modifiedElem)
			}
		}
		return value

	case reflect.Array:
		for i := 0; i < value.Len(); i++ {
			elem := value.Index(i)
			if !elem.CanSet() {
				continue
			}
			modifiedElem := bc.modifyValue(elem, stag)
			if modifiedElem.IsValid() {
				elem.Set(modifiedElem)
			}
		}
		return value

	case reflect.Map:
		if value.IsNil() {
			return value
		}
		iter := value.MapRange()
		for iter.Next() {
			// modify only values (keys are left as is)
			modifiedVal := bc.modifyValue(iter.Value(), stag)
			if modifiedVal.IsValid() {
				value.SetMapIndex(iter.Key(), modifiedVal)
			}
		}
		return value

	case reflect.Struct:
		for i := 0; i < value.NumField(); i++ {
			field := value.Field(i)
			if !field.CanSet() {
				continue
			}
			structField := value.Type().Field(i)
			modifiedField := bc.modifyValue(field, structField.Tag)
			if modifiedField.IsValid() {
				field.Set(modifiedField)
			}
		}
		return value

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
			slog.Error("Decode config value", "value", val, "error", err)
			bc.decodeErrors = append(bc.decodeErrors, err)
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
