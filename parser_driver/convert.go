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

// Copyright 2014 The ql Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSES/QL-LICENSE file.

package parser_driver

import (
	"math"
	"math/big"
	"strconv"
	"strings"

	"github.com/pingcap/errors"
	"github.com/abbychau/mysql-parser/parser_driver/mysql"
)

func truncateStr(str string, flen int) string {
	if flen != UnspecifiedLength && len(str) > flen {
		str = str[:flen]
	}
	return str
}

// IntegerUnsignedUpperBound indicates the max uint64 values of different mysql types.
func IntegerUnsignedUpperBound(intType byte) uint64 {
	switch intType {
	case mysql.TypeTiny:
		return math.MaxUint8
	case mysql.TypeShort:
		return math.MaxUint16
	case mysql.TypeInt24:
		return mysql.MaxUint24
	case mysql.TypeLong:
		return math.MaxUint32
	case mysql.TypeLonglong:
		return math.MaxUint64
	case mysql.TypeBit:
		return math.MaxUint64
	case mysql.TypeEnum:
		// enum can have at most 65535 distinct elements
		// it would be better to use len(FieldType.GetElems()), but we only have a byte type here
		return 65535
	case mysql.TypeSet:
		return math.MaxUint64
	default:
		panic("Input byte is not a mysql type")
	}
}

// IntegerSignedUpperBound indicates the max int64 values of different mysql types.
func IntegerSignedUpperBound(intType byte) int64 {
	switch intType {
	case mysql.TypeTiny:
		return math.MaxInt8
	case mysql.TypeShort:
		return math.MaxInt16
	case mysql.TypeInt24:
		return mysql.MaxInt24
	case mysql.TypeLong:
		return math.MaxInt32
	case mysql.TypeLonglong:
		return math.MaxInt64
	case mysql.TypeEnum:
		// enum can have at most 65535 distinct elements
		// it would be better to use len(FieldType.GetElems()), but we only have a byte type here
		return 65535
	default:
		panic("Input byte is not a mysql int type")
	}
}

// IntegerSignedLowerBound indicates the min int64 values of different mysql types.
func IntegerSignedLowerBound(intType byte) int64 {
	switch intType {
	case mysql.TypeTiny:
		return math.MinInt8
	case mysql.TypeShort:
		return math.MinInt16
	case mysql.TypeInt24:
		return mysql.MinInt24
	case mysql.TypeLong:
		return math.MinInt32
	case mysql.TypeLonglong:
		return math.MinInt64
	case mysql.TypeEnum:
		return 0
	default:
		panic("Input byte is not a mysql type")
	}
}

// ConvertFloatToInt converts a float64 value to a int value.
// `tp` is used in err msg, if there is overflow, this func will report err according to `tp`
func ConvertFloatToInt(fval float64, lowerBound, upperBound int64, tp byte) (int64, error) {
	val := RoundFloat(fval)
	if val < float64(lowerBound) {
		return lowerBound, overflow(val, tp)
	}

	if val >= float64(upperBound) {
		if val == float64(upperBound) {
			return upperBound, nil
		}
		return upperBound, overflow(val, tp)
	}
	return int64(val), nil
}

// ConvertIntToInt converts an int value to another int value of different precision.
func ConvertIntToInt(val int64, lowerBound int64, upperBound int64, tp byte) (int64, error) {
	if val < lowerBound {
		return lowerBound, overflow(val, tp)
	}

	if val > upperBound {
		return upperBound, overflow(val, tp)
	}

	return val, nil
}

// ConvertUintToInt converts an uint value to an int value.
func ConvertUintToInt(val uint64, upperBound int64, tp byte) (int64, error) {
	if val > uint64(upperBound) {
		return upperBound, overflow(val, tp)
	}

	return int64(val), nil
}

// ConvertIntToUint converts an int value to an uint value.
func ConvertIntToUint(flags ContextFlags, val int64, upperBound uint64, tp byte) (uint64, error) {
	if val < 0 && !flags.AllowNegativeToUnsigned() {
		return 0, overflow(val, tp)
	}

	if uint64(val) > upperBound {
		return upperBound, overflow(val, tp)
	}

	return uint64(val), nil
}

// ConvertUintToUint converts an uint value to another uint value of different precision.
func ConvertUintToUint(val uint64, upperBound uint64, tp byte) (uint64, error) {
	if val > upperBound {
		return upperBound, overflow(val, tp)
	}

	return val, nil
}

