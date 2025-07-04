// Copyright 2015 PingCAP, Inc.
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

package parser_driver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"regexp"
	"strconv"
	"strings"
	gotime "time"
	"unicode"

	"github.com/pingcap/errors"
	"github.com/abbychau/mysql-parser/parser_driver/mysql"
)

// Time format without fractional seconds precision.
const (
	DateFormat = gotime.DateOnly
	TimeFormat = gotime.DateTime
	// TimeFSPFormat is time format with fractional seconds precision.
	TimeFSPFormat = "2006-01-02 15:04:05.000000"
	// UTCTimeFormat is used to parse and format gotime.
	UTCTimeFormat = "2006-01-02 15:04:05 UTC"
)

const (
	// MinYear is the minimum for mysql year type.
	MinYear int16 = 1901
	// MaxYear is the maximum for mysql year type.
	MaxYear int16 = 2155
	// MaxDuration is the maximum for duration.
	MaxDuration int64 = 838*10000 + 59*100 + 59
	// MinTime is the minimum for mysql time type.
	MinTime = -(838*gotime.Hour + 59*gotime.Minute + 59*gotime.Second)
	// MaxTime is the maximum for mysql time type.
	MaxTime = 838*gotime.Hour + 59*gotime.Minute + 59*gotime.Second
	// ZeroDatetimeStr is the string representation of a zero datetime.
	ZeroDatetimeStr = "0000-00-00 00:00:00"
	// ZeroDateStr is the string representation of a zero date.
	ZeroDateStr = "0000-00-00"

	// TimeMaxHour is the max hour for mysql time type.
	TimeMaxHour = 838
	// TimeMaxMinute is the max minute for mysql time type.
	TimeMaxMinute = 59
	// TimeMaxSecond is the max second for mysql time type.
	TimeMaxSecond = 59
	// TimeMaxValue is the maximum value for mysql time type.
	TimeMaxValue = TimeMaxHour*10000 + TimeMaxMinute*100 + TimeMaxSecond
	// TimeMaxValueSeconds is the maximum second value for mysql time type.
	TimeMaxValueSeconds = TimeMaxHour*3600 + TimeMaxMinute*60 + TimeMaxSecond
)

const (
	// YearIndex is index of 'YEARS-MONTHS DAYS HOURS:MINUTES:SECONDS.MICROSECONDS' expr Format
	YearIndex = 0 + iota
	// MonthIndex is index of 'YEARS-MONTHS DAYS HOURS:MINUTES:SECONDS.MICROSECONDS' expr Format
	MonthIndex
	// DayIndex is index of 'YEARS-MONTHS DAYS HOURS:MINUTES:SECONDS.MICROSECONDS' expr Format
	DayIndex
	// HourIndex is index of 'YEARS-MONTHS DAYS HOURS:MINUTES:SECONDS.MICROSECONDS' expr Format
	HourIndex
	// MinuteIndex is index of 'YEARS-MONTHS DAYS HOURS:MINUTES:SECONDS.MICROSECONDS' expr Format
	MinuteIndex
	// SecondIndex is index of 'YEARS-MONTHS DAYS HOURS:MINUTES:SECONDS.MICROSECONDS' expr Format
	SecondIndex
	// MicrosecondIndex is index of 'YEARS-MONTHS DAYS HOURS:MINUTES:SECONDS.MICROSECONDS' expr Format
	MicrosecondIndex
)

const (
	// YearMonthMaxCnt is max parameters count 'YEARS-MONTHS' expr Format allowed
	YearMonthMaxCnt = 2
	// DayHourMaxCnt is max parameters count 'DAYS HOURS' expr Format allowed
	DayHourMaxCnt = 2
	// DayMinuteMaxCnt is max parameters count 'DAYS HOURS:MINUTES' expr Format allowed
	DayMinuteMaxCnt = 3
	// DaySecondMaxCnt is max parameters count 'DAYS HOURS:MINUTES:SECONDS' expr Format allowed
	DaySecondMaxCnt = 4
	// DayMicrosecondMaxCnt is max parameters count 'DAYS HOURS:MINUTES:SECONDS.MICROSECONDS' expr Format allowed
	DayMicrosecondMaxCnt = 5
	// HourMinuteMaxCnt is max parameters count 'HOURS:MINUTES' expr Format allowed
	HourMinuteMaxCnt = 2
	// HourSecondMaxCnt is max parameters count 'HOURS:MINUTES:SECONDS' expr Format allowed
	HourSecondMaxCnt = 3
	// HourMicrosecondMaxCnt is max parameters count 'HOURS:MINUTES:SECONDS.MICROSECONDS' expr Format allowed
	HourMicrosecondMaxCnt = 4
	// MinuteSecondMaxCnt is max parameters count 'MINUTES:SECONDS' expr Format allowed
	MinuteSecondMaxCnt = 2
	// MinuteMicrosecondMaxCnt is max parameters count 'MINUTES:SECONDS.MICROSECONDS' expr Format allowed
	MinuteMicrosecondMaxCnt = 3
	// SecondMicrosecondMaxCnt is max parameters count 'SECONDS.MICROSECONDS' expr Format allowed
	SecondMicrosecondMaxCnt = 2
	// TimeValueCnt is parameters count 'YEARS-MONTHS DAYS HOURS:MINUTES:SECONDS.MICROSECONDS' expr Format
	TimeValueCnt = 7
)

// Zero values for different types.
var (
	// ZeroDuration is the zero value for Duration type.
	ZeroDuration = Duration{Duration: gotime.Duration(0), Fsp: DefaultFsp}

	// ZeroCoreTime is the zero value for Time type.
	ZeroTime = Time{}

	// ZeroDatetime is the zero value for datetime Time.
	ZeroDatetime = NewTime(ZeroCoreTime, mysql.TypeDatetime, DefaultFsp)

	// ZeroTimestamp is the zero value for timestamp Time.
	ZeroTimestamp = NewTime(ZeroCoreTime, mysql.TypeTimestamp, DefaultFsp)

	// ZeroDate is the zero value for date Time.
	ZeroDate = NewTime(ZeroCoreTime, mysql.TypeDate, DefaultFsp)
)

var (
	// MinDatetime is the minimum for Golang Time type.
	MinDatetime = FromDate(1, 1, 1, 0, 0, 0, 0)
	// MaxDatetime is the maximum for mysql datetime type.
	MaxDatetime = FromDate(9999, 12, 31, 23, 59, 59, 999999)

	// BoundTimezone is the timezone for min and max timestamp.
	BoundTimezone = gotime.UTC
	// MinTimestamp is the minimum for mysql timestamp type.
	MinTimestamp = NewTime(FromDate(1970, 1, 1, 0, 0, 1, 0), mysql.TypeTimestamp, DefaultFsp)
	// MaxTimestamp is the maximum for mysql timestamp type.
	MaxTimestamp = NewTime(FromDate(2038, 1, 19, 3, 14, 7, 999999), mysql.TypeTimestamp, DefaultFsp)

	// WeekdayNames lists names of weekdays, which are used in builtin time function `dayname`.
	WeekdayNames = []string{
		"Monday",
		"Tuesday",
		"Wednesday",
		"Thursday",
		"Friday",
		"Saturday",
		"Sunday",
	}

	// MonthNames lists names of months, which are used in builtin time function `monthname`.
	MonthNames = []string{
		"January", "February",
		"March", "April",
		"May", "June",
		"July", "August",
		"September", "October",
		"November", "December",
	}
)

const (
	// GoDurationDay is the gotime.Duration which equals to a Day.
	GoDurationDay = gotime.Hour * 24
	// GoDurationWeek is the gotime.Duration which equals to a Week.
	GoDurationWeek = GoDurationDay * 7
)

// FromGoTime translates time.Time to mysql time internal representation.
func FromGoTime(t gotime.Time) CoreTime {
	// Plus 500 nanosecond for rounding of the millisecond part.
	t = t.Add(500 * gotime.Nanosecond)

	year, month, day := t.Date()
	hour, minute, second := t.Clock()
	microsecond := t.Nanosecond() / 1000
	return FromDate(year, int(month), day, hour, minute, second, microsecond)
}

// FromDate makes a internal time representation from the given date.
func FromDate(year int, month int, day int, hour int, minute int, second int, microsecond int) CoreTime {
	v := uint64(ZeroCoreTime)
	v |= (uint64(microsecond) << microsecondBitFieldOffset) & microsecondBitFieldMask
	v |= (uint64(second) << secondBitFieldOffset) & secondBitFieldMask
	v |= (uint64(minute) << minuteBitFieldOffset) & minuteBitFieldMask
	v |= (uint64(hour) << hourBitFieldOffset) & hourBitFieldMask
	v |= (uint64(day) << dayBitFieldOffset) & dayBitFieldMask
	v |= (uint64(month) << monthBitFieldOffset) & monthBitFieldMask
	v |= (uint64(year) << yearBitFieldOffset) & yearBitFieldMask
	return CoreTime(v)
}

// FromDateChecked makes a internal time representation from the given date with field overflow check.
func FromDateChecked(year, month, day, hour, minute, second, microsecond int) (CoreTime, bool) {
	if uint64(year) >= (1<<yearBitFieldWidth) ||
		uint64(month) >= (1<<monthBitFieldWidth) ||
		uint64(day) >= (1<<dayBitFieldWidth) ||
		uint64(hour) >= (1<<hourBitFieldWidth) ||
		uint64(minute) >= (1<<minuteBitFieldWidth) ||
		uint64(second) >= (1<<secondBitFieldWidth) ||
		uint64(microsecond) >= (1<<microsecondBitFieldWidth) {
		return ZeroCoreTime, false
	}
	return FromDate(year, month, day, hour, minute, second, microsecond), true
}

// coreTime is an alias to CoreTime, embedd in Time.
type coreTime = CoreTime

// Time is the struct for handling datetime, timestamp and date.
type Time struct {
	coreTime
}

// Clock returns the hour, minute, and second within the day specified by t.
func (t Time) Clock() (hour int, minute int, second int) {
	return t.Hour(), t.Minute(), t.Second()
}

const (
	// Core time bit fields.
	yearBitFieldOffset, yearBitFieldWidth               uint64 = 50, 14
	monthBitFieldOffset, monthBitFieldWidth             uint64 = 46, 4
	dayBitFieldOffset, dayBitFieldWidth                 uint64 = 41, 5
	hourBitFieldOffset, hourBitFieldWidth               uint64 = 36, 5
	minuteBitFieldOffset, minuteBitFieldWidth           uint64 = 30, 6
	secondBitFieldOffset, secondBitFieldWidth           uint64 = 24, 6
	microsecondBitFieldOffset, microsecondBitFieldWidth uint64 = 4, 20

	// fspTt bit field.
	// `fspTt` format:
	// | fsp: 3 bits | type: 1 bit |
	// When `fsp` is valid (in range [0, 6]):
	// 1. `type` bit 0 represent `DateTime`
	// 2. `type` bit 1 represent `Timestamp`
	//
	// Since s`Date` does not require `fsp`, we could use `fspTt` == 0b1110 to represent it.
	fspTtBitFieldOffset, fspTtBitFieldWidth uint64 = 0, 4

	yearBitFieldMask        uint64 = ((1 << yearBitFieldWidth) - 1) << yearBitFieldOffset
	monthBitFieldMask       uint64 = ((1 << monthBitFieldWidth) - 1) << monthBitFieldOffset
	dayBitFieldMask         uint64 = ((1 << dayBitFieldWidth) - 1) << dayBitFieldOffset
	hourBitFieldMask        uint64 = ((1 << hourBitFieldWidth) - 1) << hourBitFieldOffset
	minuteBitFieldMask      uint64 = ((1 << minuteBitFieldWidth) - 1) << minuteBitFieldOffset
	secondBitFieldMask      uint64 = ((1 << secondBitFieldWidth) - 1) << secondBitFieldOffset
	microsecondBitFieldMask uint64 = ((1 << microsecondBitFieldWidth) - 1) << microsecondBitFieldOffset
	fspTtBitFieldMask       uint64 = ((1 << fspTtBitFieldWidth) - 1) << fspTtBitFieldOffset

	fspTtForDate         uint   = 0b1110
	fspBitFieldMask      uint64 = 0b1110
	coreTimeBitFieldMask        = ^fspTtBitFieldMask
)

// NewTime constructs time from core time, type and fsp.
func NewTime(coreTime CoreTime, tp uint8, fsp int) Time {
	t := ZeroTime
	p := (*uint64)(&t.coreTime)
	*p |= uint64(coreTime) & coreTimeBitFieldMask
	if tp == mysql.TypeDate {
		*p |= uint64(fspTtForDate)
		return t
	}
	if fsp == UnspecifiedFsp {
		fsp = DefaultFsp
	}
	*p |= uint64(fsp) << 1
	if tp == mysql.TypeTimestamp {
		*p |= 1
	}
	return t
}

func (t Time) getFspTt() uint {
	return uint(uint64(t.coreTime) & fspTtBitFieldMask)
}

func (t *Time) setFspTt(fspTt uint) {
	*(*uint64)(&t.coreTime) &= ^(fspTtBitFieldMask)
	*(*uint64)(&t.coreTime) |= uint64(fspTt)
}

// Type returns type value.
func (t Time) Type() uint8 {
	if t.getFspTt() == fspTtForDate {
		return mysql.TypeDate
	}
	if uint64(t.coreTime)&1 == 1 {
		return mysql.TypeTimestamp
	}
	return mysql.TypeDatetime
}

// Fsp returns fsp value.
func (t Time) Fsp() int {
	fspTt := t.getFspTt()
	if fspTt == fspTtForDate {
		return 0
	}
	return int(fspTt >> 1)
}

// SetType updates the type in Time.
// Only DateTime/Date/Timestamp is valid.
func (t *Time) SetType(tp uint8) {
	fspTt := t.getFspTt()
	if fspTt == fspTtForDate && tp != mysql.TypeDate {
		fspTt = 0
	}
	switch tp {
	case mysql.TypeDate:
		fspTt = fspTtForDate
	case mysql.TypeTimestamp:
		fspTt |= 1
	case mysql.TypeDatetime:
		fspTt &= ^(uint(1))
	}
	t.setFspTt(fspTt)
}

// SetFsp updates the fsp in Time.
func (t *Time) SetFsp(fsp int) {
	if t.getFspTt() == fspTtForDate {
		return
	}
	if fsp == UnspecifiedFsp {
		fsp = DefaultFsp
	}
	*(*uint64)(&t.coreTime) &= ^(fspBitFieldMask)
	*(*uint64)(&t.coreTime) |= uint64(fsp) << 1
}

// CoreTime returns core time.
func (t Time) CoreTime() CoreTime {
	return CoreTime(uint64(t.coreTime) & coreTimeBitFieldMask)
}

// SetCoreTime updates core time.
func (t *Time) SetCoreTime(ct CoreTime) {
	*(*uint64)(&t.coreTime) &= ^coreTimeBitFieldMask
	*(*uint64)(&t.coreTime) |= uint64(ct) & coreTimeBitFieldMask
}

// CurrentTime returns current time with type tp.
func CurrentTime(tp uint8) Time {
	return NewTime(FromGoTime(gotime.Now()), tp, 0)
}

// ConvertTimeZone converts the time value from one timezone to another.
// The input time should be a valid timestamp.
func (t *Time) ConvertTimeZone(from, to *gotime.Location) error {
	if !t.IsZero() {
		raw, err := t.GoTime(from)
		if err != nil {
			return errors.Trace(err)
		}
		converted := raw.In(to)
		t.SetCoreTime(FromGoTime(converted))
	}
	return nil
}

func (t Time) String() string {
	if t.Type() == mysql.TypeDate {
		// We control the format, so no error would occur.
		str, err := t.DateFormat("%Y-%m-%d")
		Log(errors.Trace(err))
		return str
	}

	str, err := t.DateFormat("%Y-%m-%d %H:%i:%s")
	Log(errors.Trace(err))
	fsp := t.Fsp()
	if fsp > 0 {
		tmp := fmt.Sprintf(".%06d", t.Microsecond())
		str = str + tmp[:1+fsp]
	}

	return str
}

// IsZero returns a boolean indicating whether the time is equal to ZeroCoreTime.
func (t Time) IsZero() bool {
	return compareTime(t.coreTime, ZeroCoreTime) == 0
}

// InvalidZero returns a boolean indicating whether the month or day is zero.
// Several functions are strict when passed a DATE() function value as their argument and reject incomplete dates with a day part of zero:
// CONVERT_TZ(), DATE_ADD(), DATE_SUB(), DAYOFYEAR(), TIMESTAMPDIFF(),
// TO_DAYS(), TO_SECONDS(), WEEK(), WEEKDAY(), WEEKOFYEAR(), YEARWEEK().
// Mysql Doc: https://dev.mysql.com/doc/refman/5.7/en/date-and-time-functions.html
func (t Time) InvalidZero() bool {
	return t.Month() == 0 || t.Day() == 0
}

