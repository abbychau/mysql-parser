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
	"io"

	"github.com/pingcap/errors"
	"github.com/abbychau/mysql-parser/parser_driver/mysql"
)

// Flags is a placeholder for context flags
type Flags uint64

// BinaryJSON represents a binary JSON value
type BinaryJSON struct {
	TypeCode int
	Value    []byte
}

// JSON type codes
const (
	JSONTypeCodeObject    = iota
	JSONTypeCodeArray
	JSONTypeCodeOpaque
	JSONTypeCodeDate
	JSONTypeCodeDatetime
	JSONTypeCodeTimestamp
	JSONTypeCodeDuration
	JSONTypeCodeLiteral
	JSONTypeCodeInt64
	JSONTypeCodeUint64
	JSONTypeCodeFloat64
	JSONTypeCodeString
)

// JSON literal values
const (
	JSONLiteralFalse = iota
	JSONLiteralTrue
	JSONLiteralNil
)

// String returns a string representation of the JSON value
func (bj BinaryJSON) String() string {
	// Stub implementation
	return "json_value"
}

// GetInt64 returns int64 value from JSON
func (bj BinaryJSON) GetInt64() int64 {
	// Stub implementation
	return 0
}

// GetUint64 returns uint64 value from JSON
func (bj BinaryJSON) GetUint64() uint64 {
	// Stub implementation
	return 0
}

// GetFloat64 returns float64 value from JSON
func (bj BinaryJSON) GetFloat64() float64 {
	// Stub implementation
	return 0.0
}

// GetString returns string value from JSON
func (bj BinaryJSON) GetString() string {
	// Stub implementation
	return ""
}

// Unquote returns unquoted string value from JSON
func (bj BinaryJSON) Unquote() (string, error) {
	// Stub implementation
	return "", nil
}

// VectorFloat32 represents a vector of float32 values
type VectorFloat32 []float32

// String returns string representation of the vector
func (v VectorFloat32) String() string {
	// Stub implementation
	return "[vector]"
}

// TruncatedString returns truncated string representation of the vector
func (v VectorFloat32) TruncatedString() string {
	// Stub implementation  
	return "[vector]"
}

// Compare compares two vectors
func (v VectorFloat32) Compare(other VectorFloat32) int {
	// Stub implementation - always return 0 (equal)
	return 0
}

// ZeroCopySerialize serializes the vector without copying
func (v VectorFloat32) ZeroCopySerialize() []byte {
	// Stub implementation
	return []byte{}
}

// IsZeroValue checks if vector is zero value
func (v VectorFloat32) IsZeroValue() bool {
	// Stub implementation
	return len(v) == 0
}

// CheckDimsFitColumn checks if vector dimensions fit column constraints
func (v VectorFloat32) CheckDimsFitColumn(colLen int) error {
	// Stub implementation
	return nil
}

// EstimatedMemUsage returns estimated memory usage of the vector
func (v VectorFloat32) EstimatedMemUsage() int {
	// Stub implementation - 4 bytes per float32
	return len(v) * 4
}

// ZeroCopyDeserializeVectorFloat32 deserializes vector data
func ZeroCopyDeserializeVectorFloat32(data []byte) (VectorFloat32, []byte, error) {
	// Stub implementation
	return VectorFloat32{}, data, nil
}

// CompareBinaryJSON compares two BinaryJSON values
func CompareBinaryJSON(a, b BinaryJSON) int {
	// Stub implementation - always return 0 (equal)
	return 0
}

// ParseBinaryJSONFromString parses JSON from string
func ParseBinaryJSONFromString(s string) (BinaryJSON, error) {
	// Stub implementation
	return BinaryJSON{TypeCode: JSONTypeCodeString, Value: []byte(s)}, nil
}

// CreateBinaryJSON creates a binary JSON from interface value
func CreateBinaryJSON(val interface{}) BinaryJSON {
	// Stub implementation
	return BinaryJSON{TypeCode: JSONTypeCodeObject, Value: []byte{}}
}

// ParseVectorFloat32 parses a vector from string
func ParseVectorFloat32(s string) (VectorFloat32, error) {
	// Stub implementation
	return VectorFloat32{}, nil
}

// IsZero checks if BinaryJSON is zero value
func (bj BinaryJSON) IsZero() bool {
	// Stub implementation
	return len(bj.Value) == 0
}

// hack package replacement - provides string/byte conversion utilities
type hackPkg struct{}

// String converts byte slice to string
func (hackPkg) String(b []byte) string {
	return string(b)
}

// Slice converts string to byte slice
func (hackPkg) Slice(s string) []byte {
	return []byte(s)
}

