// Package parser_driver provides WASM-compatible replacements for TiDB's util packages
package parser_driver

import (
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"time"
)

// =============== util/hack replacements ===============

// Slice converts string to []byte without memory allocation in WASM-safe way
func Slice(s string) []byte {
	if s == "" {
		return nil
	}
	// For WASM, we use a simple conversion that may allocate but is safe
	return []byte(s)
}

// String converts []byte to string without memory allocation in WASM-safe way
func String(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	// For WASM, we use a simple conversion that may allocate but is safe
	return string(b)
}

// =============== util/logutil replacements ===============

// BgLogger provides a simple background logger for WASM
var BgLogger = &Logger{}

// Logger is a simple logger implementation for WASM
type Logger struct{}

// Info logs an info message
func (l *Logger) Info(msg string, fields ...interface{}) {
	log.Printf("[INFO] %s %v", msg, fields)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, fields ...interface{}) {
	log.Printf("[WARN] %s %v", msg, fields)
}

// Error logs an error message
func (l *Logger) Error(msg string, fields ...interface{}) {
	log.Printf("[ERROR] %s %v", msg, fields)
}

// =============== util/dbterror replacements ===============

// Error represents a database error in WASM-compatible way
type DBError struct {
	Code    int
	Message string
	Class   string
}

// Error implements the error interface
func (e *DBError) Error() string {
	return fmt.Sprintf("[%s:%d] %s", e.Class, e.Code, e.Message)
}

// NOTE: NewStd is commented out because it conflicts with terror.go
// NewStd creates a new standard error
// func NewStd(code int) *DBError {
// 	return &DBError{
// 		Code:    code,
// 		Message: fmt.Sprintf("Error code %d", code),
// 		Class:   "STD",
// 	}
// }

// NewStdErr creates a new standard error with custom message
func NewStdErr(code int, message string) *DBError {
	return &DBError{
		Code:    code,
		Message: message,
		Class:   "STD",
	}
}

// =============== util/collate replacements ===============

// CollationEnabled returns whether collation is enabled (simplified for WASM)
func CollationEnabled() bool {
	return false // Simplified for WASM - disable complex collations
}

// GetCollator returns a collator for the given collation
func GetCollator(collation string) Collator {
	return &simpleCollator{collation: collation}
}

// Collator interface for string comparison
type Collator interface {
	Compare(a, b string) int
	Key(str string) []byte
}

// simpleCollator provides basic collation for WASM
type simpleCollator struct {
	collation string
}

// Compare compares two strings (case-insensitive for most collations)
func (c *simpleCollator) Compare(a, b string) int {
	// Simplified comparison - case insensitive for most MySQL collations
	if strings.Contains(c.collation, "_ci") || c.collation == "" {
		a = strings.ToLower(a)
		b = strings.ToLower(b)
	}
	
	if a < b {
		return -1
	} else if a > b {
		return 1
	}
	return 0
}

// Key returns the sort key for a string
func (c *simpleCollator) Key(str string) []byte {
	// Simplified - just convert to bytes with case normalization
	if strings.Contains(c.collation, "_ci") || c.collation == "" {
		str = strings.ToLower(str)
	}
	return []byte(str)
}

// IsCICollation returns true if the collation is case-insensitive
func IsCICollation(collation string) bool {
	return strings.Contains(collation, "_ci") || collation == ""
}

// IsBinCollation returns true if the collation is binary
func IsBinCollation(collation string) bool {
	return strings.Contains(collation, "_bin")
}

// =============== util/mathutil replacements ===============

// Max returns the maximum of two integers
func Max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Min returns the minimum of two integers
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// MaxUint64 returns the maximum of two uint64 values
func MaxUint64(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}

// MinUint64 returns the minimum of two uint64 values
func MinUint64(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}

// MaxFloat64 returns the maximum of two float64 values
func MaxFloat64(a, b float64) float64 {
	return math.Max(a, b)
}

// MinFloat64 returns the minimum of two float64 values
func MinFloat64(a, b float64) float64 {
	return math.Min(a, b)
}

// Abs returns the absolute value of an integer
func Abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

// =============== planner/cascades/base replacements ===============

// Hasher interface for WASM-compatible hashing
type Hasher interface {
	Sum64() uint64
	Write([]byte) (int, error)
}

// SimpleHasher provides a basic hasher implementation for WASM
type SimpleHasher struct {
	hash uint64
}

// NewHasher creates a new hasher
func NewHasher() *SimpleHasher {
	return &SimpleHasher{hash: 0}
}

// Sum64 returns the current hash value
func (h *SimpleHasher) Sum64() uint64 {
	return h.hash
}

// Write adds data to the hash
func (h *SimpleHasher) Write(data []byte) (int, error) {
	// Simple FNV-like hash for WASM compatibility
	for _, b := range data {
		h.hash ^= uint64(b)
		h.hash *= 1099511628211 // FNV prime
	}
	return len(data), nil
}

// =============== util/dbterror replacements ===============

var (
	// ErrUnsupportedModifyColumn represents unsupported modify column error
	ErrUnsupportedModifyColumn = &DBError{
		Code:    1105, // ER_UNKNOWN_ERROR
		Message: "Unsupported modify column",
		Class:   "DDL",
	}
)