const numberFormat = "%Y%m%d%H%i%s"
const dateFormat = "%Y%m%d"

// ToNumber returns a formatted number.
// e.g,
// 2012-12-12 -> 20121212
// 2012-12-12T10:10:10 -> 20121212101010
// 2012-12-12T10:10:10.123456 -> 20121212101010.123456
func (t Time) ToNumber() *MyDecimal {
	dec := new(MyDecimal)
	t.FillNumber(dec)
	return dec
}

// FillNumber is the same as ToNumber,
// but reuses input decimal instead of allocating one.
func (t Time) FillNumber(dec *MyDecimal) {
	if t.IsZero() {
		dec.FromInt(0)
		return
	}

	// Fix issue #1046
	// Prevents from converting 2012-12-12 to 20121212000000
	var tfStr string
	if t.Type() == mysql.TypeDate {
		tfStr = dateFormat
	} else {
		tfStr = numberFormat
	}

	s, err := t.DateFormat(tfStr)
	if err != nil {
		BgLogger.Error("never happen because we've control the format!", "fatal")
	}

	fsp := t.Fsp()
	if fsp > 0 {
		s1 := fmt.Sprintf("%s.%06d", s, t.Microsecond())
		s = s1[:len(s)+fsp+1]
	}
	// We skip checking error here because time formatted string can be parsed certainly.
	err = dec.FromString([]byte(s))
	Log(errors.Trace(err))
}

// Convert converts t with type tp.
func (t Time) Convert(ctx Context, tp uint8) (Time, error) {
	t1 := t
	if t.Type() == tp || t.IsZero() {
		t1.SetType(tp)
		return t1, nil
	}

	t1.SetType(tp)
	err := t1.Check(ctx)
	if tp == mysql.TypeTimestamp && ErrTimestampInDSTTransition.Equal(err) {
		tAdj, adjErr := t1.AdjustedGoTime(ctx.Location())
		if adjErr == nil {
			ctx.AppendWarning(err)
			return Time{FromGoTime(tAdj)}, nil
		}
	}
	return t1, errors.Trace(err)
}

// ConvertToDuration converts mysql datetime, timestamp and date to mysql time type.
// e.g,
// 2012-12-12T10:10:10 -> 10:10:10
// 2012-12-12 -> 0
func (t Time) ConvertToDuration() (Duration, error) {
	if t.IsZero() {
		return ZeroDuration, nil
	}

	hour, minute, second := t.Clock()
	frac := t.Microsecond() * 1000

	d := gotime.Duration(hour*3600+minute*60+second)*gotime.Second + gotime.Duration(frac) //nolint:durationcheck
	// TODO: check convert validation
	return Duration{Duration: d, Fsp: t.Fsp()}, nil
}

// Compare returns an integer comparing the time instant t to o.
// If t is after o, returns 1, equal o, returns 0, before o, returns -1.
func (t Time) Compare(o Time) int {
	return compareTime(t.coreTime, o.coreTime)
}

// CompareString is like Compare,
// but parses string to Time then compares.
func (t Time) CompareString(ctx Context, str string) (int, error) {
	// use MaxFsp to parse the string
	o, err := ParseTime(ctx, str, t.Type(), MaxFsp)
	if err != nil {
		return 0, errors.Trace(err)
	}

	return t.Compare(o), nil
}

// roundTime rounds the time value according to digits count specified by fsp.
func roundTime(t gotime.Time, fsp int) gotime.Time {
	d := gotime.Duration(math.Pow10(9 - fsp))
	return t.Round(d)
}

// RoundFrac rounds the fraction part of a time-type value according to `fsp`.
func (t Time) RoundFrac(ctx Context, fsp int) (Time, error) {
	if t.Type() == mysql.TypeDate || t.IsZero() {
		// date type has no fsp
		return t, nil
	}

	fsp, err := CheckFsp(fsp)
	if err != nil {
		return t, errors.Trace(err)
	}

	if fsp == t.Fsp() {
		// have same fsp
		return t, nil
	}

	var nt CoreTime
	if t1, err := t.GoTime(ctx.Location()); err == nil {
		t1 = roundTime(t1, fsp)
		nt = FromGoTime(t1)
	} else {
		// Take the hh:mm:ss part out to avoid handle month or day = 0.
		hour, minute, second, microsecond := t.Hour(), t.Minute(), t.Second(), t.Microsecond()
		t1 := gotime.Date(1, 1, 1, hour, minute, second, microsecond*1000, ctx.Location())
		t2 := roundTime(t1, fsp)
		hour, minute, second = t2.Clock()
		microsecond = t2.Nanosecond() / 1000

		// TODO: when hh:mm:ss overflow one day after rounding, it should be add to yy:mm:dd part,
		// but mm:dd may contain 0, it makes the code complex, so we ignore it here.
		if t2.Day()-1 > 0 {
			return t, errors.Trace(ErrWrongValue.GenWithStackByArgs(TimeStr, t.String()))
		}
		var ok bool
		nt, ok = FromDateChecked(t.Year(), t.Month(), t.Day(), hour, minute, second, microsecond)
		if !ok {
			return t, errors.Trace(ErrWrongValue.GenWithStackByArgs(TimeStr, t.String()))
		}
	}

	return NewTime(nt, t.Type(), fsp), nil
}

// MarshalJSON implements Marshaler.MarshalJSON interface.
func (t Time) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.coreTime)
}

// UnmarshalJSON implements Unmarshaler.UnmarshalJSON interface.
func (t *Time) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &t.coreTime)
}

// GetFsp gets the fsp of a string.
func GetFsp(s string) int {
	index := GetFracIndex(s)
	var fsp int
	if index < 0 {
		fsp = 0
	} else {
		fsp = len(s) - index - 1
	}
	if fsp > 6 {
		fsp = 6
	}
	return fsp
}

// GetFracIndex finds the last '.' for get fracStr, index = -1 means fracStr not found.
// but for format like '2019.01.01 00:00:00', the index should be -1.
//
// It will not be affected by the time zone suffix.
// For format like '2020-01-01 12:00:00.123456+05:00' and `2020-01-01 12:00:00.123456-05:00`, the index should be 19.
// related issue https://github.com/pingcap/tidb/issues/35291 and https://github.com/pingcap/tidb/issues/49555
func GetFracIndex(s string) (index int) {
	tzIndex, _, _, _, _ := GetTimezone(s)
	var end int
	if tzIndex != -1 {
		end = tzIndex - 1
	} else {
		end = len(s) - 1
	}
	index = -1
	for i := end; i >= 0; i-- {
		if s[i] != '+' && s[i] != '-' && isPunctuation(s[i]) {
			if s[i] == '.' {
				index = i
			}
			break
		}
	}

	return index
}

// RoundFrac rounds fractional seconds precision with new fsp and returns a new one.
// We will use the “round half up” rule, e.g, >= 0.5 -> 1, < 0.5 -> 0,
// so 2011:11:11 10:10:10.888888 round 0 -> 2011:11:11 10:10:11
// and 2011:11:11 10:10:10.111111 round 0 -> 2011:11:11 10:10:10
func RoundFrac(t gotime.Time, fsp int) (gotime.Time, error) {
	_, err := CheckFsp(fsp)
	if err != nil {
		return t, errors.Trace(err)
	}
	return t.Round(gotime.Duration(math.Pow10(9-fsp)) * gotime.Nanosecond), nil //nolint:durationcheck
}

// TruncateFrac truncates fractional seconds precision with new fsp and returns a new one.
// 2011:11:11 10:10:10.888888 round 0 -> 2011:11:11 10:10:10
// 2011:11:11 10:10:10.111111 round 0 -> 2011:11:11 10:10:10
func TruncateFrac(t gotime.Time, fsp int) (gotime.Time, error) {
	if _, err := CheckFsp(fsp); err != nil {
		return t, err
	}
	return t.Truncate(gotime.Duration(math.Pow10(9-fsp)) * gotime.Nanosecond), nil //nolint:durationcheck
}

// ToPackedUint encodes Time to a packed uint64 value.
//
//	 1 bit  0
//	17 bits year*13+month   (year 0-9999, month 0-12)
//	 5 bits day             (0-31)
//	 5 bits hour            (0-23)
//	 6 bits minute          (0-59)
//	 6 bits second          (0-59)
//	24 bits microseconds    (0-999999)
//
//	Total: 64 bits = 8 bytes
//
//	0YYYYYYY.YYYYYYYY.YYdddddh.hhhhmmmm.mmssssss.ffffffff.ffffffff.ffffffff
func (t Time) ToPackedUint() (uint64, error) {
	tm := t
	if t.IsZero() {
		return 0, nil
	}
	year, month, day := tm.Year(), tm.Month(), tm.Day()
	hour, minute, sec := tm.Hour(), tm.Minute(), tm.Second()
	ymd := uint64(((year*13 + month) << 5) | day)
	hms := uint64(hour<<12 | minute<<6 | sec)
	micro := uint64(tm.Microsecond())
	return ((ymd<<17 | hms) << 24) | micro, nil
}

// FromPackedUint decodes Time from a packed uint64 value.
func (t *Time) FromPackedUint(packed uint64) error {
	if packed == 0 {
		t.SetCoreTime(ZeroCoreTime)
		return nil
	}
	ymdhms := packed >> 24
	ymd := ymdhms >> 17
	day := int(ymd & (1<<5 - 1))
	ym := ymd >> 5
	month := int(ym % 13)
	year := int(ym / 13)

	hms := ymdhms & (1<<17 - 1)
	second := int(hms & (1<<6 - 1))
	minute := int((hms >> 6) & (1<<6 - 1))
	hour := int(hms >> 12)
	microsec := int(packed % (1 << 24))

	t.SetCoreTime(FromDate(year, month, day, hour, minute, second, microsec))

	return nil
}

// Check function checks whether t matches valid Time format.
// If allowZeroInDate is false, it returns ErrZeroDate when month or day is zero.
// FIXME: See https://dev.mysql.com/doc/refman/5.7/en/sql-mode.html#sqlmode_no_zero_in_date
func (t Time) Check(ctx Context) error {
	allowZeroInDate := ctx.Flags().IgnoreZeroInDate()
	allowInvalidDate := ctx.Flags().IgnoreInvalidDateErr()
	var err error
	switch t.Type() {
	case mysql.TypeTimestamp:
		err = checkTimestampType(t.coreTime, ctx.Location())
	case mysql.TypeDatetime, mysql.TypeDate:
		err = checkDatetimeType(t.coreTime, allowZeroInDate, allowInvalidDate)
	}
	return errors.Trace(err)
}

// Sub subtracts t1 from t, returns a duration value.
// Note that sub should not be done on different time types.
func (t *Time) Sub(ctx Context, t1 *Time) Duration {
	var duration gotime.Duration
	if t.Type() == mysql.TypeTimestamp && t1.Type() == mysql.TypeTimestamp {
		a, err := t.GoTime(ctx.Location())
		Log(errors.Trace(err))
		b, err := t1.GoTime(ctx.Location())
		Log(errors.Trace(err))
		duration = a.Sub(b)
	} else {
		seconds, microseconds, neg := calcTimeTimeDiff(t.coreTime, t1.coreTime, 1)
		duration = gotime.Duration(seconds*1e9 + microseconds*1e3)
		if neg {
			duration = -duration
		}
	}

	fsp := t.Fsp()
	fsp1 := t1.Fsp()
	if fsp < fsp1 {
		fsp = fsp1
	}
	return Duration{
		Duration: duration,
		Fsp:      fsp,
	}
}

// Add adds d to t, returns the result time value.
func (t *Time) Add(ctx Context, d Duration) (Time, error) {
	seconds, microseconds, _ := calcTimeDurationDiff(t.coreTime, d)
	days := seconds / secondsIn24Hour
	year, month, day := getDateFromDaynr(uint(days))
	var tm Time
	tm.setYear(uint16(year))
	tm.setMonth(uint8(month))
	tm.setDay(uint8(day))
	calcTimeFromSec(&tm.coreTime, seconds%secondsIn24Hour, microseconds)
	if t.Type() == mysql.TypeDate {
		tm.setHour(0)
		tm.setMinute(0)
		tm.setSecond(0)
		tm.setMicrosecond(0)
	}
	fsp := max(d.Fsp, t.Fsp())
	ret := NewTime(tm.coreTime, t.Type(), fsp)
	return ret, ret.Check(ctx)
}

// TimestampDiff returns t2 - t1 where t1 and t2 are date or datetime expressions.
// The unit for the result (an integer) is given by the unit argument.
// The legal values for unit are "YEAR" "QUARTER" "MONTH" "DAY" "HOUR" "SECOND" and so on.
func TimestampDiff(unit string, t1 Time, t2 Time) int64 {
	return timestampDiff(unit, t1.coreTime, t2.coreTime)
}

// ParseDateFormat parses a formatted date string and returns separated components.
func ParseDateFormat(format string) []string {
	format = strings.TrimSpace(format)

	if len(format) == 0 {
		return nil
	}

	// Date format must start with number.
	if !isDigit(format[0]) {
		return nil
	}

	start := 0
	// Initialize `seps` with capacity of 6. The input `format` is typically
	// a date time of the form time.DateTime, which has 6 numeric parts
	// (the fractional second part is usually removed by `splitDateTime`).
	// Setting `seps`'s capacity to 6 avoids reallocation in this common case.
	seps := make([]string, 0, 6)

	for i := 1; i < len(format)-1; i++ {
		if isValidSeparator(format[i], len(seps)) {
			prevParts := len(seps)
			seps = append(seps, format[start:i])
			start = i + 1

			// consume further consecutive separators
			for j := i + 1; j < len(format); j++ {
				if !isValidSeparator(format[j], prevParts) {
					break
				}

				start++
				i++
			}

			continue
		}

		if !isDigit(format[i]) {
			return nil
		}
	}

	seps = append(seps, format[start:])
	return seps
}

// helper for date part splitting, punctuation characters are valid separators anywhere,
// while space and 'T' are valid separators only between date and time.
func isValidSeparator(c byte, prevParts int) bool {
	if isPunctuation(c) {
		return true
	}

	// for https://github.com/pingcap/tidb/issues/32232
	if prevParts == 2 && (c == 'T' || c == ' ' || c == '\t' || c == '\n' || c == '\v' || c == '\f' || c == '\r') {
		return true
	}

	if prevParts > 4 && !isDigit(c) {
		return true
	}
	return false
}

var validIdxCombinations = map[int]struct {
	h int
	m int
}{
	100: {0, 0}, // 23:59:59Z
	30:  {2, 0}, // 23:59:59+08
	50:  {4, 2}, // 23:59:59+0800
	63:  {5, 2}, // 23:59:59+08:00
	// postgres supports the following additional syntax that deviates from ISO8601, although we won't support it
	// currently, it will be fairly easy to add in the current parsing framework
	// 23:59:59Z+08
	// 23:59:59Z+08:00
}