// ConvertFloatToUint converts a float value to an uint value.
func ConvertFloatToUint(flags ContextFlags, fval float64, upperBound uint64, tp byte) (uint64, error) {
	val := RoundFloat(fval)
	if val < 0 {
		if !flags.AllowNegativeToUnsigned() {
			return 0, overflow(val, tp)
		}
		return uint64(int64(val)), overflow(val, tp)
	}

	ret, acc := new(big.Float).SetFloat64(val).Uint64()
	if acc == big.Below || ret > upperBound {
		return upperBound, overflow(val, tp)
	}
	return ret, nil
}

// convertScientificNotation converts a decimal string with scientific notation to a normal decimal string.
// 1E6 => 1000000, .12345E+5 => 12345
func convertScientificNotation(str string) (string, error) {
	// https://golang.org/ref/spec#Floating-point_literals
	eIdx := -1
	point := -1
	for i := range len(str) {
		if str[i] == '.' {
			point = i
		}
		if str[i] == 'e' || str[i] == 'E' {
			eIdx = i
			if point == -1 {
				point = i
			}
			break
		}
	}
	if eIdx == -1 {
		return str, nil
	}
	exp, err := strconv.ParseInt(str[eIdx+1:], 10, 64)
	if err != nil {
		return "", errors.WithStack(err)
	}

	f := str[:eIdx]
	if exp == 0 {
		return f, nil
	} else if exp > 0 { // move point right
		if point+int(exp) == len(f)-1 { // 123.456 >> 3 = 123456. = 123456
			return f[:point] + f[point+1:], nil
		} else if point+int(exp) < len(f)-1 { // 123.456 >> 2 = 12345.6
			return f[:point] + f[point+1:point+1+int(exp)] + "." + f[point+1+int(exp):], nil
		}
		// 123.456 >> 5 = 12345600
		return f[:point] + f[point+1:] + strings.Repeat("0", point+int(exp)-len(f)+1), nil
	}
	// move point left
	exp = -exp
	if int(exp) < point { // 123.456 << 2 = 1.23456
		return f[:point-int(exp)] + "." + f[point-int(exp):point] + f[point+1:], nil
	}
	// 123.456 << 5 = 0.00123456
	return "0." + strings.Repeat("0", int(exp)-point) + f[:point] + f[point+1:], nil
}

func convertDecimalStrToUint(str string, upperBound uint64, tp byte) (uint64, error) {
	str, err := convertScientificNotation(str)
	if err != nil {
		return 0, err
	}

	var intStr, fracStr string
	p := strings.Index(str, ".")
	if p == -1 {
		intStr = str
	} else {
		intStr = str[:p]
		fracStr = str[p+1:]
	}
	intStr = strings.TrimLeft(intStr, "0")
	if intStr == "" {
		intStr = "0"
	}
	if intStr[0] == '-' {
		return 0, overflow(str, tp)
	}

	var round uint64
	if fracStr != "" && fracStr[0] >= '5' {
		round++
	}

	upperStr := strconv.FormatUint(upperBound-round, 10)
	if len(intStr) > len(upperStr) ||
		(len(intStr) == len(upperStr) && intStr > upperStr) {
		return upperBound, overflow(str, tp)
	}

	val, err := strconv.ParseUint(intStr, 10, 64)
	if err != nil {
		return val, overflow(str, tp)
	}
	return val + round, nil
}

// ConvertDecimalToUint converts a decimal to a uint by converting it to a string first to avoid float overflow (#10181).
func ConvertDecimalToUint(d *MyDecimal, upperBound uint64, tp byte) (uint64, error) {
	return convertDecimalStrToUint(string(d.ToString()), upperBound, tp)
}

// StrToInt converts a string to an integer at the best-effort.
func StrToInt(ctx Context, str string, isFuncCast bool) (int64, error) {
	str = strings.TrimSpace(str)
	validPrefix, err := getValidIntPrefix(ctx, str, isFuncCast)
	iVal, err1 := strconv.ParseInt(validPrefix, 10, 64)
	if err1 != nil {
		return iVal, ErrOverflow.GenWithStackByArgs("BIGINT", validPrefix)
	}
	return iVal, errors.Trace(err)
}