// GenWithStackByArgs generates error with message arguments
func (e *DBError) GenWithStackByArgs(args ...interface{}) error {
	msg := e.Message
	if len(args) > 0 {
		msg = fmt.Sprintf("%s: %v", msg, args[0])
	}
	return &DBError{
		Code:    e.Code,
		Message: msg,
		Class:   e.Class,
	}
}

// =============== util/parser replacements ===============

// Simple parser utilities for WASM compatibility

// Char parses a single character
func Char(s string, c byte) (string, error) {
	if len(s) == 0 || s[0] != c {
		return "", fmt.Errorf("expected %c", c)
	}
	return s[1:], nil
}

// Space parses one or more spaces
func Space(s string, min int) (string, error) {
	count := 0
	for i := 0; i < len(s) && s[i] == ' '; i++ {
		count++
	}
	if count < min {
		return "", fmt.Errorf("expected at least %d spaces", min)
	}
	return s[count:], nil
}

// Space0 parses zero or more spaces
func Space0(s string) string {
	i := 0
	for i < len(s) && s[i] == ' ' {
		i++
	}
	return s[i:]
}

// Number parses a number from string
func Number(s string) (int, string, error) {
	if len(s) == 0 {
		return 0, s, fmt.Errorf("no number found")
	}
	
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	
	if i == 0 {
		return 0, s, fmt.Errorf("no number found")
	}
	
	num, err := strconv.Atoi(s[:i])
	return num, s[i:], err
}

// Digit parses a specific number of digits
func Digit(s string, n int) (string, string, error) {
	if len(s) < n {
		return "", s, fmt.Errorf("not enough digits")
	}
	
	for i := 0; i < n; i++ {
		if s[i] < '0' || s[i] > '9' {
			return "", s, fmt.Errorf("expected digit")
		}
	}
	
	return s[:n], s[n:], nil
}

// AnyPunct parses any punctuation character
func AnyPunct(s string) (string, error) {
	if len(s) == 0 {
		return "", fmt.Errorf("no punctuation found")
	}
	
	c := s[0]
	if (c >= '!' && c <= '/') || (c >= ':' && c <= '@') || (c >= '[' && c <= '`') || (c >= '{' && c <= '~') {
		return s[1:], nil
	}
	
	return "", fmt.Errorf("expected punctuation")
}

// =============== Context definition ===============

// Context represents an interface for type conversion and validation context
type Context interface {
	Flags() ContextFlags
	AppendWarning(err error)
	HandleTruncate(err error) error
	Location() *time.Location
}

// StmtContext is a concrete implementation of Context
type StmtContext struct {
	IgnoreTruncate    bool
	TruncateAsWarning bool
	flags             ContextFlags
	warnings          []error
}

// ContextFlags represents context flags
type ContextFlags struct {
	InInsertStmt           bool
	InUpdateStmt           bool
	InDeleteStmt           bool
	InSelectStmt           bool
	ignoreZeroInDate       bool
	DividedByZeroAsWarning bool
	ignoreTruncateErr      bool
	truncateAsWarning      bool
}

// Flags returns the context flags
func (c *StmtContext) Flags() ContextFlags {
	return c.flags
}

// IgnoreTruncateErr returns whether to ignore truncate errors
func (cf ContextFlags) IgnoreTruncateErr() bool {
	return cf.ignoreTruncateErr
}

// TruncateAsWarning returns whether to treat truncate as warning
func (cf ContextFlags) TruncateAsWarning() bool {
	return cf.truncateAsWarning
}

// AllowNegativeToUnsigned returns whether negative values can be converted to unsigned
func (cf ContextFlags) AllowNegativeToUnsigned() bool {
	return false // Default conservative behavior
}

// SkipUTF8Check returns whether to skip UTF8 validation
func (cf ContextFlags) SkipUTF8Check() bool {
	return false // Default to checking
}

// SkipASCIICheck returns whether to skip ASCII validation
func (cf ContextFlags) SkipASCIICheck() bool {
	return false // Default to checking
}

// SkipUTF8MB4Check returns whether to skip UTF8MB4 validation
func (cf ContextFlags) SkipUTF8MB4Check() bool {
	return false // Default to checking
}


// IgnoreZeroInDate returns whether to ignore zero in date
func (cf ContextFlags) IgnoreZeroInDate() bool {
	return cf.ignoreZeroInDate
}

// IgnoreInvalidDateErr returns whether to ignore invalid date errors
func (cf ContextFlags) IgnoreInvalidDateErr() bool {
	return false // Default conservative behavior
}

// CastTimeToYearThroughConcat returns whether to cast time to year through concatenation
func (cf ContextFlags) CastTimeToYearThroughConcat() bool {
	return false // Default conservative behavior
}

// IgnoreZeroDateErr returns whether to ignore zero date errors
func (cf ContextFlags) IgnoreZeroDateErr() bool {
	return false // Default conservative behavior
}

// AppendWarning adds a warning to the context
func (c *StmtContext) AppendWarning(err error) {
	if c.warnings == nil {
		c.warnings = make([]error, 0)
	}
	c.warnings = append(c.warnings, err)
}

// Location returns the timezone location for time conversions
func (c *StmtContext) Location() *time.Location {
	// Return UTC by default
	return time.UTC
}

// DefaultStmtNoWarningContext is a default context with no warnings
var DefaultStmtNoWarningContext Context = &StmtContext{
	IgnoreTruncate:    false,
	TruncateAsWarning: false,
}

// HandleTruncate method is implemented in truncate.go