package ldap

import (
	"bytes"
	"fmt"
	log "log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata"

	"github.com/go-ldap/ldap/v3"
)

var guidAlnumRe = regexp.MustCompile(`[^a-zA-Z0-9]+`)

var guidStripReplacer = strings.NewReplacer("{", "", "}", "", "-", "")

var novosibirskLoc = loadNovosibirskLoc()

func loadNovosibirskLoc() *time.Location {
	loc, err := time.LoadLocation("Asia/Novosibirsk")
	if err != nil {
		log.Error("LoadLocation", log.String("location", "Asia/Novosibirsk"), log.String("message", err.Error()))
		return time.Local
	}
	return loc
}

const (
	CpgDateTime = "2006-01-02 15:04:05"
	// CpgDateTimeT = "2006-01-02T15:04:05.000"
	CpgDateTimeT = "2006-01-02T15:04:05.000Z" // "2025-04-19T06:11:01.000Z",
	//
	UF_NORMAL_ACCOUNT   = 0x0200
	UF_PASSWORD_EXPIRED = 0x800000
)

func IsGUIDValidSize(guid string) bool {
	return len(guidAlnumRe.ReplaceAllString(guid, "")) == 32
}

func GuidToOctetString(guid string) string {
	idxs := []int{3, 2, 1, 0, 5, 4, 7, 6, 8, 9, 10, 11, 12, 13, 14, 15}
	strip := guidStripReplacer.Replace(guid)
	if len(strip) != 32 {
		return ""
	}
	var buffer bytes.Buffer
	for _, idx := range idxs {
		buffer.WriteString("\\" + strip[(idx*2):(idx*2)+2])
	}
	return buffer.String()
}

func ConvertAttributeToStringArray(attribute *ldap.EntryAttribute, withT bool) []string {
	switch strings.ToLower(attribute.Name) {
	case "thumbnailphoto":
		return []string{""}
	case "objectguid":
		if len(attribute.ByteValues) == 0 {
			return []string{""}
		}
		return []string{ConvertToDashedString(attribute.ByteValues[0])}
	case "objectsid":
		if len(attribute.ByteValues) == 0 {
			return []string{""}
		}
		return []string{decodeSID(attribute.ByteValues[0])}
	case "useraccountcontrol":
		return []string{getUserAccountControl(attribute.Values[0])}
	case "whencreated", "whenchanged":
		if withT {
			return []string{getDateFromStringT(attribute.Values[0])}
		} else {
			return []string{getDateFromString(attribute.Values[0])}
		}
	case "pwdlastset", "lastlogon", "lastlogontimestamp", "accountexpires":
		return []string{getTSFromLong(attribute.Values[0])}
	case "grouptype":
		return []string{getGroupType(attribute.Values[0])}
	default:
		return attribute.Values
	}

}
func ConvertToDashedString(guid []byte) string {
	if guid != nil && len(guid) >= 16 {
		return fmt.Sprintf("%02X%02X%02X%02X-%02X%02X-%02X%02X-%02X%02X-%02X%02X%02X%02X%02X%02X",
			guid[3]&255,
			guid[2]&255,
			guid[1]&255,
			guid[0]&255,
			guid[5]&255,
			guid[4]&255,
			guid[7]&255,
			guid[6]&255,
			guid[8]&255,
			guid[9]&255,
			guid[10]&255,
			guid[11]&255,
			guid[12]&255,
			guid[13]&255,
			guid[14]&255,
			guid[15]&255)
	}
	return ""
}

func decodeSID(b []byte) string {
	const size = 4
	if b != nil && len(b) >= 8 {
		sb := strings.Builder{}
		sb.WriteString("S-")
		sb.WriteString(strconv.Itoa(int(b[0])))
		authority := 0
		for i := 2; i <= 7; i++ {
			authority = authority | int(b[i])<<(8*(5-(i-2)))
		}
		sb.WriteString("-")
		sb.WriteString(strings.ToUpper(strconv.FormatInt(int64(authority), 16)))
		offset := 8
		subAuthorityCount := int(b[1]) & 0xFF
		for i := 0; i < subAuthorityCount; i++ {
			if offset+size > len(b) {
				return ""
			}
			var subAuthority int
			for k := 0; k < size; k++ {
				subAuthority = subAuthority | (int(b[offset+k])&0xFF)<<(8*k)
			}
			sb.WriteString("-")
			sb.WriteString(strconv.Itoa(subAuthority))
			offset += size
		}
		return sb.String()
	}
	return ""

}

func getUserAccountControl(str string) string {
	switch str {
	case "512":
		return "Enabled Account"
	case "514":
		return "Disabled Account"
	case "544":
		return "Enabled, Password Not Required"
	case "546":
		return "Disabled, Password Not Required"
	case "66048":
		return "Enabled, Password Doesn't Expire"
	case "66050":
		return "Disabled, Password Doesn't Expire"
	case "66080":
		return "Enabled, Password Doesn't Expire & Not Required"
	case "66082":
		return "Disabled, Password Doesn't Expire & Not Required"
	case "262656":
		return "Enabled, Smartcard Required"
	case "262658":
		return "Disabled, Smartcard Required"
	case "262688":
		return "Enabled, Smartcard Required, Password Not Required"
	case "262690":
		return "Disabled, Smartcard Required, Password Not Required"
	case "328192":
		return "Enabled, Smartcard Required, Password Doesn't Expire"
	case "328194":
		return "Disabled, Smartcard Required, Password Doesn't Expire"
	case "328224":
		return "Enabled, Smartcard Required, Password Doesn't Expire & Not Required"
	case "328226":
		return "Disabled, Smartcard Required, Password Doesn't Expire & Not Required"
	default:
		return str
	}
}

func getDateFromString(s string) string { // 04.03.2024 12:18:04
	if t, err := time.ParseInLocation("20060102150405Z", s, novosibirskLoc); err == nil {
		return t.Format(CpgDateTime)
	}
	return s
}

func getDateFromStringT(s string) string {
	if t, err := time.Parse("20060102150405Z", s); err == nil {
		return t.Format(CpgDateTimeT)
	} // RFC3339
	return s
}

func getTSFromLong(s string) string {
	if s != "" && s != "0" && s != strconv.FormatInt(accountExpiresNever, 10) {
		if val, err := strconv.ParseInt(s, 10, 64); err == nil {
			dt := time.UnixMilli(val/10000 - 11644473600000)
			return dt.Format(CpgDateTime)
		}
	}
	return s
}

func getGroupType(s string) string {
	switch s {
	case "2":
		return "Global group"
	case "4":
		return "Domain local group"
	case "8":
		return "Universal group"
	case "-2147483648":
		return "Security group"
	case "-2147483646":
		return "Global Security Group"
	case "-2147483644":
		return "Local Security Group"
	case "-2147483643":
		return "BuiltIn Group"
	case "-2147483640":
		return "Universal Security Group"
	default:
		return s
	}
}