// StrToUint converts a string to an unsigned integer at the best-effort.
func StrToUint(ctx Context, str string, isFuncCast bool) (uint64, error) {
	str = strings.TrimSpace(str)
	validPrefix, err := getValidIntPrefix(ctx, str, isFuncCast)
	uVal := uint64(0)
	hasParseErr := false

	if validPrefix[0] == '-' {
		// only `-000*` is valid to be converted into unsigned integer
		for _, v := range validPrefix[1:] {
			if v != '0' {
				hasParseErr = true
				break
			}
		}
	} else {
		if validPrefix[0] == '+' {
			validPrefix = validPrefix[1:]
		}
		v, e := strconv.ParseUint(validPrefix, 10, 64)
		uVal, hasParseErr = v, e != nil
	}

	if hasParseErr {
		return uVal, ErrOverflow.GenWithStackByArgs("BIGINT UNSIGNED", validPrefix)
	}
	return uVal, errors.Trace(err)
}

// StrToDateTime converts str to MySQL DateTime.
func StrToDateTime(ctx Context, str string, fsp int) (Time, error) {
	return ParseTime(ctx, str, mysql.TypeDatetime, fsp)
}

// StrToDuration converts str to Duration. It returns Duration in normal case,
// and returns Time when str is in datetime format.
// when isDuration is true, the d is returned, when it is false, the t is returned.
// See https://dev.mysql.com/doc/refman/5.5/en/date-and-time-literals.html.
func StrToDuration(ctx Context, str string, fsp int) (d Duration, t Time, isDuration bool, err error) {
	str = strings.TrimSpace(str)
	length := len(str)
	if length > 0 && str[0] == '-' {
		length--
	}
	if n := strings.IndexByte(str, '.'); n >= 0 {
		length = length - len(str[n:])
	}
	// Timestamp format is 'YYYYMMDDHHMMSS' or 'YYMMDDHHMMSS', which length is 12.
	// See #3923, it explains what we do here.
	if length >= 12 {
		t, err = StrToDateTime(ctx, str, fsp)
		if err == nil {
			return d, t, false, nil
		}
	}

	d, _, err = ParseDuration(ctx, str, fsp)
	if ErrTruncatedWrongVal.Equal(err) {
		err = ctx.HandleTruncate(err)
	}
	return d, t, true, errors.Trace(err)
}

// NumberToDuration converts number to Duration.
func NumberToDuration(number int64, fsp int) (Duration, error) {
	if number > TimeMaxValue {
		// Try to parse DATETIME.
		if number >= 10000000000 { // '2001-00-00 00-00-00'
			if t, err := ParseDatetimeFromNum(DefaultStmtNoWarningContext, number); err == nil {
				dur, err1 := t.ConvertToDuration()
				return dur, errors.Trace(err1)
			}
		}
		dur := MaxMySQLDuration(fsp)
		return dur, ErrOverflow.GenWithStackByArgs("Duration", strconv.Itoa(int(number)))
	} else if number < -TimeMaxValue {
		dur := MaxMySQLDuration(fsp)
		dur.Duration = -dur.Duration
		return dur, ErrOverflow.GenWithStackByArgs("Duration", strconv.Itoa(int(number)))
	}
	var neg bool
	if neg = number < 0; neg {
		number = -number
	}

	if number/10000 > TimeMaxHour || number%100 >= 60 || (number/100)%100 >= 60 {
		return ZeroDuration, errors.Trace(ErrTruncatedWrongVal.GenWithStackByArgs(TimeStr, strconv.FormatInt(number, 10)))
	}
	dur := NewDuration(int(number/10000), int((number/100)%100), int(number%100), 0, fsp)
	if neg {
		dur.Duration = -dur.Duration
	}
	return dur, nil
}

// getValidIntPrefix gets prefix of the string which can be successfully parsed as int.
func getValidIntPrefix(ctx Context, str string, isFuncCast bool) (string, error) {
	if !isFuncCast {
		floatPrefix, err := getValidFloatPrefix(ctx, str, isFuncCast)
		if err != nil {
			return floatPrefix, errors.Trace(err)
		}
		return floatStrToIntStr(floatPrefix, str)
	}

	validLen := 0

	for i := range len(str) {
		c := str[i]
		if (c == '+' || c == '-') && i == 0 {
			continue
		}

		if c >= '0' && c <= '9' {
			validLen = i + 1
			continue
		}

		break
	}
	valid := str[:validLen]
	if valid == "" {
		valid = "0"
	}
	if validLen == 0 || validLen != len(str) {
		return valid, errors.Trace(ctx.HandleTruncate(ErrTruncatedWrongVal.GenWithStackByArgs("INTEGER", str)))
	}
	return valid, nil
}