// GetTimezone parses the trailing timezone information of a given time string literal. If idx = -1 is returned, it
// means timezone information not found, otherwise it indicates the index of the starting index of the timezone
// information. If the timezone contains sign, hour part and/or minute part, it will be returned as is, otherwise an
// empty string will be returned.
//
// Supported syntax:
//
//	MySQL compatible: ((?P<tz_sign>[-+])(?P<tz_hour>[0-9]{2}):(?P<tz_minute>[0-9]{2})){0,1}$, see
//	  https://dev.mysql.com/doc/refman/8.0/en/time-zone-support.html and https://dev.mysql.com/doc/refman/8.0/en/datetime.html
//	  the first link specified that timezone information should be in "[H]H:MM, prefixed with a + or -" while the
//	  second link specified that for string literal, "hour values less than than 10, a leading zero is required.".
//	ISO-8601: Z|((((?P<tz_sign>[-+])(?P<tz_hour>[0-9]{2})(:(?P<tz_minute>[0-9]{2}){0,1}){0,1})|((?P<tz_minute>[0-9]{2}){0,1}){0,1}))$
//	  see https://www.cl.cam.ac.uk/~mgk25/iso-time.html
func GetTimezone(lit string) (idx int, tzSign, tzHour, tzSep, tzMinute string) {
	idx, zidx, sidx, spidx := -1, -1, -1, -1
	// idx is for the position of the starting of the timezone information
	// zidx is for the z symbol
	// sidx is for the sign
	// spidx is for the separator
	l := len(lit)
	// the following loop finds the first index of Z, sign, and separator from backwards.
	for i := l - 1; 0 <= i; i-- {
		if lit[i] == 'Z' {
			zidx = i
			break
		}
		if sidx == -1 && (lit[i] == '-' || lit[i] == '+') {
			sidx = i
		}
		if spidx == -1 && lit[i] == ':' {
			spidx = i
		}
	}
	// we could enumerate all valid combinations of these values and look it up in a table, see validIdxCombinations
	// zidx can be -1 (23:59:59+08:00), l-1 (23:59:59Z)
	// sidx can be -1, l-3, l-5, l-6
	// spidx can be -1, l-3
	k := 0
	if l-zidx == 1 {
		k += 100
	}
	if t := l - sidx; t == 3 || t == 5 || t == 6 {
		k += t * 10
	}
	if l-spidx == 3 {
		k += 3
	}
	if v, ok := validIdxCombinations[k]; ok {
		hidx, midx := l-v.h, l-v.m
		valid := func(v string) bool {
			return '0' <= v[0] && v[0] <= '9' && '0' <= v[1] && v[1] <= '9'
		}
		if sidx != -1 {
			tzSign = lit[sidx : sidx+1]
			idx = sidx
		}
		if zidx != -1 {
			idx = zidx
		}
		if (l - spidx) == 3 {
			tzSep = lit[spidx : spidx+1]
		}
		if v.h != 0 {
			tzHour = lit[hidx : hidx+2]
			if !valid(tzHour) {
				return -1, "", "", "", ""
			}
		}
		if v.m != 0 {
			tzMinute = lit[midx : midx+2]
			if !valid(tzMinute) {
				return -1, "", "", "", ""
			}
		}
		return
	}
	return -1, "", "", "", ""
}

// See https://dev.mysql.com/doc/refman/5.7/en/date-and-time-literals.html.
// splitDateTime splits the string literal into 3 parts, date & time, FSP(Fractional Seconds Precision) and time zone.
// For FSP, The only delimiter recognized between a date & time part and a fractional seconds part is the decimal point,
// therefore we could look from backwards at the literal to find the index of the decimal point.
// For time zone, the possible delimiter could be +/- (w.r.t. MySQL 8.0, see
// https://dev.mysql.com/doc/refman/8.0/en/datetime.html) and Z/z (w.r.t. ISO 8601, see section Time zone in
// https://www.cl.cam.ac.uk/~mgk25/iso-time.html). We also look from backwards for the delimiter, see GetTimezone.
func splitDateTime(format string) (seps []string, fracStr string, hasTZ bool, tzSign, tzHour, tzSep, tzMinute string, truncated bool) {
	tzIndex, tzSign, tzHour, tzSep, tzMinute := GetTimezone(format)
	if tzIndex > 0 {
		hasTZ = true
		for ; tzIndex > 0 && isPunctuation(format[tzIndex-1]); tzIndex-- {
			// In case of multiple separators, e.g. 2020-10--10
			continue
		}
		format = format[:tzIndex]
	}
	fracIndex := GetFracIndex(format)
	if fracIndex > 0 {
		// Only contain digits
		fracEnd := fracIndex + 1
		for fracEnd < len(format) && isDigit(format[fracEnd]) {
			fracEnd++
		}
		truncated = (fracEnd != len(format))
		fracStr = format[fracIndex+1 : fracEnd]
		for ; fracIndex > 0 && isPunctuation(format[fracIndex-1]); fracIndex-- {
			// In case of multiple separators, e.g. 2020-10..10
			continue
		}
		format = format[:fracIndex]
	}
	seps = ParseDateFormat(format)
	return
}

// See https://dev.mysql.com/doc/refman/5.7/en/date-and-time-literals.html.
func parseDatetime(ctx Context, str string, fsp int, isFloat bool) (Time, error) {
	var (
		year, month, day, hour, minute, second, deltaHour, deltaMinute int
		fracStr                                                        string
		tzSign, tzHour, tzSep, tzMinute                                string
		hasTZ, hhmmss                                                  bool
		err                                                            error
	)

	seps, fracStr, hasTZ, tzSign, tzHour, tzSep, tzMinute, truncatedOrIncorrect := splitDateTime(str)
	if truncatedOrIncorrect {
		ctx.AppendWarning(ErrTruncatedWrongVal.FastGenByArgs("datetime", str))
	}
	/*
		if we have timezone parsed, there are the following cases to be considered, however some of them are wrongly parsed, and we should consider absorb them back to seps.

		1. Z, then it must be time zone information, and we should not tamper with it
		2. -HH, it might be from
		    1. no fracStr
		        1. YYYY-MM-DD
		        2. YYYY-MM-DD-HH
		        3. YYYY-MM-DD HH-MM
		        4. YYYY-MM-DD HH:MM-SS
		        5. YYYY-MM-DD HH:MM:SS-HH (correct, no need absorb)
		    2. with fracStr
		        1. YYYY.MM-DD
		        2. YYYY-MM.DD-HH
		        3. YYYY-MM-DD.HH-MM
		        4. YYYY-MM-DD HH.MM-SS
		        5. YYYY-MM-DD HH:MM.SS-HH (correct, no need absorb)
		3. -HH:MM, similarly it might be from
		    1. no fracStr
		        1. YYYY-MM:DD
		        2. YYYY-MM-DD:HH
		        3. YYYY-MM-DD-HH:MM
		        4. YYYY-MM-DD HH-MM:SS
		        5. YYYY-MM-DD HH:MM-SS:HH (invalid)
		        6. YYYY-MM-DD HH:MM:SS-HH:MM (correct, no need absorb)
		    2. with fracStr
		        1. YYYY.MM-DD:HH
		        2. YYYY-MM.DD-HH:MM
		        3. YYYY-MM-DD.HH-MM:SS
		        4. YYYY-MM-DD HH.MM-SS:HH (invalid)
		        5. YYYY-MM-DD HH:MM.SS-HH:MM (correct, no need absorb)
		4. -HHMM, there should only be one case, that is both the date and time part have existed, only then could we have fracStr or time zone
		    1. YYYY-MM-DD HH:MM:SS.FSP-HHMM (correct, no need absorb)

		to summarize, FSP and timezone is only valid if we have date and time presented, otherwise we should consider absorbing
		FSP or timezone into seps. additionally, if we want to absorb timezone, we either absorb them all, or not, meaning
		we won't only absorb tzHour but not tzMinute.

		additional case to consider is that when the time literal is presented in float string (e.g. `YYYYMMDD.HHMMSS`), in
		this case, FSP should not be absorbed and only `+HH:MM` would be allowed (i.e. Z, +HHMM, +HH that comes from ISO8601
		should be banned), because it only conforms to MySQL's timezone parsing logic, but it is not valid in ISO8601.
		However, I think it is generally acceptable to allow a wider spectrum of timezone format in string literal.
	*/

	// noAbsorb tests if can absorb FSP or TZ
	noAbsorb := func(seps []string) bool {
		// if we have more than 5 parts (i.e. 6), the tailing part can't be absorbed
		// or if we only have 1 part, but its length is longer than 4, then it is at least YYMMD, in this case, FSP can
		// not be absorbed, and it will be handled later, and the leading sign prevents TZ from being absorbed, because
		// if date part has no separators, we can't use -/+ as separators between date & time.
		return len(seps) > 5 || (len(seps) == 1 && len(seps[0]) > 4)
	}
	if len(fracStr) != 0 && !isFloat {
		if !noAbsorb(seps) {
			seps = append(seps, fracStr)
			fracStr = ""
		}
	}
	if hasTZ && tzSign != "" {
		// if tzSign is empty, we can be sure that the string literal contains timezone (such as 2010-10-10T10:10:10Z),
		// therefore we could safely skip this branch.
		if !noAbsorb(seps) && !(tzMinute != "" && tzSep == "") {
			// we can't absorb timezone if there is no separate between tzHour and tzMinute
			if len(tzHour) != 0 {
				seps = append(seps, tzHour)
			}
			if len(tzMinute) != 0 {
				seps = append(seps, tzMinute)
			}
			hasTZ = false
		}
	}
	switch len(seps) {
	case 0:
		return ZeroDatetime, errors.Trace(ErrWrongValue.GenWithStackByArgs(DateTimeStr, str))
	case 1:
		l := len(seps[0])
		// Values specified as numbers
		if isFloat {
			numOfTime, err := StrToInt(ctx, seps[0], false)
			if err != nil {
				return ZeroDatetime, errors.Trace(ErrWrongValue.GenWithStackByArgs(DateTimeStr, str))
			}

			dateTime, err := ParseDatetimeFromNum(ctx, numOfTime)
			if err != nil {
				return ZeroDatetime, errors.Trace(ErrWrongValue.GenWithStackByArgs(DateTimeStr, str))
			}

			year, month, day, hour, minute, second =
				dateTime.Year(), dateTime.Month(), dateTime.Day(), dateTime.Hour(), dateTime.Minute(), dateTime.Second()

			// case: 0.XXX or like "20170118.999"
			if seps[0] == "0" || (l >= 9 && l <= 14) {
				hhmmss = true
			}

			break
		}

		// Values specified as strings
		switch l {
		case 14: // No delimiter.
			// YYYYMMDDHHMMSS
			_, err = fmt.Sscanf(seps[0], "%4d%2d%2d%2d%2d%2d", &year, &month, &day, &hour, &minute, &second)
			hhmmss = true
		case 12: // YYMMDDHHMMSS
			_, err = fmt.Sscanf(seps[0], "%2d%2d%2d%2d%2d%2d", &year, &month, &day, &hour, &minute, &second)
			year = adjustYear(year)
			hhmmss = true
		case 11: // YYMMDDHHMMS
			_, err = fmt.Sscanf(seps[0], "%2d%2d%2d%2d%2d%1d", &year, &month, &day, &hour, &minute, &second)
			year = adjustYear(year)
			hhmmss = true
		case 10: // YYMMDDHHMM
			_, err = fmt.Sscanf(seps[0], "%2d%2d%2d%2d%2d", &year, &month, &day, &hour, &minute)
			year = adjustYear(year)
		case 9: // YYMMDDHHM
			_, err = fmt.Sscanf(seps[0], "%2d%2d%2d%2d%1d", &year, &month, &day, &hour, &minute)
			year = adjustYear(year)
		case 8: // YYYYMMDD
			_, err = fmt.Sscanf(seps[0], "%4d%2d%2d", &year, &month, &day)
		case 7: // YYMMDDH
			_, err = fmt.Sscanf(seps[0], "%2d%2d%2d%1d", &year, &month, &day, &hour)
			year = adjustYear(year)
		case 6, 5:
			// YYMMDD && YYMMD
			_, err = fmt.Sscanf(seps[0], "%2d%2d%2d", &year, &month, &day)
			year = adjustYear(year)
		default:
			return ZeroDatetime, errors.Trace(ErrWrongValue.GenWithStackByArgs(TimeStr, str))
		}
		if l == 5 || l == 6 || l == 8 {
			// YYMMDD or YYYYMMDD
			// We must handle float => string => datetime, the difference is that fractional
			// part of float type is discarded directly, while fractional part of string type
			// is parsed to HH:MM:SS.
			if !isFloat {
				// '20170118.123423' => 2017-01-18 12:34:23.234
				switch len(fracStr) {
				case 0:
				case 1, 2:
					_, err = fmt.Sscanf(fracStr, "%2d ", &hour)
				case 3, 4:
					_, err = fmt.Sscanf(fracStr, "%2d%2d ", &hour, &minute)
				default:
					_, err = fmt.Sscanf(fracStr, "%2d%2d%2d ", &hour, &minute, &second)
				}
				truncatedOrIncorrect = err != nil
			}
			// 20170118.123423 => 2017-01-18 00:00:00
		}
		if l == 9 || l == 10 {
			if len(fracStr) == 0 {
				second = 0
			} else {
				_, err = fmt.Sscanf(fracStr, "%2d ", &second)
			}
			truncatedOrIncorrect = err != nil
		}
		if truncatedOrIncorrect {
			ctx.AppendWarning(ErrTruncatedWrongVal.FastGenByArgs("datetime", str))
			err = nil
		}
	case 2:
		return ZeroDatetime, errors.Trace(ErrWrongValue.FastGenByArgs(DateTimeStr, str))
	case 3:
		// YYYY-MM-DD
		err = scanTimeArgs(seps, &year, &month, &day)
	case 4:
		// YYYY-MM-DD HH
		err = scanTimeArgs(seps, &year, &month, &day, &hour)
	case 5:
		// YYYY-MM-DD HH-MM
		err = scanTimeArgs(seps, &year, &month, &day, &hour, &minute)
	case 6:
		// We don't have fractional seconds part.
		// YYYY-MM-DD HH-MM-SS
		err = scanTimeArgs(seps, &year, &month, &day, &hour, &minute, &second)
		hhmmss = true
	default:
		// For case like `2020-05-28 23:59:59 00:00:00`, the seps should be > 6, the reluctant parts should be truncated.
		seps = seps[:6]
		// YYYY-MM-DD HH-MM-SS
		ctx.AppendWarning(ErrTruncatedWrongVal.FastGenByArgs("datetime", str))
		err = scanTimeArgs(seps, &year, &month, &day, &hour, &minute, &second)
		hhmmss = true
	}
	if err != nil {
		if err == io.EOF {
			return ZeroDatetime, errors.Trace(ErrWrongValue.GenWithStackByArgs(DateTimeStr, str))
		}
		return ZeroDatetime, errors.Trace(err)
	}

	// If str is sepereated by delimiters, the first one is year, and if the year is 1/2 digit,
	// we should adjust it.
	// TODO: adjust year is very complex, now we only consider the simplest way.
	if len(seps[0]) <= 2 && !isFloat {
		if !(year == 0 && month == 0 && day == 0 && hour == 0 && minute == 0 && second == 0 && fracStr == "") {
			year = adjustYear(year)
		}
		// Skip a special case "00-00-00".
	}

	var microsecond int
	var overflow bool
	if hhmmss {
		// If input string is "20170118.999", without hhmmss, fsp is meaningless.
		// TODO: this case is not only meaningless, but erroneous, please confirm.
		microsecond, overflow, err = ParseFrac(fracStr, fsp)
		if err != nil {
			return ZeroDatetime, errors.Trace(err)
		}
	}

	tmp, ok := FromDateChecked(year, month, day, hour, minute, second, microsecond)
	if !ok {
		return ZeroDatetime, errors.Trace(ErrWrongValue.GenWithStackByArgs(DateTimeStr, str))
	}
	if overflow {
		// Convert to Go time and add 1 second, to handle input like 2017-01-05 08:40:59.575601
		var t1 gotime.Time
		if t1, err = tmp.GoTime(ctx.Location()); err != nil {
			return ZeroDatetime, errors.Trace(err)
		}
		tmp = FromGoTime(t1.Add(gotime.Second))
	}
	if hasTZ {
		// without hhmmss, timezone is also meaningless
		if !hhmmss {
			return ZeroDatetime, errors.Trace(ErrWrongValue.GenWithStack(DateTimeStr, str))
		}
		if len(tzHour) != 0 {
			deltaHour = int((tzHour[0]-'0')*10 + (tzHour[1] - '0'))
		}
		if len(tzMinute) != 0 {
			deltaMinute = int((tzMinute[0]-'0')*10 + (tzMinute[1] - '0'))
		}
		// allowed delta range is [-14:00, 14:00], and we will intentionally reject -00:00
		if deltaHour > 14 || deltaMinute > 59 || (deltaHour == 14 && deltaMinute != 0) || (tzSign == "-" && deltaHour == 0 && deltaMinute == 0) {
			return ZeroDatetime, errors.Trace(ErrWrongValue.GenWithStackByArgs(DateTimeStr, str))
		}
		// by default, if the temporal string literal does not contain timezone information, it will be in the timezone
		// specified by the time_zone system variable. However, if the timezone is specified in the string literal, we
		// will use the specified timezone to interpret the string literal and convert it into the system timezone.
		offset := deltaHour*60*60 + deltaMinute*60
		if tzSign == "-" {
			offset = -offset
		}
		loc := gotime.FixedZone(fmt.Sprintf("UTC%s%s:%s", tzSign, tzHour, tzMinute), offset)
		t1, err := tmp.GoTime(loc)
		if err != nil {
			return ZeroDatetime, errors.Trace(err)
		}
		t1 = t1.In(ctx.Location())
		tmp = FromGoTime(t1)
	}

	nt := NewTime(tmp, mysql.TypeDatetime, fsp)

	return nt, nil
}

func scanTimeArgs(seps []string, args ...*int) error {
	if len(seps) != len(args) {
		return errors.Trace(ErrWrongValue.GenWithStackByArgs(TimeStr, seps))
	}

	var err error
	for i, s := range seps {
		*args[i], err = strconv.Atoi(s)
		if err != nil {
			return errors.Trace(err)
		}
	}
	return nil
}