// hack is a global instance of hackPkg
var hack hackPkg



// IsTypeBlob returns a boolean indicating whether the tp is a blob type.
func IsTypeBlob(tp byte) bool {
	switch tp {
	case mysql.TypeTinyBlob, mysql.TypeMediumBlob, mysql.TypeBlob, mysql.TypeLongBlob:
		return true
	}
	return false
}

// IsTypeChar returns a boolean indicating
// whether the tp is the char type like a string type or a varchar type.
func IsTypeChar(tp byte) bool {
	return tp == mysql.TypeString || tp == mysql.TypeVarchar
}

// IsTypeVector returns whether tp is a vector type.
func IsTypeVector(tp byte) bool {
	return tp == mysql.TypeTiDBVectorFloat32
}

// IsTypeVarchar returns a boolean indicating
// whether the tp is the varchar type like a varstring type or a varchar type.
func IsTypeVarchar(tp byte) bool {
	return tp == mysql.TypeVarString || tp == mysql.TypeVarchar
}

// IsTypeUnspecified returns a boolean indicating whether the tp is the Unspecified type.
func IsTypeUnspecified(tp byte) bool {
	return tp == mysql.TypeUnspecified
}

// IsTypePrefixable returns a boolean indicating
// whether an index on a column with the tp can be defined with a prefix.
func IsTypePrefixable(tp byte) bool {
	return IsTypeBlob(tp) || IsTypeChar(tp)
}

// IsTypeFractionable returns a boolean indicating
// whether the tp can has time fraction.
func IsTypeFractionable(tp byte) bool {
	return tp == mysql.TypeDatetime || tp == mysql.TypeDuration || tp == mysql.TypeTimestamp
}

// IsTypeTime returns a boolean indicating
// whether the tp is time type like datetime, date or timestamp.
func IsTypeTime(tp byte) bool {
	return tp == mysql.TypeDatetime || tp == mysql.TypeDate || tp == mysql.TypeTimestamp
}

// IsTypeFloat indicates whether the type is TypeFloat
func IsTypeFloat(tp byte) bool {
	return tp == mysql.TypeFloat
}

// IsTypeInteger returns a boolean indicating whether the tp is integer type.
func IsTypeInteger(tp byte) bool {
	switch tp {
	case mysql.TypeTiny, mysql.TypeShort, mysql.TypeInt24, mysql.TypeLong, mysql.TypeLonglong, mysql.TypeYear:
		return true
	}
	return false
}

// IsTypeStoredAsInteger returns a boolean indicating whether the tp is stored as integer type.
func IsTypeStoredAsInteger(tp byte) bool {
	switch tp {
	case mysql.TypeTiny, mysql.TypeShort, mysql.TypeInt24, mysql.TypeLong, mysql.TypeLonglong:
		return true
	case mysql.TypeYear:
		return true
	// Enum and Set are stored as integer type but they can not be pushed down to TiFlash
	// case mysql.TypeEnum, mysql.TypeSet:
	// 	return true
	case mysql.TypeDatetime, mysql.TypeDate, mysql.TypeTimestamp, mysql.TypeDuration:
		return true
	}
	return false
}

// IsTypeNumeric returns a boolean indicating whether the tp is numeric type.
func IsTypeNumeric(tp byte) bool {
	switch tp {
	case mysql.TypeBit, mysql.TypeTiny, mysql.TypeInt24, mysql.TypeLong, mysql.TypeLonglong, mysql.TypeNewDecimal,
		mysql.TypeFloat, mysql.TypeDouble, mysql.TypeShort:
		return true
	}
	return false
}

// IsTypeBit returns a boolean indicating whether the tp is bit type.
func IsTypeBit(ft *FieldType) bool {
	return ft.GetType() == mysql.TypeBit
}

// IsTemporalWithDate returns a boolean indicating
// whether the tp is time type with date.
func IsTemporalWithDate(tp byte) bool {
	return IsTypeTime(tp)
}

// IsBinaryStr returns a boolean indicating
// whether the field type is a binary string type.
func IsBinaryStr(ft *FieldType) bool {
	return ft.GetCollate() == CollationBin && IsString(ft.GetType())
}

// IsNonBinaryStr returns a boolean indicating
// whether the field type is a non-binary string type.
func IsNonBinaryStr(ft *FieldType) bool {
	if ft.GetCollate() != CollationBin && IsString(ft.GetType()) {
		return true
	}
	return false
}