// roundIntStr is to round a **valid int string** base on the number following dot.
func roundIntStr(numNextDot byte, intStr string) string {
	if numNextDot < '5' {
		return intStr
	}
	retStr := []byte(intStr)
	idx := len(intStr) - 1
	for ; idx >= 1; idx-- {
		if retStr[idx] != '9' {
			retStr[idx]++
			break
		}
		retStr[idx] = '0'
	}
	if idx == 0 {
		if intStr[0] == '9' {
			retStr[0] = '1'
			retStr = append(retStr, '0')
		} else if isDigit(intStr[0]) {
			retStr[0]++
		} else {
			retStr[1] = '1'
			retStr = append(retStr, '0')
		}
	}
	return string(retStr)
}

var maxUintStr = strconv.FormatUint(math.MaxUint64, 10)
var minIntStr = strconv.FormatInt(math.MinInt64, 10)

// floatStrToIntStr converts a valid float string into valid integer string which can be parsed by
// strconv.ParseInt, we can't parse float first then convert it to string because precision will
// be lost. For example, the string value "18446744073709551615" which is the max number of unsigned
// int will cause some precision to lose. intStr[0] may be a positive and negative sign like '+' or '-'.
//
// This func will find serious overflow such as the len of intStr > 20 (without prefix `+/-`)
// however, it will not check whether the intStr overflow BIGINT.
func floatStrToIntStr(validFloat string, oriStr string) (intStr string, _ error) {
	var dotIdx = -1
	var eIdx = -1
	for i := range len(validFloat) {
		switch validFloat[i] {
		case '.':
			dotIdx = i
		case 'e', 'E':
			eIdx = i
		}
	}
	if eIdx == -1 {
		if dotIdx == -1 {
			return validFloat, nil
		}
		var digits []byte
		if validFloat[0] == '-' || validFloat[0] == '+' {
			dotIdx--
			digits = []byte(validFloat[1:])
		} else {
			digits = []byte(validFloat)
		}
		if dotIdx == 0 {
			intStr = "0"
		} else {
			intStr = string(digits)[:dotIdx]
		}
		if len(digits) > dotIdx+1 {
			intStr = roundIntStr(digits[dotIdx+1], intStr)
		}
		if (len(intStr) > 1 || intStr[0] != '0') && validFloat[0] == '-' {
			intStr = "-" + intStr
		}
		return intStr, nil
	}
	// intCnt and digits contain the prefix `+/-` if validFloat[0] is `+/-`
	var intCnt int
	digits := make([]byte, 0, len(validFloat))
	if dotIdx == -1 {
		digits = append(digits, validFloat[:eIdx]...)
		intCnt = len(digits)
	} else {
		digits = append(digits, validFloat[:dotIdx]...)
		intCnt = len(digits)
		digits = append(digits, validFloat[dotIdx+1:eIdx]...)
	}
	exp, err := strconv.Atoi(validFloat[eIdx+1:])
	if err != nil {
		if digits[0] == '-' {
			intStr = minIntStr
		} else {
			intStr = maxUintStr
		}
		return intStr, ErrOverflow.GenWithStackByArgs("BIGINT", oriStr)
	}
	intCnt += exp
	if exp >= 0 && (intCnt > 21 || intCnt < 0) {
		// MaxInt64 has 19 decimal digits.
		// MaxUint64 has 20 decimal digits.
		// And the intCnt may contain the len of `+/-`,
		// so I use 21 here as the early detection.
		if digits[0] == '-' {
			intStr = minIntStr
		} else {
			intStr = maxUintStr
		}
		return intStr, ErrOverflow.GenWithStackByArgs("BIGINT", oriStr)
	}
	if intCnt <= 0 {
		intStr = "0"
		if intCnt == 0 && len(digits) > 0 && isDigit(digits[0]) {
			intStr = roundIntStr(digits[0], intStr)
		}
		return intStr, nil
	}
	if intCnt == 1 && (digits[0] == '-' || digits[0] == '+') {
		intStr = "0"
		if len(digits) > 1 {
			intStr = roundIntStr(digits[1], intStr)
		}
		if intStr[0] == '1' {
			intStr = string(digits[:1]) + intStr
		}
		return intStr, nil
	}
	if intCnt <= len(digits) {
		intStr = string(digits[:intCnt])
		if intCnt < len(digits) {
			intStr = roundIntStr(digits[intCnt], intStr)
		}
	} else {
		// convert scientific notation decimal number
		extraZeroCount := intCnt - len(digits)
		intStr = string(digits) + strings.Repeat("0", extraZeroCount)
	}
	return intStr, nil
}