// ParseYear parses a formatted string and returns a year number.
func ParseYear(str string) (int16, error) {
	v, err := strconv.ParseInt(str, 10, 16)
	if err != nil {
		return 0, errors.Trace(err)
	}
	y := int16(v)

	if len(str) == 2 || len(str) == 1 {
		y = int16(adjustYear(int(y)))
	} else if len(str) != 4 {
		return 0, errors.Trace(ErrInvalidYearFormat)
	}

	if y < MinYear || y > MaxYear {
		return 0, errors.Trace(ErrInvalidYearFormat)
	}

	return y, nil
}

// adjustYear adjusts year according to y.
// See https://dev.mysql.com/doc/refman/5.7/en/two-digit-years.html
func adjustYear(y int) int {
	if y >= 0 && y <= 69 {
		y = 2000 + y
	} else if y >= 70 && y <= 99 {
		y = 1900 + y
	}
	return y
}

// AdjustYear is used for adjusting year and checking its validation.
func AdjustYear(y int64, adjustZero bool) (int64, error) {
	if y == 0 && !adjustZero {
		return y, nil
	}
	y = int64(adjustYear(int(y)))
	if y < 0 {
		return 0, errors.Trace(ErrWarnDataOutOfRange)
	}
	if y < int64(MinYear) {
		return int64(MinYear), errors.Trace(ErrWarnDataOutOfRange)
	}
	if y > int64(MaxYear) {
		return int64(MaxYear), errors.Trace(ErrWarnDataOutOfRange)
	}

	return y, nil
}

// NewDuration construct duration with time.
func NewDuration(hour, minute, second, microsecond int, fsp int) Duration {
	return Duration{
		Duration: gotime.Duration(hour)*gotime.Hour + gotime.Duration(minute)*gotime.Minute + gotime.Duration(second)*gotime.Second + gotime.Duration(microsecond)*gotime.Microsecond, //nolint:durationcheck
		Fsp:      fsp,
	}
}

// Duration is the type for MySQL TIME type.
type Duration struct {
	gotime.Duration
	// Fsp is short for Fractional Seconds Precision.
	// See http://dev.mysql.com/doc/refman/5.7/en/fractional-seconds.html
	Fsp int
}

// MaxMySQLDuration returns Duration with maximum mysql time.
func MaxMySQLDuration(fsp int) Duration {
	return NewDuration(TimeMaxHour, TimeMaxMinute, TimeMaxSecond, 0, fsp)
}

// Neg negative d, returns a duration value.
func (d Duration) Neg() Duration {
	return Duration{
		Duration: -d.Duration,
		Fsp:      d.Fsp,
	}
}

// Add adds d to d, returns a duration value.
func (d Duration) Add(v Duration) (Duration, error) {
	if v == (Duration{}) {
		return d, nil
	}
	dsum, err := AddInt64(int64(d.Duration), int64(v.Duration))
	if err != nil {
		return Duration{}, errors.Trace(err)
	}
	if d.Fsp >= v.Fsp {
		return Duration{Duration: gotime.Duration(dsum), Fsp: d.Fsp}, nil
	}
	return Duration{Duration: gotime.Duration(dsum), Fsp: v.Fsp}, nil
}

// Sub subtracts d to d, returns a duration value.
func (d Duration) Sub(v Duration) (Duration, error) {
	if v == (Duration{}) {
		return d, nil
	}
	dsum, err := SubInt64(int64(d.Duration), int64(v.Duration))
	if err != nil {
		return Duration{}, errors.Trace(err)
	}
	if d.Fsp >= v.Fsp {
		return Duration{Duration: gotime.Duration(dsum), Fsp: d.Fsp}, nil
	}
	return Duration{Duration: gotime.Duration(dsum), Fsp: v.Fsp}, nil
}

// DurationFormat returns a textual representation of the duration value formatted
// according to layout.
// See http://dev.mysql.com/doc/refman/5.7/en/date-and-time-functions.html#function_date-format
func (d Duration) DurationFormat(layout string) (string, error) {
	var buf bytes.Buffer
	inPatternMatch := false
	for _, b := range layout {
		if inPatternMatch {
			if err := d.convertDateFormat(b, &buf); err != nil {
				return "", errors.Trace(err)
			}
			inPatternMatch = false
			continue
		}

		// It's not in pattern match now.
		if b == '%' {
			inPatternMatch = true
		} else {
			buf.WriteRune(b)
		}
	}
	return buf.String(), nil
}

func (d Duration) convertDateFormat(b rune, buf *bytes.Buffer) error {
	switch b {
	case 'H':
		buf.WriteString(FormatIntWidthN(d.Hour(), 2))
	case 'k':
		buf.WriteString(strconv.FormatInt(int64(d.Hour()), 10))
	case 'h', 'I':
		t := d.Hour()
		if t%12 == 0 {
			buf.WriteString("12")
		} else {
			buf.WriteString(FormatIntWidthN(t%12, 2))
		}
	case 'l':
		t := d.Hour()
		if t%12 == 0 {
			buf.WriteString("12")
		} else {
			buf.WriteString(strconv.FormatInt(int64(t%12), 10))
		}
	case 'i':
		buf.WriteString(FormatIntWidthN(d.Minute(), 2))
	case 'p':
		hour := d.Hour()
		if hour/12%2 == 0 {
			buf.WriteString("AM")
		} else {
			buf.WriteString("PM")
		}
	case 'r':
		h := d.Hour()
		h %= 24
		switch {
		case h == 0:
			fmt.Fprintf(buf, "%02d:%02d:%02d AM", 12, d.Minute(), d.Second())
		case h == 12:
			fmt.Fprintf(buf, "%02d:%02d:%02d PM", 12, d.Minute(), d.Second())
		case h < 12:
			fmt.Fprintf(buf, "%02d:%02d:%02d AM", h, d.Minute(), d.Second())
		default:
			fmt.Fprintf(buf, "%02d:%02d:%02d PM", h-12, d.Minute(), d.Second())
		}
	case 'T':
		fmt.Fprintf(buf, "%02d:%02d:%02d", d.Hour(), d.Minute(), d.Second())
	case 'S', 's':
		buf.WriteString(FormatIntWidthN(d.Second(), 2))
	case 'f':
		fmt.Fprintf(buf, "%06d", d.MicroSecond())
	default:
		buf.WriteRune(b)
	}

	return nil
}

// String returns the time formatted using default TimeFormat and fsp.
func (d Duration) String() string {
	var buf bytes.Buffer

	sign, hours, minutes, seconds, fraction := splitDuration(d.Duration)
	if sign < 0 {
		buf.WriteByte('-')
	}

	fmt.Fprintf(&buf, "%02d:%02d:%02d", hours, minutes, seconds)
	if d.Fsp > 0 {
		buf.WriteString(".")
		buf.WriteString(d.formatFrac(fraction))
	}

	p := buf.String()

	return p
}

func (d Duration) formatFrac(frac int) string {
	s := fmt.Sprintf("%06d", frac)
	return s[0:d.Fsp]
}

// ToNumber changes duration to number format.
// e.g,
// 10:10:10 -> 101010
func (d Duration) ToNumber() *MyDecimal {
	sign, hours, minutes, seconds, fraction := splitDuration(d.Duration)
	var (
		s       string
		signStr string
	)

	if sign < 0 {
		signStr = "-"
	}

	if d.Fsp == 0 {
		s = fmt.Sprintf("%s%02d%02d%02d", signStr, hours, minutes, seconds)
	} else {
		s = fmt.Sprintf("%s%02d%02d%02d.%s", signStr, hours, minutes, seconds, d.formatFrac(fraction))
	}

	// We skip checking error here because time formatted string can be parsed certainly.
	dec := new(MyDecimal)
	err := dec.FromString([]byte(s))
	Log(errors.Trace(err))
	return dec
}

// ConvertToTime converts duration to Time.
// Tp is TypeDatetime, TypeTimestamp and TypeDate.
func (d Duration) ConvertToTime(ctx Context, tp uint8) (Time, error) {
	year, month, day := gotime.Now().In(ctx.Location()).Date()
	datePart := FromDate(year, int(month), day, 0, 0, 0, 0)
	mixDateAndDuration(&datePart, d)

	t := NewTime(datePart, mysql.TypeDatetime, d.Fsp)
	return t.Convert(ctx, tp)
}

// ConvertToTimeWithTimestamp converts duration to Time by system timestamp.
// Tp is TypeDatetime, TypeTimestamp and TypeDate.
func (d Duration) ConvertToTimeWithTimestamp(ctx Context, tp uint8, ts gotime.Time) (Time, error) {
	year, month, day := ts.In(ctx.Location()).Date()
	datePart := FromDate(year, int(month), day, 0, 0, 0, 0)
	mixDateAndDuration(&datePart, d)

	t := NewTime(datePart, mysql.TypeDatetime, d.Fsp)
	return t.Convert(ctx, tp)
}

// ConvertToYear converts duration to Year.
func (d Duration) ConvertToYear(ctx Context) (int64, error) {
	return d.ConvertToYearFromNow(ctx, gotime.Now())
}

// ConvertToYearFromNow converts duration to Year, with the `now` specified by the argument.
func (d Duration) ConvertToYearFromNow(ctx Context, now gotime.Time) (int64, error) {
	if ctx.Flags().CastTimeToYearThroughConcat() {
		// this error will never happen, because we always give a valid FSP
		dur, _ := d.RoundFrac(DefaultFsp, ctx.Location())
		// the range of a duration will never exceed the range of `mysql.TypeLonglong`
		ival, _ := dur.ToNumber().ToInt()

		return AdjustYear(ival, false)
	}

	year, month, day := now.In(ctx.Location()).Date()
	datePart := FromDate(year, int(month), day, 0, 0, 0, 0)
	mixDateAndDuration(&datePart, d)

	return AdjustYear(int64(datePart.Year()), false)
}

// RoundFrac rounds fractional seconds precision with new fsp and returns a new one.
// We will use the “round half up” rule, e.g, >= 0.5 -> 1, < 0.5 -> 0,
// so 10:10:10.999999 round 0 -> 10:10:11
// and 10:10:10.000000 round 0 -> 10:10:10
func (d Duration) RoundFrac(fsp int, loc *gotime.Location) (Duration, error) {
	tz := loc
	if tz == nil {
		BgLogger.Warn("use gotime.local because sc.timezone is nil")
		tz = gotime.Local
	}

	fsp, err := CheckFsp(fsp)
	if err != nil {
		return d, errors.Trace(err)
	}

	if fsp == d.Fsp {
		return d, nil
	}

	n := gotime.Date(0, 0, 0, 0, 0, 0, 0, tz)
	nd := n.Add(d.Duration).Round(gotime.Duration(math.Pow10(9-fsp)) * gotime.Nanosecond).Sub(n) //nolint:durationcheck
	return Duration{Duration: nd, Fsp: fsp}, nil
}

// Compare returns an integer comparing the Duration instant t to o.
// If d is after o, returns 1, equal o, returns 0, before o, returns -1.
func (d Duration) Compare(o Duration) int {
	if d.Duration > o.Duration {
		return 1
	} else if d.Duration == o.Duration {
		return 0
	}
	return -1
}

// CompareString is like Compare,
// but parses str to Duration then compares.
func (d Duration) CompareString(ctx Context, str string) (int, error) {
	// use MaxFsp to parse the string
	o, _, err := ParseDuration(ctx, str, MaxFsp)
	if err != nil {
		return 0, err
	}

	return d.Compare(o), nil
}

// Hour returns current hour.
// e.g, hour("11:11:11") -> 11
func (d Duration) Hour() int {
	_, hour, _, _, _ := splitDuration(d.Duration)
	return hour
}

// Minute returns current minute.
// e.g, hour("11:11:11") -> 11
func (d Duration) Minute() int {
	_, _, minute, _, _ := splitDuration(d.Duration)
	return minute
}

// Second returns current second.
// e.g, hour("11:11:11") -> 11
func (d Duration) Second() int {
	_, _, _, second, _ := splitDuration(d.Duration)
	return second
}

// MicroSecond returns current microsecond.
// e.g, hour("11:11:11.11") -> 110000
func (d Duration) MicroSecond() int {
	_, _, _, _, frac := splitDuration(d.Duration)
	return frac
}

func isNegativeDuration(str string) (bool, string) {
	rest, err := Char(str, '-')

	if err != nil {
		return false, str
	}

	return true, rest
}

func matchColon(str string) (string, error) {
	rest := Space0(str)
	rest, err := Char(rest, ':')
	if err != nil {
		return str, err
	}
	rest = Space0(rest)
	return rest, nil
}

func matchDayHHMMSS(str string) (int, [3]int, string, error) {
	day, rest, err := Number(str)
	if err != nil {
		return 0, [3]int{}, str, err
	}

	rest, err = Space(rest, 1)
	if err != nil {
		return 0, [3]int{}, str, err
	}

	hhmmss, rest, err := matchHHMMSSDelimited(rest, false)
	if err != nil {
		return 0, [3]int{}, str, err
	}

	return day, hhmmss, rest, nil
}

func matchHHMMSSDelimited(str string, requireColon bool) ([3]int, string, error) {
	hhmmss := [3]int{}

	hour, rest, err := Number(str)
	if err != nil {
		return [3]int{}, str, err
	}
	hhmmss[0] = hour

	for i := 1; i < 3; i++ {
		remain, err := matchColon(rest)
		if err != nil {
			if i == 1 && requireColon {
				return [3]int{}, str, err
			}
			break
		}
		num, remain, err := Number(remain)
		if err != nil {
			return [3]int{}, str, err
		}
		hhmmss[i] = num
		rest = remain
	}

	return hhmmss, rest, nil
}

func matchHHMMSSCompact(str string) ([3]int, string, error) {
	num, rest, err := Number(str)
	if err != nil {
		return [3]int{}, str, err
	}
	hhmmss := [3]int{num / 10000, (num / 100) % 100, num % 100}
	return hhmmss, rest, nil
}

func hhmmssAddOverflow(hms []int, overflow bool) {
	mod := []int{-1, 60, 60}
	for i := 2; i >= 0 && overflow; i-- {
		hms[i]++
		if hms[i] == mod[i] {
			overflow = true
			hms[i] = 0
		} else {
			overflow = false
		}
	}
}

func checkHHMMSS(hms [3]int) bool {
	m, s := hms[1], hms[2]
	return m < 60 && s < 60
}

// matchFrac returns overflow, fraction, rest, error
func matchFrac(str string, fsp int) (bool, int, string, error) {
	rest, err := Char(str, '.')
	if err != nil {
		return false, 0, str, nil
	}

	digits, rest, err := Digit(rest, 0)
	if err != nil {
		return false, 0, str, err
	}

	frac, overflow, err := ParseFrac(digits, fsp)
	if err != nil {
		return false, 0, str, err
	}

	return overflow, frac, rest, nil
}

func matchDuration(str string, fsp int) (Duration, bool, error) {
	fsp, err := CheckFsp(fsp)
	if err != nil {
		return ZeroDuration, true, errors.Trace(err)
	}

	if len(str) == 0 {
		return ZeroDuration, true, ErrTruncatedWrongVal.GenWithStackByArgs("time", str)
	}

	negative, rest := isNegativeDuration(str)
	rest = Space0(rest)
	charsLen := len(rest)

	hhmmss := [3]int{}

	if day, hms, remain, err := matchDayHHMMSS(rest); err == nil {
		hms[0] += 24 * day
		rest, hhmmss = remain, hms
	} else if hms, remain, err := matchHHMMSSDelimited(rest, true); err == nil {
		rest, hhmmss = remain, hms
	} else if hms, remain, err := matchHHMMSSCompact(rest); err == nil {
		rest, hhmmss = remain, hms
	} else {
		return ZeroDuration, true, ErrTruncatedWrongVal.GenWithStackByArgs("time", str)
	}

	rest = Space0(rest)
	overflow, frac, rest, err := matchFrac(rest, fsp)
	if err != nil || (len(rest) > 0 && charsLen >= 12) {
		return ZeroDuration, true, ErrTruncatedWrongVal.GenWithStackByArgs("time", str)
	}

	if overflow {
		hhmmssAddOverflow(hhmmss[:], overflow)
		frac = 0
	}

	if !checkHHMMSS(hhmmss) {
		return ZeroDuration, true, ErrTruncatedWrongVal.GenWithStackByArgs("time", str)
	}

	if hhmmss[0] > TimeMaxHour {
		var t gotime.Duration
		if negative {
			t = MinTime
		} else {
			t = MaxTime
		}
		return Duration{t, fsp}, false, ErrTruncatedWrongVal.GenWithStackByArgs("time", str)
	}

	d := gotime.Duration(hhmmss[0]*3600+hhmmss[1]*60+hhmmss[2])*gotime.Second + gotime.Duration(frac)*gotime.Microsecond //nolint:durationcheck
	if negative {
		d = -d
	}
	d, err = TruncateOverflowMySQLTime(d)
	if err == nil && len(rest) > 0 {
		return Duration{d, fsp}, false, ErrTruncatedWrongVal.GenWithStackByArgs("time", str)
	}
	return Duration{d, fsp}, false, errors.Trace(err)
}

