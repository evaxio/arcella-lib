package utils

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// StrToInt converts str to an int; parse errors are swallowed and 0 is
// returned.
func StrToInt(str string) int {
	val, _ := strconv.Atoi(str)
	return val
}

func StrToInt32Default(str string, defaultValue int32) int32 {
	val, err := strconv.ParseInt(str, 10, 32)
	if err != nil {
		return defaultValue
	}
	return int32(val)
}

func StrToInt64Default(str string, defaultValue int64) int64 {
	val, err := strconv.Atoi(str)
	if err != nil {
		return defaultValue
	}
	return int64(val)
}

func StrToUInt64Default(str string, defaultValue uint64) uint64 {
	val, err := strconv.ParseUint(str, 10, 64)
	if err != nil {
		return defaultValue
	}
	return val
}

func StrToFloat32Default(str string, defaultValue float32) float32 {
	value, err := strconv.ParseFloat(strings.ReplaceAll(str, ",", "."), 32)
	if err != nil {
		return defaultValue
	}
	return float32(value)
}

func Int64ToStr(i int64) string {
	return strconv.FormatInt(i, 10)
}

func UInt64ToStr(i uint64) string {
	return strconv.FormatUint(i, 10)
}

func UInt32ToStr(i uint32) string {
	return strconv.FormatUint(uint64(i), 10)
}

func UInt16ToStr(i uint16) string {
	return strconv.Itoa(int(i))
}

func Int16ToStr(i int16) string {
	return strconv.Itoa(int(i))
}

func Int32ToStr(i int32) string {
	return strconv.FormatInt(int64(i), 10)
}

func IntToStr(i int) string {
	return strconv.Itoa(i)
}

func Int8ToStr(i int8) string {
	return IntToStr(int(i))
}

// Trim removes ALL whitespace from str, including internal whitespace, not
// just the leading/trailing edges.
func Trim(str string) string {
	return strings.Join(strings.Fields(str), "")
}

func GetSubStr(content string, startsWith string, endsWith string) string {
	startPos := strings.Index(content, startsWith)
	if startPos > -1 {
		startPos = startPos + len(startsWith)
		var endPos int
		if endsWith == "" {
			endPos = len(content) - startPos
		} else {
			endPos = strings.Index(content[startPos:], endsWith)
		}
		if endPos != -1 {
			return content[startPos : startPos+endPos]
		}
	}
	return ""
}

func SubStr(content string, start int, length int) string {
	total, startByte, endByte := 0, 0, 0
	for i := 0; i < len(content); {
		if total == start {
			startByte = i
		}
		if total == start+length {
			endByte = i
		}
		_, size := utf8.DecodeRuneInString(content[i:])
		total++
		i += size
	}
	if start > total {
		return ""
	}
	if start < 0 || length < 0 {
		// the original []rune-based implementation panicked on these inputs
		startByte, endByte = 0, -1
	} else {
		if start == total {
			return ""
		}
		if start+length >= total {
			endByte = len(content)
		}
	}
	return content[startByte:endByte]
}

var numbersAndCommaRe = regexp.MustCompile(`[^,0123456789]`)

func NumbersAndComma(inStr string) string {
	return numbersAndCommaRe.ReplaceAllString(inStr, "")
}

// Flat64ToStr formats a float64 (strconv.FormatFloat 'f' format).
func Flat64ToStr(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

// Flat32ToStr formats a float32 (strconv.FormatFloat 'f' format).
func Flat32ToStr(value float32) string {
	return strconv.FormatFloat(float64(value), 'f', -1, 32)
}

func Reverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// var nonAlphanumericRegex = regexp.MustCompile(`[^a-zA-Z0-9 ]+`)
var nonAlphanumericRegex = regexp.MustCompile(`[^a-zA-Z0-9~@#$^*()_+={}|\\,.?: -]+`)

func StripRegex(str string) string {
	return nonAlphanumericRegex.ReplaceAllString(str, "")
}

var numOnlyRegex = regexp.MustCompile(`[^0-9]+`)

func NumOnly(str string) string {
	return numOnlyRegex.ReplaceAllString(str, "")
}

func RemoveUnreadable(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsGraphic(r) {
			return r
		}
		return -1
	}, s)
}