// StrToFloat converts a string to a float64 at the best-effort.
func StrToFloat(ctx Context, str string, isFuncCast bool) (float64, error) {
	str = strings.TrimSpace(str)
	validStr, err := getValidFloatPrefix(ctx, str, isFuncCast)
	f, err1 := strconv.ParseFloat(validStr, 64)
	if err1 != nil {
		if err2, ok := err1.(*strconv.NumError); ok {
			// value will truncate to MAX/MIN if out of range.
			if err2.Err == strconv.ErrRange {
				err1 = ctx.HandleTruncate(ErrTruncatedWrongVal.GenWithStackByArgs("DOUBLE", str))
				if math.IsInf(f, 1) {
					f = math.MaxFloat64
				} else if math.IsInf(f, -1) {
					f = -math.MaxFloat64
				}
			}
		}
		return f, errors.Trace(err1)
	}
	return f, errors.Trace(err)
}

// ConvertJSONToInt64 casts JSON into int64.
func ConvertJSONToInt64(ctx Context, j BinaryJSON, unsigned bool) (int64, error) {
	return ConvertJSONToInt(ctx, j, unsigned, mysql.TypeLonglong)
}

// ConvertJSONToInt casts JSON into int by type.
func ConvertJSONToInt(ctx Context, j BinaryJSON, unsigned bool, tp byte) (int64, error) {
	switch j.TypeCode {
	case JSONTypeCodeObject, JSONTypeCodeArray, JSONTypeCodeOpaque, JSONTypeCodeDate, JSONTypeCodeDatetime, JSONTypeCodeTimestamp, JSONTypeCodeDuration:
		return 0, ctx.HandleTruncate(ErrTruncatedWrongVal.GenWithStackByArgs("INTEGER", j.String()))
	case JSONTypeCodeLiteral:
		switch j.Value[0] {
		case JSONLiteralFalse:
			return 0, nil
		case JSONLiteralNil:
			return 0, ctx.HandleTruncate(ErrTruncatedWrongVal.GenWithStackByArgs("INTEGER", j.String()))
		default:
			return 1, nil
		}
	case JSONTypeCodeInt64:
		i := j.GetInt64()
		if unsigned {
			uBound := IntegerUnsignedUpperBound(tp)
			u, err := ConvertIntToUint(ctx.Flags(), i, uBound, tp)
			return int64(u), err
		}

		lBound := IntegerSignedLowerBound(tp)
		uBound := IntegerSignedUpperBound(tp)
		return ConvertIntToInt(i, lBound, uBound, tp)
	case JSONTypeCodeUint64:
		u := j.GetUint64()
		if unsigned {
			uBound := IntegerUnsignedUpperBound(tp)
			u, err := ConvertUintToUint(u, uBound, tp)
			return int64(u), err
		}

		uBound := IntegerSignedUpperBound(tp)
		return ConvertUintToInt(u, uBound, tp)
	case JSONTypeCodeFloat64:
		f := j.GetFloat64()
		if !unsigned {
			lBound := IntegerSignedLowerBound(tp)
			uBound := IntegerSignedUpperBound(tp)
			u, e := ConvertFloatToInt(f, lBound, uBound, tp)
			return u, e
		}
		bound := IntegerUnsignedUpperBound(tp)
		u, err := ConvertFloatToUint(ctx.Flags(), f, bound, tp)
		return int64(u), err
	case JSONTypeCodeString:
		str := j.GetString()
		// The behavior of casting json string as an integer is consistent with casting a string as an integer.
		// See the `builtinCastStringAsIntSig` in `expression` pkg. The only difference is that this function
		// doesn't append any warning. This behavior is compatible with MySQL.
		isNegative := len(str) > 1 && str[0] == '-'
		if !isNegative {
			r, err := StrToUint(ctx, str, false)
			return int64(r), err
		}

		return StrToInt(ctx, str, false)
	}
	return 0, errors.New("Unknown type code in JSON")
}