// canFallbackToDateTime return true
//  1. the string is failed to be parsed by `matchDuration`
//  2. the string is start with a series of digits whose length match the full format of DateTime literal (12, 14)
//     or the string start with a date literal.
func canFallbackToDateTime(str string) bool {
	digits, rest, err := Digit(str, 1)
	if err != nil {
		return false
	}
	if len(digits) == 12 || len(digits) == 14 {
		return true
	}

	rest, err = AnyPunct(rest)
	if err != nil {
		return false
	}

	_, rest, err = Digit(rest, 1)
	if err != nil {
		return false
	}

	rest, err = AnyPunct(rest)
	if err != nil {
		return false
	}

	_, rest, err = Digit(rest, 1)
	if err != nil {
		return false
	}

	return len(rest) > 0 && (rest[0] == ' ' || rest[0] == 'T')
}

// ParseDuration parses the time form a formatted string with a fractional seconds part,
// returns the duration type Time value and bool to indicate whether the result is null.
// See http://dev.mysql.com/doc/refman/5.7/en/fractional-seconds.html
func ParseDuration(ctx Context, str string, fsp int) (Duration, bool, error) {
	rest := strings.TrimSpace(str)
	d, isNull, err := matchDuration(rest, fsp)
	if err == nil {
		return d, isNull, nil
	}
	if !canFallbackToDateTime(rest) {
		return d, isNull, ErrTruncatedWrongVal.GenWithStackByArgs("time", str)
	}

	datetime, err := ParseDatetime(ctx, rest)
	if err != nil {
		return ZeroDuration, true, ErrTruncatedWrongVal.GenWithStackByArgs("time", str)
	}

	d, err = datetime.ConvertToDuration()
	if err != nil {
		return ZeroDuration, true, ErrTruncatedWrongVal.GenWithStackByArgs("time", str)
	}

	d, err = d.RoundFrac(fsp, ctx.Location())
	return d, false, err
}

// TruncateOverflowMySQLTime truncates d when it overflows, and returns ErrTruncatedWrongVal.
func TruncateOverflowMySQLTime(d gotime.Duration) (gotime.Duration, error) {
	if d > MaxTime {
		return MaxTime, ErrTruncatedWrongVal.GenWithStackByArgs("time", d)
	} else if d < MinTime {
		return MinTime, ErrTruncatedWrongVal.GenWithStackByArgs("time", d)
	}

	return d, nil
}

func splitDuration(t gotime.Duration) (sign int, hours int, minutes int, seconds int, fraction int) {
	sign = 1
	if t < 0 {
		t = -t
		sign = -1
	}

	hoursDuration := t / gotime.Hour
	t -= hoursDuration * gotime.Hour //nolint:durationcheck
	minutesDuration := t / gotime.Minute
	t -= minutesDuration * gotime.Minute //nolint:durationcheck
	secondsDuration := t / gotime.Second
	t -= secondsDuration * gotime.Second //nolint:durationcheck
	fractionDuration := t / gotime.Microsecond

	return sign, int(hoursDuration), int(minutesDuration), int(secondsDuration), int(fractionDuration)
}

var maxDaysInMonth = []int{31, 29, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}

func getTime(ctx Context, num, originNum int64, tp byte) (Time, error) {
	s1 := num / 1000000
	s2 := num - s1*1000000

	year := int(s1 / 10000)
	s1 %= 10000
	month := int(s1 / 100)
	day := int(s1 % 100)

	hour := int(s2 / 10000)
	s2 %= 10000
	minute := int(s2 / 100)
	second := int(s2 % 100)

	ct, ok := FromDateChecked(year, month, day, hour, minute, second, 0)
	if !ok {
		numStr := strconv.FormatInt(originNum, 10)
		return ZeroDatetime, errors.Trace(ErrWrongValue.GenWithStackByArgs(TimeStr, numStr))
	}
	t := NewTime(ct, tp, DefaultFsp)
	err := t.Check(ctx)
	return t, errors.Trace(err)
}

// parseDateTimeFromNum parses date time from num.
// See number_to_datetime function.
// https://github.com/mysql/mysql-server/blob/5.7/sql-common/my_time.c
func parseDateTimeFromNum(ctx Context, num int64) (Time, error) {
	t := ZeroDate
	// Check zero.
	if num == 0 {
		return t, nil
	}
	originNum := num

	// Check datetime type.
	if num >= 10000101000000 {
		t.SetType(mysql.TypeDatetime)
		return getTime(ctx, num, originNum, t.Type())
	}

	// Check MMDD.
	if num < 101 {
		return t, errors.Trace(ErrWrongValue.GenWithStackByArgs(TimeStr, strconv.FormatInt(num, 10)))
	}

	// Adjust year
	// YYMMDD, year: 2000-2069
	if num <= (70-1)*10000+1231 {
		num = (num + 20000000) * 1000000
		return getTime(ctx, num, originNum, t.Type())
	}

	// Check YYMMDD.
	if num < 70*10000+101 {
		return t, errors.Trace(ErrWrongValue.GenWithStackByArgs(TimeStr, strconv.FormatInt(num, 10)))
	}

	// Adjust year
	// YYMMDD, year: 1970-1999
	if num <= 991231 {
		num = (num + 19000000) * 1000000
		return getTime(ctx, num, originNum, t.Type())
	}

	// Adjust hour/min/second.
	if num <= 99991231 {
		num = num * 1000000
		return getTime(ctx, num, originNum, t.Type())
	}

	// Check MMDDHHMMSS.
	if num < 101000000 {
		return t, errors.Trace(ErrWrongValue.GenWithStackByArgs(TimeStr, strconv.FormatInt(num, 10)))
	}

	// Set TypeDatetime type.
	t.SetType(mysql.TypeDatetime)

	// Adjust year
	// YYMMDDHHMMSS, 2000-2069
	if num <= 69*10000000000+1231235959 {
		num = num + 20000000000000
		return getTime(ctx, num, originNum, t.Type())
	}

	// Check YYYYMMDDHHMMSS.
	if num < 70*10000000000+101000000 {
		return t, errors.Trace(ErrWrongValue.GenWithStackByArgs(TimeStr, strconv.FormatInt(num, 10)))
	}

	// Adjust year
	// YYMMDDHHMMSS, 1970-1999
	if num <= 991231235959 {
		num = num + 19000000000000
		return getTime(ctx, num, originNum, t.Type())
	}

	return getTime(ctx, num, originNum, t.Type())
}

// ParseTime parses a formatted string with type tp and specific fsp.
// Type is TypeDatetime, TypeTimestamp and TypeDate.
// Fsp is in range [0, 6].
// MySQL supports many valid datetime format, but still has some limitation.
// If delimiter exists, the date part and time part is separated by a space or T,
// other punctuation character can be used as the delimiter between date parts or time parts.
// If no delimiter, the format must be YYYYMMDDHHMMSS or YYMMDDHHMMSS
// If we have fractional seconds part, we must use decimal points as the delimiter.
// The valid datetime range is from '1000-01-01 00:00:00.000000' to '9999-12-31 23:59:59.999999'.
// The valid timestamp range is from '1970-01-01 00:00:01.000000' to '2038-01-19 03:14:07.999999'.
// The valid date range is from '1000-01-01' to '9999-12-31'
// explicitTz is used to handle a data race of timeZone, refer to https://github.com/pingcap/tidb/issues/40710. It only works for timestamp now, be careful to use it!
func ParseTime(ctx Context, str string, tp byte, fsp int) (Time, error) {
	return parseTime(ctx, str, tp, fsp, false)
}

// ParseTimeFromFloatString is similar to ParseTime, except that it's used to parse a float converted string.
func ParseTimeFromFloatString(ctx Context, str string, tp byte, fsp int) (Time, error) {
	// MySQL compatibility: 0.0 should not be converted to null, see #11203
	if len(str) >= 3 && str[:3] == "0.0" {
		return NewTime(ZeroCoreTime, tp, DefaultFsp), nil
	}
	return parseTime(ctx, str, tp, fsp, true)
}

func parseTime(ctx Context, str string, tp byte, fsp int, isFloat bool) (Time, error) {
	fsp, err := CheckFsp(fsp)
	if err != nil {
		return NewTime(ZeroCoreTime, tp, DefaultFsp), errors.Trace(err)
	}

	t, err := parseDatetime(ctx, str, fsp, isFloat)
	if err != nil {
		return NewTime(ZeroCoreTime, tp, DefaultFsp), errors.Trace(err)
	}

	t.SetType(tp)
	if err = t.Check(ctx); err != nil {
		if tp == mysql.TypeTimestamp && !t.IsZero() {
			tAdjusted, errAdjusted := adjustTimestampErrForDST(ctx.Location(), str, tp, t, err)
			if ErrTimestampInDSTTransition.Equal(errAdjusted) {
				return tAdjusted, errors.Trace(errAdjusted)
			}
		}
		return NewTime(ZeroCoreTime, tp, DefaultFsp), errors.Trace(err)
	}
	return t, nil
}

func adjustTimestampErrForDST(loc *gotime.Location, str string, tp byte, t Time, err error) (Time, error) {
	if tp != mysql.TypeTimestamp || t.IsZero() {
		return t, err
	}
	minTS, maxTS := MinTimestamp, MaxTimestamp
	minErr := minTS.ConvertTimeZone(gotime.UTC, loc)
	maxErr := maxTS.ConvertTimeZone(gotime.UTC, loc)
	if minErr == nil && maxErr == nil &&
		t.Compare(minTS) > 0 && t.Compare(maxTS) < 0 {
		// Handle the case when the timestamp given is in the DST transition
		if tAdjusted, err2 := t.AdjustedGoTime(loc); err2 == nil {
			t.SetCoreTime(FromGoTime(tAdjusted))
			return t, errors.Trace(ErrTimestampInDSTTransition.GenWithStackByArgs(str, loc.String()))
		}
	}
	return t, err
}

// ParseDatetime is a helper function wrapping ParseTime with datetime type and default fsp.
func ParseDatetime(ctx Context, str string) (Time, error) {
	return ParseTime(ctx, str, mysql.TypeDatetime, GetFsp(str))
}

// ParseTimestamp is a helper function wrapping ParseTime with timestamp type and default fsp.
func ParseTimestamp(ctx Context, str string) (Time, error) {
	return ParseTime(ctx, str, mysql.TypeTimestamp, GetFsp(str))
}

// ParseDate is a helper function wrapping ParseTime with date type.
func ParseDate(ctx Context, str string) (Time, error) {
	// date has no fractional seconds precision
	return ParseTime(ctx, str, mysql.TypeDate, MinFsp)
}

// ParseTimeFromYear parse a `YYYY` formed year to corresponded Datetime type.
// Note: the invoker must promise the `year` is in the range [MinYear, MaxYear].
func ParseTimeFromYear(year int64) (Time, error) {
	if year == 0 {
		return NewTime(ZeroCoreTime, mysql.TypeDate, DefaultFsp), nil
	}

	dt := FromDate(int(year), 0, 0, 0, 0, 0, 0)
	return NewTime(dt, mysql.TypeDatetime, DefaultFsp), nil
}

// ParseTimeFromNum parses a formatted int64,
// returns the value which type is tp.
func ParseTimeFromNum(ctx Context, num int64, tp byte, fsp int) (Time, error) {
	// MySQL compatibility: 0 should not be converted to null, see #11203
	if num == 0 {
		zt := NewTime(ZeroCoreTime, tp, DefaultFsp)
		if !ctx.Flags().IgnoreZeroDateErr() {
			switch tp {
			case mysql.TypeTimestamp:
				return zt, ErrTruncatedWrongVal.GenWithStackByArgs(TimestampStr, "0")
			case mysql.TypeDate:
				return zt, ErrTruncatedWrongVal.GenWithStackByArgs(DateStr, "0")
			case mysql.TypeDatetime:
				return zt, ErrTruncatedWrongVal.GenWithStackByArgs(DateTimeStr, "0")
			}
		}
		return zt, nil
	}
	fsp, err := CheckFsp(fsp)
	if err != nil {
		return NewTime(ZeroCoreTime, tp, DefaultFsp), errors.Trace(err)
	}

	t, err := parseDateTimeFromNum(ctx, num)
	if err != nil {
		return NewTime(ZeroCoreTime, tp, DefaultFsp), errors.Trace(err)
	}

	t.SetType(tp)
	t.SetFsp(fsp)
	if err := t.Check(ctx); err != nil {
		return NewTime(ZeroCoreTime, tp, DefaultFsp), errors.Trace(err)
	}
	return t, nil
}

// ParseDatetimeFromNum is a helper function wrapping ParseTimeFromNum with datetime type and default fsp.
func ParseDatetimeFromNum(ctx Context, num int64) (Time, error) {
	return ParseTimeFromNum(ctx, num, mysql.TypeDatetime, DefaultFsp)
}

// ParseTimestampFromNum is a helper function wrapping ParseTimeFromNum with timestamp type and default fsp.
func ParseTimestampFromNum(ctx Context, num int64) (Time, error) {
	return ParseTimeFromNum(ctx, num, mysql.TypeTimestamp, DefaultFsp)
}

// ParseDateFromNum is a helper function wrapping ParseTimeFromNum with date type.
func ParseDateFromNum(ctx Context, num int64) (Time, error) {
	// date has no fractional seconds precision
	return ParseTimeFromNum(ctx, num, mysql.TypeDate, MinFsp)
}

// TimeFromDays Converts a day number to a date.
func TimeFromDays(num int64) Time {
	if num < 0 {
		return NewTime(FromDate(0, 0, 0, 0, 0, 0, 0), mysql.TypeDate, 0)
	}
	year, month, day := getDateFromDaynr(uint(num))
	ct, ok := FromDateChecked(int(year), int(month), int(day), 0, 0, 0, 0)
	if !ok {
		return NewTime(FromDate(0, 0, 0, 0, 0, 0, 0), mysql.TypeDate, 0)
	}
	return NewTime(ct, mysql.TypeDate, 0)
}

func checkDateType(t CoreTime, allowZeroInDate, allowInvalidDate bool) error {
	year, month, day := t.Year(), t.Month(), t.Day()
	if year == 0 && month == 0 && day == 0 {
		return nil
	}

	if !allowZeroInDate && (month == 0 || day == 0) {
		return ErrWrongValue.GenWithStackByArgs(DateTimeStr, fmt.Sprintf("%04d-%02d-%02d", year, month, day))
	}

	if err := checkDateRange(t); err != nil {
		return errors.Trace(err)
	}

	if err := checkMonthDay(year, month, day, allowInvalidDate); err != nil {
		return errors.Trace(err)
	}

	return nil
}

func checkDateRange(t CoreTime) error {
	// Oddly enough, MySQL document says date range should larger than '1000-01-01',
	// but we can insert '0001-01-01' actually.
	if t.Year() < 0 || t.Month() < 0 || t.Day() < 0 {
		return errors.Trace(ErrWrongValue.GenWithStackByArgs(TimeStr, t))
	}
	if compareTime(t, MaxDatetime) > 0 {
		return errors.Trace(ErrWrongValue.GenWithStackByArgs(TimeStr, t))
	}
	return nil
}

func checkMonthDay(year, month, day int, allowInvalidDate bool) error {
	if month < 0 || month > 12 {
		return errors.Trace(ErrWrongValue.GenWithStackByArgs(DateTimeStr, fmt.Sprintf("%d-%d-%d", year, month, day)))
	}

	maxDay := 31
	if !allowInvalidDate {
		if month > 0 {
			maxDay = maxDaysInMonth[month-1]
		}
		if month == 2 && !isLeapYear(uint16(year)) {
			maxDay = 28
		}
	}

	if day < 0 || day > maxDay {
		return errors.Trace(ErrWrongValue.GenWithStackByArgs(DateTimeStr, fmt.Sprintf("%d-%d-%d", year, month, day)))
	}
	return nil
}

