// Copyright 2016 PingCAP, Inc.
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
	"github.com/abbychau/mysql-parser/parser_driver/mysql"
)

// const strings for ErrWrongValue
const (
	DateTimeStr  = "datetime"
	DateStr      = "date"
	TimeStr      = "time"
	TimestampStr = "timestamp"
)

var (
	// ErrInvalidDefault is returned when meet a invalid default value.
	ErrInvalidDefault = NewStd(mysql.ErrInvalidDefault)
	// ErrDataTooLong is returned when converts a string value that is longer than field type length.
	ErrDataTooLong = NewStd(mysql.ErrDataTooLong)
	// ErrIllegalValueForType is returned when value of type is illegal.
	ErrIllegalValueForType = NewStd(mysql.ErrIllegalValueForType)
	// ErrTruncated is returned when data has been truncated during conversion.
	ErrTruncated = NewStd(mysql.WarnDataTruncated)
	// ErrOverflow is returned when data is out of range for a field type.
	ErrOverflow = NewStd(mysql.ErrDataOutOfRange)
	// ErrDivByZero is return when do division by 0.
	ErrDivByZero = NewStd(mysql.ErrDivisionByZero)
	// ErrTooBigDisplayWidth is return when display width out of range for column.
	ErrTooBigDisplayWidth = NewStd(mysql.ErrTooBigDisplaywidth)
	// ErrTooBigFieldLength is return when column length too big for column.
	ErrTooBigFieldLength = NewStd(mysql.ErrTooBigFieldlength)
	// ErrTooBigSet is returned when too many strings for column.
	ErrTooBigSet = NewStd(mysql.ErrTooBigSet)
	// ErrTooBigScale is returned when type DECIMAL/NUMERIC scale is bigger than mysql.MaxDecimalScale.
	ErrTooBigScale = NewStd(mysql.ErrTooBigScale)
	// ErrTooBigPrecision is returned when type DECIMAL/NUMERIC or DATETIME precision is bigger than mysql.MaxDecimalWidth or types.MaxFsp
	ErrTooBigPrecision = NewStd(mysql.ErrTooBigPrecision)
	// ErrBadNumber is return when parsing an invalid binary decimal number.
	ErrBadNumber = NewStd(1367) // ER_INCORRECT_VALUE
	// ErrInvalidFieldSize is returned when the precision of a column is out of range.
	ErrInvalidFieldSize = NewStd(mysql.ErrInvalidFieldSize)
	// ErrMBiggerThanD is returned when precision less than the scale.
	ErrMBiggerThanD = NewStd(mysql.ErrMBiggerThanD)
	// ErrWarnDataOutOfRange is returned when the value in a numeric column that is outside the permissible range of the column data type.
	// See https://dev.mysql.com/doc/refman/5.5/en/out-of-range-and-overflow.html for details
	ErrWarnDataOutOfRange = NewStd(mysql.ErrWarnDataOutOfRange)
	// ErrDuplicatedValueInType is returned when enum column has duplicated value.
	ErrDuplicatedValueInType = NewStd(mysql.ErrDuplicatedValueInType)
	// ErrDatetimeFunctionOverflow is returned when the calculation in datetime function cause overflow.
	ErrDatetimeFunctionOverflow = NewStd(mysql.ErrDatetimeFunctionOverflow)
	// ErrCastAsSignedOverflow is returned when positive out-of-range integer, and convert to its negative complement.
	ErrCastAsSignedOverflow = NewStd(1690) // ER_DATA_OUT_OF_RANGE
	// ErrCastNegIntAsUnsigned is returned when a negative integer be casted to an unsigned int.
	ErrCastNegIntAsUnsigned = NewStd(1264) // ER_WARN_DATA_OUT_OF_RANGE
	// ErrInvalidYearFormat is returned when the input is not a valid year format.
	ErrInvalidYearFormat = NewStd(1367) // ER_INCORRECT_VALUE
	// ErrInvalidYear is returned when the input value is not a valid year.
	ErrInvalidYear = NewStd(1367) // ER_INCORRECT_VALUE
	// ErrTruncatedWrongVal is returned when data has been truncated during conversion.
	ErrTruncatedWrongVal = NewStd(mysql.ErrTruncatedWrongValue)
	// ErrInvalidWeekModeFormat is returned when the week mode is wrong.
	ErrInvalidWeekModeFormat = NewStd(1367) // ER_INCORRECT_VALUE
	// ErrWrongFieldSpec is returned when the column specifier incorrect.
	ErrWrongFieldSpec = NewStd(mysql.ErrWrongFieldSpec)
	// ErrSyntax is returned when the syntax is not allowed.
	ErrSyntax = NewStd(mysql.ErrParse)
	// ErrWrongValue is returned when the input value is in wrong format.
	ErrWrongValue = NewStd(mysql.ErrTruncatedWrongValue)
	// ErrWrongValue2 is returned when the input value is in wrong format.
	ErrWrongValue2 = NewStd(mysql.ErrWrongValue)
	// ErrWrongValueForType is returned when the input value is in wrong format for function.
	ErrWrongValueForType = NewStd(mysql.ErrWrongValueForType)
	// ErrPartitionStatsMissing is returned when the partition-level stats is missing and the build global-level stats fails.
	// Put this error here is to prevent `import cycle not allowed`.
	ErrPartitionStatsMissing = NewStd(1105) // ER_UNKNOWN_ERROR
	// ErrPartitionColumnStatsMissing is returned when the partition-level column stats is missing and the build global-level stats fails.
	// Put this error here is to prevent `import cycle not allowed`.
	ErrPartitionColumnStatsMissing = NewStd(1105) // ER_UNKNOWN_ERROR
	// ErrInvalidJSONCharset is returned when the JSON charset is invalid.
	ErrInvalidJSONCharset = NewStd(1273) // ER_UNKNOWN_CHARACTER_SET
	// ErrIncorrectDatetimeValue is returned when the input value is in wrong format for datetime.
	ErrIncorrectDatetimeValue = NewStd(1292) // ER_INCORRECT_DATETIME_VALUE
	// ErrJSONBadOneOrAllArg is returned when the one_or_all argument isn't 'one' or 'all'.
	ErrJSONBadOneOrAllArg = NewStd(mysql.ErrJSONBadOneOrAllArg)
	// ErrJSONVacuousPath is returned for path expressions that are not allowed in that context.
	ErrJSONVacuousPath = NewStd(mysql.ErrJSONVacuousPath)
	// ErrTimestampInDSTTransition is returned if the converted timestamp is in the Daylight Saving Time
	// transition when time leaps forward (normally skips one hour).
	ErrTimestampInDSTTransition = NewStd(1105) // ER_UNKNOWN_ERROR
)