// ConvertJSONToFloat casts JSON into float64.
func ConvertJSONToFloat(ctx Context, j BinaryJSON) (float64, error) {
	switch j.TypeCode {
	case JSONTypeCodeObject, JSONTypeCodeArray, JSONTypeCodeOpaque, JSONTypeCodeDate, JSONTypeCodeDatetime, JSONTypeCodeTimestamp, JSONTypeCodeDuration:
		return 0, ctx.HandleTruncate(ErrTruncatedWrongVal.GenWithStackByArgs("FLOAT", j.String()))
	case JSONTypeCodeLiteral:
		switch j.Value[0] {
		case JSONLiteralFalse:
			return 0, nil
		case JSONLiteralNil:
			return 0, ctx.HandleTruncate(ErrTruncatedWrongVal.GenWithStackByArgs("FLOAT", j.String()))
		default:
			return 1, nil
		}
	case JSONTypeCodeInt64:
		return float64(j.GetInt64()), nil
	case JSONTypeCodeUint64:
		return float64(j.GetUint64()), nil
	case JSONTypeCodeFloat64:
		return j.GetFloat64(), nil
	case JSONTypeCodeString:
		str := j.GetString()
		return StrToFloat(ctx, str, false)
	}
	return 0, errors.New("Unknown type code in JSON")
}

// ConvertJSONToDecimal casts JSON into decimal.
func ConvertJSONToDecimal(ctx Context, j BinaryJSON) (*MyDecimal, error) {
	var err error = nil
	res := new(MyDecimal)
	switch j.TypeCode {
	case JSONTypeCodeObject, JSONTypeCodeArray, JSONTypeCodeOpaque, JSONTypeCodeDate, JSONTypeCodeDatetime, JSONTypeCodeTimestamp, JSONTypeCodeDuration:
		err = ErrTruncatedWrongVal.GenWithStackByArgs("DECIMAL", j.String())
	case JSONTypeCodeLiteral:
		switch j.Value[0] {
		case JSONLiteralFalse:
			res = res.FromInt(0)
		case JSONLiteralNil:
			err = ErrTruncatedWrongVal.GenWithStackByArgs("DECIMAL", j.String())
		default:
			res = res.FromInt(1)
		}
	case JSONTypeCodeInt64:
		res = res.FromInt(j.GetInt64())
	case JSONTypeCodeUint64:
		res = res.FromUint(j.GetUint64())
	case JSONTypeCodeFloat64:
		err = res.FromFloat64(j.GetFloat64())
	case JSONTypeCodeString:
		err = res.FromString([]byte(j.GetString()))
	}
	err = ctx.HandleTruncate(err)
	if err != nil {
		return res, errors.Trace(err)
	}
	return res, errors.Trace(err)
}

// getValidFloatPrefix gets prefix of string which can be successfully parsed as float.
func getValidFloatPrefix(ctx Context, s string, isFuncCast bool) (valid string, err error) {
	if isFuncCast && s == "" {
		return "0", nil
	}

	var (
		sawDot   bool
		sawDigit bool
		validLen int
		eIdx     = -1
	)
	for i := range len(s) {
		c := s[i]
		if c == '+' || c == '-' {
			if i != 0 && i != eIdx+1 { // "1e+1" is valid.
				break
			}
		} else if c == '.' {
			if sawDot || eIdx > 0 { // "1.1." or "1e1.1"
				break
			}
			sawDot = true
			if sawDigit { // "123." is valid.
				validLen = i + 1
			}
		} else if c == 'e' || c == 'E' {
			if !sawDigit { // "+.e"
				break
			}
			if eIdx != -1 { // "1e5e"
				break
			}
			eIdx = i
		} else if c == '\u0000' {
			s = s[:validLen]
			break
		} else if c < '0' || c > '9' {
			break
		} else {
			sawDigit = true
			validLen = i + 1
		}
	}
	valid = s[:validLen]
	if valid == "" {
		valid = "0"
	}
	if validLen == 0 || validLen != len(s) {
		err = errors.Trace(ctx.HandleTruncate(ErrTruncatedWrongVal.GenWithStackByArgs("DOUBLE", s)))
	}
	return valid, err
}

// ToString converts an interface to a string.
func ToString(value any) (string, error) {
	switch v := value.(type) {
	case bool:
		if v {
			return "1", nil
		}
		return "0", nil
	case int:
		return strconv.FormatInt(int64(v), 10), nil
	case int64:
		return strconv.FormatInt(v, 10), nil
	case uint64:
		return strconv.FormatUint(v, 10), nil
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32), nil
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), nil
	case string:
		return v, nil
	case []byte:
		return string(v), nil
	case Time:
		return v.String(), nil
	case Duration:
		return v.String(), nil
	case *MyDecimal:
		return v.String(), nil
	case BinaryLiteral:
		return v.ToString(), nil
	case Enum:
		return v.String(), nil
	case Set:
		return v.String(), nil
	case BinaryJSON:
		return v.String(), nil
	default:
		return "", errors.Errorf("cannot convert %v(type %T) to string", value, value)
	}
}