func checkTimestampType(t CoreTime, tz *gotime.Location) error {
	if compareTime(t, ZeroCoreTime) == 0 {
		return nil
	}

	var checkTime CoreTime
	if tz != BoundTimezone {
		convertTime := NewTime(t, mysql.TypeTimestamp, DefaultFsp)
		err := convertTime.ConvertTimeZone(tz, BoundTimezone)
		if err != nil {
			_, err2 := adjustTimestampErrForDST(tz, t.String(), mysql.TypeTimestamp, Time{t}, err)
			return err2
		}
		checkTime = convertTime.coreTime
	} else {
		checkTime = t
	}
	if compareTime(checkTime, MaxTimestamp.coreTime) > 0 || compareTime(checkTime, MinTimestamp.coreTime) < 0 {
		return errors.Trace(ErrWrongValue.GenWithStackByArgs(TimeStr, t))
	}

	if _, err := t.GoTime(tz); err != nil {
		return errors.Trace(err)
	}

	return nil
}

func checkDatetimeType(t CoreTime, allowZeroInDate, allowInvalidDate bool) error {
	if err := checkDateType(t, allowZeroInDate, allowInvalidDate); err != nil {
		return errors.Trace(err)
	}

	hour, minute, second := t.Hour(), t.Minute(), t.Second()
	if hour < 0 || hour >= 24 {
		return errors.Trace(ErrWrongValue.GenWithStackByArgs(TimeStr, strconv.Itoa(hour)))
	}
	if minute < 0 || minute >= 60 {
		return errors.Trace(ErrWrongValue.GenWithStackByArgs(TimeStr, strconv.Itoa(minute)))
	}
	if second < 0 || second >= 60 {
		return errors.Trace(ErrWrongValue.GenWithStackByArgs(TimeStr, strconv.Itoa(second)))
	}

	return nil
}

// ExtractDatetimeNum extracts time value number from datetime unit and format.
func ExtractDatetimeNum(t *Time, unit string) (int64, error) {
	// TODO: Consider time_zone variable.
	switch strings.ToUpper(unit) {
	case "DAY":
		return int64(t.Day()), nil
	case "WEEK":
		week := t.Week(0)
		return int64(week), nil
	case "MONTH":
		return int64(t.Month()), nil
	case "QUARTER":
		m := int64(t.Month())
		// 1 - 3 -> 1
		// 4 - 6 -> 2
		// 7 - 9 -> 3
		// 10 - 12 -> 4
		return (m + 2) / 3, nil
	case "YEAR":
		return int64(t.Year()), nil
	case "DAY_MICROSECOND":
		h, m, s := t.Clock()
		d := t.Day()
		return int64(d*1000000+h*10000+m*100+s)*1000000 + int64(t.Microsecond()), nil
	case "DAY_SECOND":
		h, m, s := t.Clock()
		d := t.Day()
		return int64(d)*1000000 + int64(h)*10000 + int64(m)*100 + int64(s), nil
	case "DAY_MINUTE":
		h, m, _ := t.Clock()
		d := t.Day()
		return int64(d)*10000 + int64(h)*100 + int64(m), nil
	case "DAY_HOUR":
		h, _, _ := t.Clock()
		d := t.Day()
		return int64(d)*100 + int64(h), nil
	case "YEAR_MONTH":
		y, m := t.Year(), t.Month()
		return int64(y)*100 + int64(m), nil
	default:
		return 0, errors.Errorf("invalid unit %s", unit)
	}
}

// ExtractDurationNum extracts duration value number from duration unit and format.
func ExtractDurationNum(d *Duration, unit string) (res int64, err error) {
	switch strings.ToUpper(unit) {
	case "MICROSECOND":
		res = int64(d.MicroSecond())
	case "SECOND":
		res = int64(d.Second())
	case "MINUTE":
		res = int64(d.Minute())
	case "HOUR":
		res = int64(d.Hour())
	case "SECOND_MICROSECOND":
		res = int64(d.Second())*1000000 + int64(d.MicroSecond())
	case "MINUTE_MICROSECOND":
		res = int64(d.Minute())*100000000 + int64(d.Second())*1000000 + int64(d.MicroSecond())
	case "MINUTE_SECOND":
		res = int64(d.Minute()*100 + d.Second())
	case "HOUR_MICROSECOND":
		res = int64(d.Hour())*10000000000 + int64(d.Minute())*100000000 + int64(d.Second())*1000000 + int64(d.MicroSecond())
	case "HOUR_SECOND":
		res = int64(d.Hour())*10000 + int64(d.Minute())*100 + int64(d.Second())
	case "HOUR_MINUTE":
		res = int64(d.Hour())*100 + int64(d.Minute())
	case "DAY_MICROSECOND":
		res = int64(d.Hour()*10000+d.Minute()*100+d.Second())*1000000 + int64(d.MicroSecond())
	case "DAY_SECOND":
		res = int64(d.Hour())*10000 + int64(d.Minute())*100 + int64(d.Second())
	case "DAY_MINUTE":
		res = int64(d.Hour())*100 + int64(d.Minute())
	case "DAY_HOUR":
		res = int64(d.Hour())
	default:
		return 0, errors.Errorf("invalid unit %s", unit)
	}
	if d.Duration < 0 {
		res = -res
	}
	return res, nil
}

// parseSingleTimeValue parse the format according the given unit. If we set strictCheck true, we'll check whether
// the converted value not exceed the range of MySQL's TIME type.
// The returned values are year, month, day, nanosecond and fsp.
func parseSingleTimeValue(unit string, format string, strictCheck bool) (year int64, month int64, day int64, nanosecond int64, fsp int, err error) {
	// Format is a preformatted number, it format should be A[.[B]].
	decimalPointPos := strings.IndexRune(format, '.')
	if decimalPointPos == -1 {
		decimalPointPos = len(format)
	}
	sign := int64(1)
	if len(format) > 0 && format[0] == '-' {
		sign = int64(-1)
	}

	// We should also continue even if an error occurs here
	// because the called may ignore the error and use the return value.
	iv, err := strconv.ParseInt(format[0:decimalPointPos], 10, 64)
	if err != nil {
		err = ErrWrongValue.GenWithStackByArgs(DateTimeStr, format)
	}
	riv := iv // Rounded integer value

	decimalLen := 0
	dv := int64(0)
	lf := len(format) - 1
	// Has fraction part
	if decimalPointPos < lf {
		var tmpErr error
		dvPre := oneToSixDigitRegex.FindString(format[decimalPointPos+1:]) // the numberical prefix of the fraction part
		decimalLen = len(dvPre)
		if decimalLen >= 6 {
			// MySQL rounds down to 1e-6.
			if dv, tmpErr = strconv.ParseInt(dvPre[0:6], 10, 64); tmpErr != nil && err == nil {
				err = ErrWrongValue.GenWithStackByArgs(DateTimeStr, format)
			}
		} else {
			if dv, tmpErr = strconv.ParseInt(dvPre+"000000"[:6-decimalLen], 10, 64); tmpErr != nil && err == nil {
				err = ErrWrongValue.GenWithStackByArgs(DateTimeStr, format)
			}
		}
		if dv >= 500000 { // Round up, and we should keep 6 digits for microsecond, so dv should in [000000, 999999].
			riv += sign
		}
		if unit != "SECOND" && err == nil {
			err = ErrTruncatedWrongVal.GenWithStackByArgs(format)
		}
		dv *= sign
	}
	switch strings.ToUpper(unit) {
	case "MICROSECOND":
		if strictCheck && Abs(riv) > TimeMaxValueSeconds*1000 {
			return 0, 0, 0, 0, 0, ErrDatetimeFunctionOverflow.GenWithStackByArgs("time")
		}
		dayCount := riv / int64(GoDurationDay/gotime.Microsecond)
		riv %= int64(GoDurationDay / gotime.Microsecond)
		return 0, 0, dayCount, riv * int64(gotime.Microsecond), MaxFsp, err
	case "SECOND":
		if strictCheck && Abs(iv) > TimeMaxValueSeconds {
			return 0, 0, 0, 0, 0, ErrDatetimeFunctionOverflow.GenWithStackByArgs("time")
		}
		dayCount := iv / int64(GoDurationDay/gotime.Second)
		iv %= int64(GoDurationDay / gotime.Second)
		return 0, 0, dayCount, iv*int64(gotime.Second) + dv*int64(gotime.Microsecond), decimalLen, err
	case "MINUTE":
		if strictCheck && Abs(riv) > TimeMaxHour*60+TimeMaxMinute {
			return 0, 0, 0, 0, 0, ErrDatetimeFunctionOverflow.GenWithStackByArgs("time")
		}
		dayCount := riv / int64(GoDurationDay/gotime.Minute)
		riv %= int64(GoDurationDay / gotime.Minute)
		return 0, 0, dayCount, riv * int64(gotime.Minute), 0, err
	case "HOUR":
		if strictCheck && Abs(riv) > TimeMaxHour {
			return 0, 0, 0, 0, 0, ErrDatetimeFunctionOverflow.GenWithStackByArgs("time")
		}
		dayCount := riv / 24
		riv %= 24
		return 0, 0, dayCount, riv * int64(gotime.Hour), 0, err
	case "DAY":
		if strictCheck && Abs(riv) > TimeMaxHour/24 {
			return 0, 0, 0, 0, 0, ErrDatetimeFunctionOverflow.GenWithStackByArgs("time")
		}
		return 0, 0, riv, 0, 0, err
	case "WEEK":
		if strictCheck && 7*Abs(riv) > TimeMaxHour/24 {
			return 0, 0, 0, 0, 0, ErrDatetimeFunctionOverflow.GenWithStackByArgs("time")
		}
		return 0, 0, 7 * riv, 0, 0, err
	case "MONTH":
		if strictCheck && Abs(riv) > 1 {
			return 0, 0, 0, 0, 0, ErrDatetimeFunctionOverflow.GenWithStackByArgs("time")
		}
		return 0, riv, 0, 0, 0, err
	case "QUARTER":
		if strictCheck {
			return 0, 0, 0, 0, 0, ErrDatetimeFunctionOverflow.GenWithStackByArgs("time")
		}
		return 0, 3 * riv, 0, 0, 0, err
	case "YEAR":
		if strictCheck {
			return 0, 0, 0, 0, 0, ErrDatetimeFunctionOverflow.GenWithStackByArgs("time")
		}
		return riv, 0, 0, 0, 0, err
	}

	return 0, 0, 0, 0, 0, errors.Errorf("invalid singel timeunit - %s", unit)
}

// parseTimeValue gets years, months, days, nanoseconds and fsp from a string
// nanosecond will not exceed length of single day
// MySQL permits any punctuation delimiter in the expr format.
// See https://dev.mysql.com/doc/refman/8.0/en/expressions.html#temporal-intervals
func parseTimeValue(format string, index, cnt int) (years int64, months int64, days int64, nanoseconds int64, fsp int, err error) {
	neg := false
	originalFmt := format
	fsp = map[bool]int{true: MaxFsp, false: MinFsp}[index == MicrosecondIndex]
	format = strings.TrimSpace(format)
	if len(format) > 0 && format[0] == '-' {
		neg = true
		format = format[1:]
	}
	fields := make([]string, TimeValueCnt)
	for i := range fields {
		fields[i] = "0"
	}
	matches := numericRegex.FindAllString(format, -1)
	if len(matches) > cnt {
		return 0, 0, 0, 0, 0, ErrWrongValue.GenWithStackByArgs(DateTimeStr, originalFmt)
	}
	for i := range matches {
		if neg {
			fields[index] = "-" + matches[len(matches)-1-i]
		} else {
			fields[index] = matches[len(matches)-1-i]
		}
		index--
	}

	// ParseInt may return an error when overflowed, but we should continue to parse the rest of the string because
	// the caller may ignore the error and use the return value.
	// In this case, we should return a big value to make sure the result date after adding this interval
	// is also overflowed and NULL is returned to the user.
	years, err = strconv.ParseInt(fields[YearIndex], 10, 64)
	if err != nil {
		err = ErrWrongValue.GenWithStackByArgs(DateTimeStr, originalFmt)
	}
	var tmpErr error
	months, tmpErr = strconv.ParseInt(fields[MonthIndex], 10, 64)
	if err == nil && tmpErr != nil {
		err = ErrWrongValue.GenWithStackByArgs(DateTimeStr, originalFmt)
	}
	days, tmpErr = strconv.ParseInt(fields[DayIndex], 10, 64)
	if err == nil && tmpErr != nil {
		err = ErrWrongValue.GenWithStackByArgs(DateTimeStr, originalFmt)
	}

	hours, tmpErr := strconv.ParseInt(fields[HourIndex], 10, 64)
	if tmpErr != nil && err == nil {
		err = ErrWrongValue.GenWithStackByArgs(DateTimeStr, originalFmt)
	}
	minutes, tmpErr := strconv.ParseInt(fields[MinuteIndex], 10, 64)
	if tmpErr != nil && err == nil {
		err = ErrWrongValue.GenWithStackByArgs(DateTimeStr, originalFmt)
	}
	seconds, tmpErr := strconv.ParseInt(fields[SecondIndex], 10, 64)
	if tmpErr != nil && err == nil {
		err = ErrWrongValue.GenWithStackByArgs(DateTimeStr, originalFmt)
	}
	microseconds, tmpErr := strconv.ParseInt(alignFrac(fields[MicrosecondIndex], MaxFsp), 10, 64)
	if tmpErr != nil && err == nil {
		err = ErrWrongValue.GenWithStackByArgs(DateTimeStr, originalFmt)
	}
	seconds = hours*3600 + minutes*60 + seconds
	days += seconds / (3600 * 24)
	seconds %= 3600 * 24
	return years, months, days, seconds*int64(gotime.Second) + microseconds*int64(gotime.Microsecond), fsp, err
}

func parseAndValidateDurationValue(format string, index, cnt int) (int64, int, error) {
	year, month, day, nano, fsp, err := parseTimeValue(format, index, cnt)
	if err != nil {
		return 0, 0, err
	}
	if year != 0 || month != 0 || Abs(day) > TimeMaxHour/24 {
		return 0, 0, ErrDatetimeFunctionOverflow.GenWithStackByArgs("time")
	}
	dur := day*int64(GoDurationDay) + nano
	if Abs(dur) > int64(MaxTime) {
		return 0, 0, ErrDatetimeFunctionOverflow.GenWithStackByArgs("time")
	}
	return dur, fsp, nil
}

// ParseDurationValue parses time value from time unit and format.
// Returns y years m months d days + n nanoseconds
// Nanoseconds will no longer than one day.
func ParseDurationValue(unit string, format string) (y int64, m int64, d int64, n int64, fsp int, _ error) {
	switch strings.ToUpper(unit) {
	case "MICROSECOND", "SECOND", "MINUTE", "HOUR", "DAY", "WEEK", "MONTH", "QUARTER", "YEAR":
		return parseSingleTimeValue(unit, format, false)
	case "SECOND_MICROSECOND":
		return parseTimeValue(format, MicrosecondIndex, SecondMicrosecondMaxCnt)
	case "MINUTE_MICROSECOND":
		return parseTimeValue(format, MicrosecondIndex, MinuteMicrosecondMaxCnt)
	case "MINUTE_SECOND":
		return parseTimeValue(format, SecondIndex, MinuteSecondMaxCnt)
	case "HOUR_MICROSECOND":
		return parseTimeValue(format, MicrosecondIndex, HourMicrosecondMaxCnt)
	case "HOUR_SECOND":
		return parseTimeValue(format, SecondIndex, HourSecondMaxCnt)
	case "HOUR_MINUTE":
		return parseTimeValue(format, MinuteIndex, HourMinuteMaxCnt)
	case "DAY_MICROSECOND":
		return parseTimeValue(format, MicrosecondIndex, DayMicrosecondMaxCnt)
	case "DAY_SECOND":
		return parseTimeValue(format, SecondIndex, DaySecondMaxCnt)
	case "DAY_MINUTE":
		return parseTimeValue(format, MinuteIndex, DayMinuteMaxCnt)
	case "DAY_HOUR":
		return parseTimeValue(format, HourIndex, DayHourMaxCnt)
	case "YEAR_MONTH":
		return parseTimeValue(format, MonthIndex, YearMonthMaxCnt)
	default:
		return 0, 0, 0, 0, 0, errors.Errorf("invalid single timeunit - %s", unit)
	}
}