// NeedRestoredData returns if a type needs restored data.
// If the type is char and the collation is _bin, NeedRestoredData() returns false.
func NeedRestoredData(ft *FieldType) bool {
	if CollationEnabled() &&
		IsNonBinaryStr(ft) &&
		(!IsBinCollation(ft.GetCollate()) || IsTypeVarchar(ft.GetType())) &&
		ft.GetCollate() != "utf8mb4_0900_bin" {
		return true
	}
	return false
}

// IsString returns a boolean indicating
// whether the field type is a string type.
func IsString(tp byte) bool {
	return IsTypeChar(tp) || IsTypeBlob(tp) || IsTypeVarchar(tp) || IsTypeUnspecified(tp)
}

// IsStringKind returns a boolean indicating whether the tp is a string type.
func IsStringKind(kind byte) bool {
	return kind == KindString || kind == KindBytes
}

var kind2Str = map[byte]string{
	KindNull:          "null",
	KindInt64:         "bigint",
	KindUint64:        "unsigned bigint",
	KindFloat32:       "float",
	KindFloat64:       "double",
	KindString:        "char",
	KindBytes:         "bytes",
	KindBinaryLiteral: "bit/hex literal",
	KindMysqlDecimal:  "decimal",
	KindMysqlDuration: "time",
	KindMysqlEnum:     "enum",
	KindMysqlBit:      "bit",
	KindMysqlSet:      "set",
	KindMysqlTime:     "datetime",
	KindInterface:     "interface",
	KindMinNotNull:    "min_not_null",
	KindMaxValue:      "max_value",
	KindRaw:           "raw",
	KindMysqlJSON:     "json",
	KindVectorFloat32: "vector",
}

// TypeStr converts tp to a string.
func TypeStr(tp byte) string {
	switch tp {
	case mysql.TypeBit:
		return "bit"
	case mysql.TypeTiny:
		return "tinyint"
	case mysql.TypeShort:
		return "smallint"
	case mysql.TypeLong:
		return "int"
	case mysql.TypeFloat:
		return "float"
	case mysql.TypeDouble:
		return "double"
	case mysql.TypeNull:
		return "null"
	case mysql.TypeTimestamp:
		return "timestamp"
	case mysql.TypeLonglong:
		return "bigint"
	case mysql.TypeInt24:
		return "mediumint"
	case mysql.TypeDate:
		return "date"
	case mysql.TypeDuration:
		return "time"
	case mysql.TypeDatetime:
		return "datetime"
	case mysql.TypeYear:
		return "year"
	case mysql.TypeNewDate:
		return "date"
	case mysql.TypeVarchar:
		return "varchar"
	case mysql.TypeJSON:
		return "json"
	case mysql.TypeNewDecimal:
		return "decimal"
	case mysql.TypeEnum:
		return "enum"
	case mysql.TypeSet:
		return "set"
	case mysql.TypeTinyBlob:
		return "tinyblob"
	case mysql.TypeMediumBlob:
		return "mediumblob"
	case mysql.TypeLongBlob:
		return "longblob"
	case mysql.TypeBlob:
		return "blob"
	case mysql.TypeVarString:
		return "var_string"
	case mysql.TypeString:
		return "char"
	case mysql.TypeGeometry:
		return "geometry"
	case mysql.TypeTiDBVectorFloat32:
		return "vector"
	default:
		return "unknown"
	}
}

// KindStr converts kind to a string.
func KindStr(kind byte) (r string) {
	return kind2Str[kind]
}

// TypeToStr converts a field to a string.
// It is used for converting Text to Blob,
// or converting Char to Binary.
// Args:
//
//	tp: type enum
//	cs: charset
func TypeToStr(tp byte, cs string) (typeStr string, err error) {
	// Simplified implementation - in full version this would handle charset conversions
	typeStr = TypeStr(tp)
	return typeStr, nil
}

// EOFAsNil filtrates errors,
// If err is equal to io.EOF returns nil.
func EOFAsNil(err error) error {
	if ErrorEqual(err, io.EOF) {
		return nil
	}
	return errors.Trace(err)
}

// InvOp2 returns an invalid operation error.
func InvOp2(x, y any, o Op) (any, error) {
	return nil, errors.Errorf("Invalid operation: %v %v %v (mismatched types %T and %T)", x, o, y, x, y)
}

// overflow returns an overflowed error.
func overflow(v any, tp byte) error {
	return ErrOverflow.GenWithStack("constant %v overflows %s", v, TypeStr(tp))
}

// IsTypeTemporal checks if a type is a temporal type.
func IsTypeTemporal(tp byte) bool {
	switch tp {
	case mysql.TypeDuration, mysql.TypeDatetime, mysql.TypeTimestamp,
		mysql.TypeDate, mysql.TypeNewDate:
		return true
	}
	return false
}
