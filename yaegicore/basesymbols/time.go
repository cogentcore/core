// Code generated by 'yaegi extract time'. DO NOT EDIT.

//go:build go1.22
// +build go1.22

package basesymbols

import (
	"go/constant"
	"go/token"
	"reflect"
	"time"
)

func init() {
	Symbols["time/time"] = map[string]reflect.Value{
		// function, constant and variable definitions
		"ANSIC":                  reflect.ValueOf(constant.MakeFromLiteral("\"Mon Jan _2 15:04:05 2006\"", token.STRING, 0)),
		"After":                  reflect.ValueOf(time.After),
		"AfterFunc":              reflect.ValueOf(time.AfterFunc),
		"April":                  reflect.ValueOf(time.April),
		"August":                 reflect.ValueOf(time.August),
		"Date":                   reflect.ValueOf(time.Date),
		"DateOnly":               reflect.ValueOf(constant.MakeFromLiteral("\"2006-01-02\"", token.STRING, 0)),
		"DateTime":               reflect.ValueOf(constant.MakeFromLiteral("\"2006-01-02 15:04:05\"", token.STRING, 0)),
		"December":               reflect.ValueOf(time.December),
		"February":               reflect.ValueOf(time.February),
		"FixedZone":              reflect.ValueOf(time.FixedZone),
		"Friday":                 reflect.ValueOf(time.Friday),
		"Hour":                   reflect.ValueOf(time.Hour),
		"January":                reflect.ValueOf(time.January),
		"July":                   reflect.ValueOf(time.July),
		"June":                   reflect.ValueOf(time.June),
		"Kitchen":                reflect.ValueOf(constant.MakeFromLiteral("\"3:04PM\"", token.STRING, 0)),
		"Layout":                 reflect.ValueOf(constant.MakeFromLiteral("\"01/02 03:04:05PM '06 -0700\"", token.STRING, 0)),
		"LoadLocation":           reflect.ValueOf(time.LoadLocation),
		"LoadLocationFromTZData": reflect.ValueOf(time.LoadLocationFromTZData),
		"Local":                  reflect.ValueOf(&time.Local).Elem(),
		"March":                  reflect.ValueOf(time.March),
		"May":                    reflect.ValueOf(time.May),
		"Microsecond":            reflect.ValueOf(time.Microsecond),
		"Millisecond":            reflect.ValueOf(time.Millisecond),
		"Minute":                 reflect.ValueOf(time.Minute),
		"Monday":                 reflect.ValueOf(time.Monday),
		"Nanosecond":             reflect.ValueOf(time.Nanosecond),
		"NewTicker":              reflect.ValueOf(time.NewTicker),
		"NewTimer":               reflect.ValueOf(time.NewTimer),
		"November":               reflect.ValueOf(time.November),
		"Now":                    reflect.ValueOf(time.Now),
		"October":                reflect.ValueOf(time.October),
		"Parse":                  reflect.ValueOf(time.Parse),
		"ParseDuration":          reflect.ValueOf(time.ParseDuration),
		"ParseInLocation":        reflect.ValueOf(time.ParseInLocation),
		"RFC1123":                reflect.ValueOf(constant.MakeFromLiteral("\"Mon, 02 Jan 2006 15:04:05 MST\"", token.STRING, 0)),
		"RFC1123Z":               reflect.ValueOf(constant.MakeFromLiteral("\"Mon, 02 Jan 2006 15:04:05 -0700\"", token.STRING, 0)),
		"RFC3339":                reflect.ValueOf(constant.MakeFromLiteral("\"2006-01-02T15:04:05Z07:00\"", token.STRING, 0)),
		"RFC3339Nano":            reflect.ValueOf(constant.MakeFromLiteral("\"2006-01-02T15:04:05.999999999Z07:00\"", token.STRING, 0)),
		"RFC822":                 reflect.ValueOf(constant.MakeFromLiteral("\"02 Jan 06 15:04 MST\"", token.STRING, 0)),
		"RFC822Z":                reflect.ValueOf(constant.MakeFromLiteral("\"02 Jan 06 15:04 -0700\"", token.STRING, 0)),
		"RFC850":                 reflect.ValueOf(constant.MakeFromLiteral("\"Monday, 02-Jan-06 15:04:05 MST\"", token.STRING, 0)),
		"RubyDate":               reflect.ValueOf(constant.MakeFromLiteral("\"Mon Jan 02 15:04:05 -0700 2006\"", token.STRING, 0)),
		"Saturday":               reflect.ValueOf(time.Saturday),
		"Second":                 reflect.ValueOf(time.Second),
		"September":              reflect.ValueOf(time.September),
		"Since":                  reflect.ValueOf(time.Since),
		"Sleep":                  reflect.ValueOf(time.Sleep),
		"Stamp":                  reflect.ValueOf(constant.MakeFromLiteral("\"Jan _2 15:04:05\"", token.STRING, 0)),
		"StampMicro":             reflect.ValueOf(constant.MakeFromLiteral("\"Jan _2 15:04:05.000000\"", token.STRING, 0)),
		"StampMilli":             reflect.ValueOf(constant.MakeFromLiteral("\"Jan _2 15:04:05.000\"", token.STRING, 0)),
		"StampNano":              reflect.ValueOf(constant.MakeFromLiteral("\"Jan _2 15:04:05.000000000\"", token.STRING, 0)),
		"Sunday":                 reflect.ValueOf(time.Sunday),
		"Thursday":               reflect.ValueOf(time.Thursday),
		"Tick":                   reflect.ValueOf(time.Tick),
		"TimeOnly":               reflect.ValueOf(constant.MakeFromLiteral("\"15:04:05\"", token.STRING, 0)),
		"Tuesday":                reflect.ValueOf(time.Tuesday),
		"UTC":                    reflect.ValueOf(&time.UTC).Elem(),
		"Unix":                   reflect.ValueOf(time.Unix),
		"UnixDate":               reflect.ValueOf(constant.MakeFromLiteral("\"Mon Jan _2 15:04:05 MST 2006\"", token.STRING, 0)),
		"UnixMicro":              reflect.ValueOf(time.UnixMicro),
		"UnixMilli":              reflect.ValueOf(time.UnixMilli),
		"Until":                  reflect.ValueOf(time.Until),
		"Wednesday":              reflect.ValueOf(time.Wednesday),

		// type definitions
		"Duration":   reflect.ValueOf((*time.Duration)(nil)),
		"Location":   reflect.ValueOf((*time.Location)(nil)),
		"Month":      reflect.ValueOf((*time.Month)(nil)),
		"ParseError": reflect.ValueOf((*time.ParseError)(nil)),
		"Ticker":     reflect.ValueOf((*time.Ticker)(nil)),
		"Time":       reflect.ValueOf((*time.Time)(nil)),
		"Timer":      reflect.ValueOf((*time.Timer)(nil)),
		"Weekday":    reflect.ValueOf((*time.Weekday)(nil)),
	}
}