// ExtractDurationValue extract the value from format to Duration.
func ExtractDurationValue(unit string, format string) (Duration, error) {
	unit = strings.ToUpper(unit)
	switch unit {
	case "MICROSECOND", "SECOND", "MINUTE", "HOUR", "DAY", "WEEK", "MONTH", "QUARTER", "YEAR":
		_, month, day, nano, fsp, err := parseSingleTimeValue(unit, format, true)
		if err != nil {
			return ZeroDuration, err
		}
		dur := Duration{Duration: gotime.Duration((month*30+day)*int64(GoDurationDay) + nano), Fsp: fsp}
		return dur, err
	case "SECOND_MICROSECOND":
		d, fsp, err := parseAndValidateDurationValue(format, MicrosecondIndex, SecondMicrosecondMaxCnt)
		if err != nil {
			return ZeroDuration, err
		}
		return Duration{Duration: gotime.Duration(d), Fsp: fsp}, nil
	case "MINUTE_MICROSECOND":
		d, fsp, err := parseAndValidateDurationValue(format, MicrosecondIndex, MinuteMicrosecondMaxCnt)
		if err != nil {
			return ZeroDuration, err
		}
		return Duration{Duration: gotime.Duration(d), Fsp: fsp}, nil
	case "MINUTE_SECOND":
		d, fsp, err := parseAndValidateDurationValue(format, SecondIndex, MinuteSecondMaxCnt)
		if err != nil {
			return ZeroDuration, err
		}
		return Duration{Duration: gotime.Duration(d), Fsp: fsp}, nil
	case "HOUR_MICROSECOND":
		d, fsp, err := parseAndValidateDurationValue(format, MicrosecondIndex, HourMicrosecondMaxCnt)
		if err != nil {
			return ZeroDuration, err
		}
		return Duration{Duration: gotime.Duration(d), Fsp: fsp}, nil
	case "HOUR_SECOND":
		d, fsp, err := parseAndValidateDurationValue(format, SecondIndex, HourSecondMaxCnt)
		if err != nil {
			return ZeroDuration, err
		}
		return Duration{Duration: gotime.Duration(d), Fsp: fsp}, nil
	case "HOUR_MINUTE":
		d, fsp, err := parseAndValidateDurationValue(format, MinuteIndex, HourMinuteMaxCnt)
		if err != nil {
			return ZeroDuration, err
		}
		return Duration{Duration: gotime.Duration(d), Fsp: fsp}, nil
	case "DAY_MICROSECOND":
		d, fsp, err := parseAndValidateDurationValue(format, MicrosecondIndex, DayMicrosecondMaxCnt)
		if err != nil {
			return ZeroDuration, err
		}
		return Duration{Duration: gotime.Duration(d), Fsp: fsp}, nil
	case "DAY_SECOND":
		d, fsp, err := parseAndValidateDurationValue(format, SecondIndex, DaySecondMaxCnt)
		if err != nil {
			return ZeroDuration, err
		}
		return Duration{Duration: gotime.Duration(d), Fsp: fsp}, nil
	case "DAY_MINUTE":
		d, fsp, err := parseAndValidateDurationValue(format, MinuteIndex, DayMinuteMaxCnt)
		if err != nil {
			return ZeroDuration, err
		}
		return Duration{Duration: gotime.Duration(d), Fsp: fsp}, nil
	case "DAY_HOUR":
		d, fsp, err := parseAndValidateDurationValue(format, HourIndex, DayHourMaxCnt)
		if err != nil {
			return ZeroDuration, err
		}
		return Duration{Duration: gotime.Duration(d), Fsp: fsp}, nil
	case "YEAR_MONTH":
		_, _, err := parseAndValidateDurationValue(format, MonthIndex, YearMonthMaxCnt)
		if err != nil {
			return ZeroDuration, err
		}
		// MONTH must exceed the limit of mysql's duration. So just returns overflow error.
		return ZeroDuration, ErrDatetimeFunctionOverflow.GenWithStackByArgs("time")
	default:
		return ZeroDuration, errors.Errorf("invalid single timeunit - %s", unit)
	}
}

// IsClockUnit returns true when unit is interval unit with hour, minute, second or microsecond.
func IsClockUnit(unit string) bool {
	switch strings.ToUpper(unit) {
	case "MICROSECOND", "SECOND", "MINUTE", "HOUR",
		"SECOND_MICROSECOND", "MINUTE_MICROSECOND", "HOUR_MICROSECOND", "DAY_MICROSECOND",
		"MINUTE_SECOND", "HOUR_SECOND", "DAY_SECOND",
		"HOUR_MINUTE", "DAY_MINUTE",
		"DAY_HOUR":
		return true
	default:
		return false
	}
}

// IsDateUnit returns true when unit is interval unit with year, quarter, month, week or day.
func IsDateUnit(unit string) bool {
	switch strings.ToUpper(unit) {
	case "DAY", "WEEK", "MONTH", "QUARTER", "YEAR",
		"DAY_MICROSECOND", "DAY_SECOND", "DAY_MINUTE", "DAY_HOUR",
		"YEAR_MONTH":
		return true
	default:
		return false
	}
}

// IsMicrosecondUnit returns true when unit is interval unit with microsecond.
func IsMicrosecondUnit(unit string) bool {
	switch strings.ToUpper(unit) {
	case "MICROSECOND", "SECOND_MICROSECOND", "MINUTE_MICROSECOND", "HOUR_MICROSECOND", "DAY_MICROSECOND":
		return true
	default:
		return false
	}
}

// IsDateFormat returns true when the specified time format could contain only date.
func IsDateFormat(format string) bool {
	format = strings.TrimSpace(format)
	seps := ParseDateFormat(format)
	length := len(format)
	switch len(seps) {
	case 1:
		// "20129" will be parsed to 2020-12-09, which is date format.
		if (length == 8) || (length == 6) || (length == 5) {
			return true
		}
	case 3:
		return true
	}
	return false
}

// ParseTimeFromInt64 parses mysql time value from int64.
func ParseTimeFromInt64(ctx Context, num int64) (Time, error) {
	return parseDateTimeFromNum(ctx, num)
}

// ParseTimeFromFloat64 parses mysql time value from float64.
// It is used in scenarios that distinguish date and datetime, e.g., date_add/sub() with first argument being real.
// For example, 20010203 parses to date (no HMS) and 20010203040506 parses to datetime (with HMS).
func ParseTimeFromFloat64(ctx Context, f float64) (Time, error) {
	intPart := int64(f)
	t, err := parseDateTimeFromNum(ctx, intPart)
	if err != nil {
		return ZeroTime, err
	}
	if t.Type() == mysql.TypeDatetime {
		// US part is only kept when the integral part is recognized as datetime.
		fracPart := uint32(math.Round((f - float64(intPart)) * 1000000.0))
		ct := t.CoreTime()
		ct.setMicrosecond(fracPart)
		t.SetCoreTime(ct)
	}
	return t, err
}

// ParseTimeFromDecimal parses mysql time value from decimal.
// It is used in scenarios that distinguish date and datetime, e.g., date_add/sub() with first argument being decimal.
// For example, 20010203 parses to date (no HMS) and 20010203040506 parses to datetime (with HMS).
func ParseTimeFromDecimal(ctx Context, dec *MyDecimal) (t Time, err error) {
	intPart, err := dec.ToInt()
	if err != nil && !ErrorEqual(err, ErrTruncated) {
		return ZeroTime, err
	}
	fsp := min(MaxFsp, int(dec.GetDigitsFrac()))
	t, err = parseDateTimeFromNum(ctx, intPart)
	if err != nil {
		return ZeroTime, err
	}
	t.SetFsp(fsp)
	if fsp == 0 || t.Type() == mysql.TypeDate {
		// Shortcut for integer value or date value (fractional part omitted).
		return t, err
	}

	intPartDec := new(MyDecimal).FromInt(intPart)
	fracPartDec := new(MyDecimal)
	err = DecimalSub(dec, intPartDec, fracPartDec)
	if err != nil {
		return ZeroTime, errors.Trace(NewStd(mysql.ErrIncorrectDatetimeValue).GenWithStackByArgs(dec.ToString()))
	}
	million := new(MyDecimal).FromInt(1000000)
	msPartDec := new(MyDecimal)
	err = DecimalMul(fracPartDec, million, msPartDec)
	if err != nil && !ErrorEqual(err, ErrTruncated) {
		return ZeroTime, errors.Trace(NewStd(mysql.ErrIncorrectDatetimeValue).GenWithStackByArgs(dec.ToString()))
	}
	msPart, err := msPartDec.ToInt()
	if err != nil && !ErrorEqual(err, ErrTruncated) {
		return ZeroTime, errors.Trace(NewStd(mysql.ErrIncorrectDatetimeValue).GenWithStackByArgs(dec.ToString()))
	}

	ct := t.CoreTime()
	ct.setMicrosecond(uint32(msPart))
	t.SetCoreTime(ct)

	return t, nil
}

// DateFormat returns a textual representation of the time value formatted
// according to layout.
// See http://dev.mysql.com/doc/refman/5.7/en/date-and-time-functions.html#function_date-format
func (t Time) DateFormat(layout string) (string, error) {
	var buf bytes.Buffer
	inPatternMatch := false
	for _, b := range layout {
		if inPatternMatch {
			if err := t.convertDateFormat(b, &buf); err != nil {
				return "", errors.Trace(err)
			}
			inPatternMatch = false
			continue
		}

		// It's not in pattern match now.
		if b == '%' {
			inPatternMatch = true
		} else {
			buf.WriteRune(b)
		}
	}
	return buf.String(), nil
}

var abbrevWeekdayName = []string{
	"Sun", "Mon", "Tue",
	"Wed", "Thu", "Fri", "Sat",
}

func (t Time) convertDateFormat(b rune, buf *bytes.Buffer) error {
	switch b {
	case 'b':
		m := t.Month()
		if m == 0 || m > 12 {
			return errors.Trace(ErrWrongValue.GenWithStackByArgs(TimeStr, strconv.Itoa(m)))
		}
		buf.WriteString(MonthNames[m-1][:3])
	case 'M':
		m := t.Month()
		if m == 0 || m > 12 {
			return errors.Trace(ErrWrongValue.GenWithStackByArgs(TimeStr, strconv.Itoa(m)))
		}
		buf.WriteString(MonthNames[m-1])
	case 'm':
		buf.WriteString(FormatIntWidthN(t.Month(), 2))
	case 'c':
		buf.WriteString(strconv.FormatInt(int64(t.Month()), 10))
	case 'D':
		buf.WriteString(strconv.FormatInt(int64(t.Day()), 10))
		buf.WriteString(abbrDayOfMonth(t.Day()))
	case 'd':
		buf.WriteString(FormatIntWidthN(t.Day(), 2))
	case 'e':
		buf.WriteString(strconv.FormatInt(int64(t.Day()), 10))
	case 'j':
		fmt.Fprintf(buf, "%03d", t.YearDay())
	case 'H':
		buf.WriteString(FormatIntWidthN(t.Hour(), 2))
	case 'k':
		buf.WriteString(strconv.FormatInt(int64(t.Hour()), 10))
	case 'h', 'I':
		tt := t.Hour()
		if tt%12 == 0 {
			buf.WriteString("12")
		} else {
			buf.WriteString(FormatIntWidthN(tt%12, 2))
		}
	case 'l':
		tt := t.Hour()
		if tt%12 == 0 {
			buf.WriteString("12")
		} else {
			buf.WriteString(strconv.FormatInt(int64(tt%12), 10))
		}
	case 'i':
		buf.WriteString(FormatIntWidthN(t.Minute(), 2))
	case 'p':
		hour := t.Hour()
		if hour/12%2 == 0 {
			buf.WriteString("AM")
		} else {
			buf.WriteString("PM")
		}
	case 'r':
		h := t.Hour()
		h %= 24
		switch {
		case h == 0:
			fmt.Fprintf(buf, "%02d:%02d:%02d AM", 12, t.Minute(), t.Second())
		case h == 12:
			fmt.Fprintf(buf, "%02d:%02d:%02d PM", 12, t.Minute(), t.Second())
		case h < 12:
			fmt.Fprintf(buf, "%02d:%02d:%02d AM", h, t.Minute(), t.Second())
		default:
			fmt.Fprintf(buf, "%02d:%02d:%02d PM", h-12, t.Minute(), t.Second())
		}
	case 'T':
		fmt.Fprintf(buf, "%02d:%02d:%02d", t.Hour(), t.Minute(), t.Second())
	case 'S', 's':
		buf.WriteString(FormatIntWidthN(t.Second(), 2))
	case 'f':
		fmt.Fprintf(buf, "%06d", t.Microsecond())
	case 'U':
		w := t.Week(0)
		buf.WriteString(FormatIntWidthN(w, 2))
	case 'u':
		w := t.Week(1)
		buf.WriteString(FormatIntWidthN(w, 2))
	case 'V':
		w := t.Week(2)
		buf.WriteString(FormatIntWidthN(w, 2))
	case 'v':
		_, w := t.YearWeek(3)
		buf.WriteString(FormatIntWidthN(w, 2))
	case 'a':
		weekday := t.Weekday()
		buf.WriteString(abbrevWeekdayName[weekday])
	case 'W':
		buf.WriteString(t.Weekday().String())
	case 'w':
		buf.WriteString(strconv.FormatInt(int64(t.Weekday()), 10))
	case 'X':
		year, _ := t.YearWeek(2)
		if year < 0 {
			buf.WriteString(strconv.FormatUint(uint64(math.MaxUint32), 10))
		} else {
			buf.WriteString(FormatIntWidthN(year, 4))
		}
	case 'x':
		year, _ := t.YearWeek(3)
		if year < 0 {
			buf.WriteString(strconv.FormatUint(uint64(math.MaxUint32), 10))
		} else {
			buf.WriteString(FormatIntWidthN(year, 4))
		}
	case 'Y':
		buf.WriteString(FormatIntWidthN(t.Year(), 4))
	case 'y':
		str := FormatIntWidthN(t.Year(), 4)
		buf.WriteString(str[2:])
	default:
		buf.WriteRune(b)
	}

	return nil
}

// FormatIntWidthN uses to format int with width. Insufficient digits are filled by 0.
func FormatIntWidthN(num, n int) string {
	numString := strconv.FormatInt(int64(num), 10)
	if len(numString) >= n {
		return numString
	}
	padBytes := make([]byte, n-len(numString))
	for i := range padBytes {
		padBytes[i] = '0'
	}
	return string(padBytes) + numString
}

func abbrDayOfMonth(day int) string {
	var str string
	switch day {
	case 1, 21, 31:
		str = "st"
	case 2, 22:
		str = "nd"
	case 3, 23:
		str = "rd"
	default:
		str = "th"
	}
	return str
}

// StrToDate converts date string according to format.
// See https://dev.mysql.com/doc/refman/5.7/en/date-and-time-functions.html#function_date-format
func (t *Time) StrToDate(typeCtx Context, date, format string) bool {
	ctx := make(map[string]int)
	var tm CoreTime
	success, warning := strToDate(&tm, date, format, ctx)
	if !success {
		t.SetCoreTime(ZeroCoreTime)
		t.SetType(mysql.TypeDatetime)
		t.SetFsp(0)
		return false
	}
	if err := mysqlTimeFix(&tm, ctx); err != nil {
		return false
	}

	t.SetCoreTime(tm)
	t.SetType(mysql.TypeDatetime)
	if t.Check(typeCtx) != nil {
		return false
	}
	if warning {
		// Only append this warning when success but still need warning.
		// Currently this only happens when `date` has extra characters at the end.
		typeCtx.AppendWarning(ErrTruncatedWrongVal.FastGenByArgs(DateTimeStr, date))
	}
	return true
}

// mysqlTimeFix fixes the Time use the values in the context.
func mysqlTimeFix(t *CoreTime, ctx map[string]int) error {
	// Key of the ctx is the format char, such as `%j` `%p` and so on.
	if yearOfDay, ok := ctx["%j"]; ok {
		// TODO: Implement the function that converts day of year to yy:mm:dd.
		_ = yearOfDay
	}
	if valueAMorPm, ok := ctx["%p"]; ok {
		if _, ok := ctx["%H"]; ok {
			return ErrWrongValue.GenWithStackByArgs(TimeStr, t)
		}
		if t.Hour() == 0 {
			return ErrWrongValue.GenWithStackByArgs(TimeStr, t)
		}
		if t.Hour() == 12 {
			// 12 is a special hour.
			switch valueAMorPm {
			case constForAM:
				t.setHour(0)
			case constForPM:
				t.setHour(12)
			}
			return nil
		}
		if valueAMorPm == constForPM {
			t.setHour(t.getHour() + 12)
		}
	} else {
		if _, ok := ctx["%h"]; ok && t.Hour() == 12 {
			t.setHour(0)
		}
	}
	return nil
}

// strToDate converts date string according to format,
// the value will be stored in argument t or ctx.
// The second return value is true when success but still need to append a warning.
func strToDate(t *CoreTime, date string, format string, ctx map[string]int) (success bool, warning bool) {
	date = skipWhiteSpace(date)
	format = skipWhiteSpace(format)

	token, formatRemain, succ := getFormatToken(format)
	if !succ {
		return false, false
	}

	if token == "" {
		if len(date) != 0 {
			// Extra characters at the end of date are ignored, but a warning should be reported at this case.
			return true, true
		}
		// Normal case. Both token and date are empty now.
		return true, false
	}

	if len(date) == 0 {
		ctx[token] = 0
		return true, false
	}

	dateRemain, succ := matchDateWithToken(t, date, token, ctx)
	if !succ {
		return false, false
	}

	return strToDate(t, dateRemain, formatRemain, ctx)
}

// getFormatToken takes one format control token from the string.
// format "%d %H %m" will get token "%d" and the remain is " %H %m".
func getFormatToken(format string) (token string, remain string, succ bool) {
	if len(format) == 0 {
		return "", "", true
	}

	// Just one character.
	if len(format) == 1 {
		if format[0] == '%' {
			return "", "", false
		}
		return format, "", true
	}

	// More than one character.
	if format[0] == '%' {
		return format[:2], format[2:], true
	}

	return format[:1], format[1:], true
}

func skipWhiteSpace(input string) string {
	for i, c := range input {
		if !unicode.IsSpace(c) {
			return input[i:]
		}
	}
	return ""
}

var monthAbbrev = map[string]gotime.Month{
	"jan": gotime.January,
	"feb": gotime.February,
	"mar": gotime.March,
	"apr": gotime.April,
	"may": gotime.May,
	"jun": gotime.June,
	"jul": gotime.July,
	"aug": gotime.August,
	"sep": gotime.September,
	"oct": gotime.October,
	"nov": gotime.November,
	"dec": gotime.December,
}

type dateFormatParser func(t *CoreTime, date string, ctx map[string]int) (remain string, succ bool)

var dateFormatParserTable = map[string]dateFormatParser{
	"%b": abbreviatedMonth,      // Abbreviated month name (Jan..Dec)
	"%c": monthNumeric,          // Month, numeric (0..12)
	"%d": dayOfMonthNumeric,     // Day of the month, numeric (0..31)
	"%e": dayOfMonthNumeric,     // Day of the month, numeric (0..31)
	"%f": microSeconds,          // Microseconds (000000..999999)
	"%h": hour12Numeric,         // Hour (01..12)
	"%H": hour24Numeric,         // Hour (00..23)
	"%I": hour12Numeric,         // Hour (01..12)
	"%i": minutesNumeric,        // Minutes, numeric (00..59)
	"%j": dayOfYearNumeric,      // Day of year (001..366)
	"%k": hour24Numeric,         // Hour (0..23)
	"%l": hour12Numeric,         // Hour (1..12)
	"%M": fullNameMonth,         // Month name (January..December)
	"%m": monthNumeric,          // Month, numeric (00..12)
	"%p": isAMOrPM,              // AM or PM
	"%r": time12Hour,            // Time, 12-hour (hh:mm:ss followed by AM or PM)
	"%s": secondsNumeric,        // Seconds (00..59)
	"%S": secondsNumeric,        // Seconds (00..59)
	"%T": time24Hour,            // Time, 24-hour (hh:mm:ss)
	"%Y": yearNumericFourDigits, // Year, numeric, four digits
	"%#": skipAllNums,           // Skip all numbers
	"%.": skipAllPunct,          // Skip all punctation characters
	"%@": skipAllAlpha,          // Skip all alpha characters
	// Deprecated since MySQL 5.7.5
	"%y": yearNumericTwoDigits, // Year, numeric (two digits)
	// TODO: Add the following...
	// "%a": abbreviatedWeekday,         // Abbreviated weekday name (Sun..Sat)
	// "%D": dayOfMonthWithSuffix,       // Day of the month with English suffix (0th, 1st, 2nd, 3rd)
	// "%U": weekMode0,                  // Week (00..53), where Sunday is the first day of the week; WEEK() mode 0
	// "%u": weekMode1,                  // Week (00..53), where Monday is the first day of the week; WEEK() mode 1
	// "%V": weekMode2,                  // Week (01..53), where Sunday is the first day of the week; WEEK() mode 2; used with %X
	// "%v": weekMode3,                  // Week (01..53), where Monday is the first day of the week; WEEK() mode 3; used with %x
	// "%W": weekdayName,                // Weekday name (Sunday..Saturday)
	// "%w": dayOfWeek,                  // Day of the week (0=Sunday..6=Saturday)
	// "%X": yearOfWeek,                 // Year for the week where Sunday is the first day of the week, numeric, four digits; used with %V
	// "%x": yearOfWeek,                 // Year for the week, where Monday is the first day of the week, numeric, four digits; used with %v
}

// GetFormatType checks the type(Duration, Date or Datetime) of a format string.
func GetFormatType(format string) (isDuration, isDate bool) {
	format = skipWhiteSpace(format)
	var token string
	var succ bool
	for {
		token, format, succ = getFormatToken(format)
		if len(token) == 0 {
			break
		}
		if !succ {
			isDuration, isDate = false, false
			break
		}
		if len(token) >= 2 && token[0] == '%' {
			switch token[1] {
			case 'h', 'H', 'i', 'I', 's', 'S', 'k', 'l', 'f', 'r', 'T':
				isDuration = true
			case 'y', 'Y', 'm', 'M', 'c', 'b', 'D', 'd', 'e':
				isDate = true
			}
		}
		if isDuration && isDate {
			break
		}
	}
	return
}

func matchDateWithToken(t *CoreTime, date string, token string, ctx map[string]int) (remain string, succ bool) {
	if parse, ok := dateFormatParserTable[token]; ok {
		return parse(t, date, ctx)
	}

	if strings.HasPrefix(date, token) {
		return date[len(token):], true
	}
	return date, false
}

// Try to parse digits with number of `limit` starting from `input`
// Return <number, n chars to step forward> if success.
// Return <_, 0> if fail.
func parseNDigits(input string, limit int) (number int, step int) {
	if limit <= 0 {
		return 0, 0
	}

	var num uint64 = 0
	step = 0
	for ; step < len(input) && step < limit && '0' <= input[step] && input[step] <= '9'; step++ {
		num = num*10 + uint64(input[step]-'0')
	}
	return int(num), step
}

func secondsNumeric(t *CoreTime, input string, _ map[string]int) (string, bool) {
	v, step := parseNDigits(input, 2)
	if step <= 0 || v >= 60 {
		return input, false
	}
	t.setSecond(uint8(v))
	return input[step:], true
}

func minutesNumeric(t *CoreTime, input string, _ map[string]int) (string, bool) {
	v, step := parseNDigits(input, 2)
	if step <= 0 || v >= 60 {
		return input, false
	}
	t.setMinute(uint8(v))
	return input[step:], true
}

type parseState int32

const (
	parseStateNormal    parseState = 1
	parseStateFail      parseState = 2
	parseStateEndOfLine parseState = 3
)

func parseSep(input string) (string, parseState) {
	input = skipWhiteSpace(input)
	if len(input) == 0 {
		return input, parseStateEndOfLine
	}
	if input[0] != ':' {
		return input, parseStateFail
	}
	if input = skipWhiteSpace(input[1:]); len(input) == 0 {
		return input, parseStateEndOfLine
	}
	return input, parseStateNormal
}

func time12Hour(t *CoreTime, input string, _ map[string]int) (string, bool) {
	tryParse := func(input string) (string, parseState) {
		// hh:mm:ss AM
		/// Note that we should update `t` as soon as possible, or we
		/// can not get correct result for incomplete input like "12:13"
		/// that is shorter than "hh:mm:ss"
		hour, step := parseNDigits(input, 2) // 1..12
		if step <= 0 || hour > 12 || hour == 0 {
			return input, parseStateFail
		}
		// Handle special case: 12:34:56 AM -> 00:34:56
		// For PM, we will add 12 it later
		if hour == 12 {
			hour = 0
		}
		t.setHour(uint8(hour))

		// ':'
		var state parseState
		if input, state = parseSep(input[step:]); state != parseStateNormal {
			return input, state
		}

		minute, step := parseNDigits(input, 2) // 0..59
		if step <= 0 || minute > 59 {
			return input, parseStateFail
		}
		t.setMinute(uint8(minute))

		// ':'
		if input, state = parseSep(input[step:]); state != parseStateNormal {
			return input, state
		}

		second, step := parseNDigits(input, 2) // 0..59
		if step <= 0 || second > 59 {
			return input, parseStateFail
		}
		t.setSecond(uint8(second))

		input = skipWhiteSpace(input[step:])
		if len(input) == 0 {
			// No "AM"/"PM" suffix, it is ok
			return input, parseStateEndOfLine
		} else if len(input) < 2 {
			// some broken char, fail
			return input, parseStateFail
		}

		switch {
		case hasCaseInsensitivePrefix(input, "AM"):
			t.setHour(uint8(hour))
		case hasCaseInsensitivePrefix(input, "PM"):
			t.setHour(uint8(hour + 12))
		default:
			return input, parseStateFail
		}

		return input[2:], parseStateNormal
	}

	remain, state := tryParse(input)
	if state == parseStateFail {
		return input, false
	}
	return remain, true
}

func time24Hour(t *CoreTime, input string, _ map[string]int) (string, bool) {
	tryParse := func(input string) (string, parseState) {
		// hh:mm:ss
		/// Note that we should update `t` as soon as possible, or we
		/// can not get correct result for incomplete input like "12:13"
		/// that is shorter than "hh:mm:ss"
		hour, step := parseNDigits(input, 2) // 0..23
		if step <= 0 || hour > 23 {
			return input, parseStateFail
		}
		t.setHour(uint8(hour))

		// ':'
		var state parseState
		if input, state = parseSep(input[step:]); state != parseStateNormal {
			return input, state
		}

		minute, step := parseNDigits(input, 2) // 0..59
		if step <= 0 || minute > 59 {
			return input, parseStateFail
		}
		t.setMinute(uint8(minute))

		// ':'
		if input, state = parseSep(input[step:]); state != parseStateNormal {
			return input, state
		}

		second, step := parseNDigits(input, 2) // 0..59
		if step <= 0 || second > 59 {
			return input, parseStateFail
		}
		t.setSecond(uint8(second))
		return input[step:], parseStateNormal
	}

	remain, state := tryParse(input)
	if state == parseStateFail {
		return input, false
	}
	return remain, true
}

const (
	constForAM = 1 + iota
	constForPM
)

func isAMOrPM(_ *CoreTime, input string, ctx map[string]int) (string, bool) {
	if len(input) < 2 {
		return input, false
	}

	s := strings.ToLower(input[:2])
	switch s {
	case "am":
		ctx["%p"] = constForAM
	case "pm":
		ctx["%p"] = constForPM
	default:
		return input, false
	}
	return input[2:], true
}

// oneToSixDigitRegex: it was just for [0, 999999]
var oneToSixDigitRegex = regexp.MustCompile("^[0-9]{0,6}")

// numericRegex: it was for any numeric characters
var numericRegex = regexp.MustCompile("[0-9]+")

func dayOfMonthNumeric(t *CoreTime, input string, _ map[string]int) (string, bool) {
	v, step := parseNDigits(input, 2) // 0..31
	if step <= 0 || v > 31 {
		return input, false
	}
	t.setDay(uint8(v))
	return input[step:], true
}

func hour24Numeric(t *CoreTime, input string, ctx map[string]int) (string, bool) {
	v, step := parseNDigits(input, 2) // 0..23
	if step <= 0 || v > 23 {
		return input, false
	}
	t.setHour(uint8(v))
	ctx["%H"] = v
	return input[step:], true
}

func hour12Numeric(t *CoreTime, input string, ctx map[string]int) (string, bool) {
	v, step := parseNDigits(input, 2) // 1..12
	if step <= 0 || v > 12 || v == 0 {
		return input, false
	}
	t.setHour(uint8(v))
	ctx["%h"] = v
	return input[step:], true
}

func microSeconds(t *CoreTime, input string, _ map[string]int) (string, bool) {
	v, step := parseNDigits(input, 6)
	if step <= 0 {
		t.setMicrosecond(0)
		return input, true
	}
	for i := step; i < 6; i++ {
		v *= 10
	}
	t.setMicrosecond(uint32(v))
	return input[step:], true
}

func yearNumericFourDigits(t *CoreTime, input string, ctx map[string]int) (string, bool) {
	return yearNumericNDigits(t, input, ctx, 4)
}

func yearNumericTwoDigits(t *CoreTime, input string, ctx map[string]int) (string, bool) {
	return yearNumericNDigits(t, input, ctx, 2)
}

func yearNumericNDigits(t *CoreTime, input string, _ map[string]int, n int) (string, bool) {
	year, step := parseNDigits(input, n)
	if step <= 0 {
		return input, false
	} else if step <= 2 {
		year = adjustYear(year)
	}
	t.setYear(uint16(year))
	return input[step:], true
}

func dayOfYearNumeric(_ *CoreTime, input string, ctx map[string]int) (string, bool) {
	// MySQL declares that "%j" should be "Day of year (001..366)". But actually,
	// it accepts a number that is up to three digits, which range is [1, 999].
	v, step := parseNDigits(input, 3)
	if step <= 0 || v == 0 {
		return input, false
	}
	ctx["%j"] = v
	return input[step:], true
}

func abbreviatedMonth(t *CoreTime, input string, _ map[string]int) (string, bool) {
	if len(input) >= 3 {
		monthName := strings.ToLower(input[:3])
		if month, ok := monthAbbrev[monthName]; ok {
			t.setMonth(uint8(month))
			return input[len(monthName):], true
		}
	}
	return input, false
}

func hasCaseInsensitivePrefix(input, prefix string) bool {
	if len(input) < len(prefix) {
		return false
	}
	return strings.EqualFold(input[:len(prefix)], prefix)
}

func fullNameMonth(t *CoreTime, input string, _ map[string]int) (string, bool) {
	for i, month := range MonthNames {
		if hasCaseInsensitivePrefix(input, month) {
			t.setMonth(uint8(i + 1))
			return input[len(month):], true
		}
	}
	return input, false
}

func monthNumeric(t *CoreTime, input string, _ map[string]int) (string, bool) {
	v, step := parseNDigits(input, 2) // 1..12
	if step <= 0 || v > 12 {
		return input, false
	}
	t.setMonth(uint8(v))
	return input[step:], true
}

// DateFSP gets fsp from date string.
func DateFSP(date string) (fsp int) {
	i := strings.LastIndex(date, ".")
	if i != -1 {
		fsp = len(date) - i - 1
	}
	return
}

// DateTimeIsOverflow returns if this date is overflow.
// See: https://dev.mysql.com/doc/refman/8.0/en/datetime.html
func DateTimeIsOverflow(ctx Context, date Time) (bool, error) {
	tz := ctx.Location()
	if tz == nil {
		BgLogger.Warn("use gotime.local because sc.timezone is nil")
		tz = gotime.Local
	}

	var err error
	var b, e, t gotime.Time
	switch date.Type() {
	case mysql.TypeDate, mysql.TypeDatetime:
		if b, err = MinDatetime.GoTime(tz); err != nil {
			return false, err
		}
		if e, err = MaxDatetime.GoTime(tz); err != nil {
			return false, err
		}
	case mysql.TypeTimestamp:
		minTS, maxTS := MinTimestamp, MaxTimestamp
		if tz != gotime.UTC {
			if err = minTS.ConvertTimeZone(gotime.UTC, tz); err != nil {
				return false, err
			}
			if err = maxTS.ConvertTimeZone(gotime.UTC, tz); err != nil {
				return false, err
			}
		}
		if b, err = minTS.GoTime(tz); err != nil {
			return false, err
		}
		if e, err = maxTS.GoTime(tz); err != nil {
			return false, err
		}
	default:
		return false, nil
	}

	if t, err = date.AdjustedGoTime(tz); err != nil {
		return false, err
	}

	inRange := (t.After(b) || t.Equal(b)) && (t.Before(e) || t.Equal(e))
	return !inRange, nil
}

func skipAllNums(_ *CoreTime, input string, _ map[string]int) (string, bool) {
	retIdx := 0
	for i, ch := range input {
		if !unicode.IsNumber(ch) {
			break
		}
		retIdx = i + 1
	}
	return input[retIdx:], true
}

func skipAllPunct(_ *CoreTime, input string, _ map[string]int) (string, bool) {
	retIdx := 0
	for i, ch := range input {
		if !unicode.IsPunct(ch) {
			break
		}
		retIdx = i + 1
	}
	return input[retIdx:], true
}

func skipAllAlpha(_ *CoreTime, input string, _ map[string]int) (string, bool) {
	retIdx := 0
	for i, ch := range input {
		if !unicode.IsLetter(ch) {
			break
		}
		retIdx = i + 1
	}
	return input[retIdx:], true
}
