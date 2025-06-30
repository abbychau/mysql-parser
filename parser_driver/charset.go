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

// Basic charset constants needed for self-contained operation
const (
	// CharsetBin is used for marking binary charset.
	CharsetBin = "binary"
	// CharsetUTF8 is the default charset for string types.
	CharsetUTF8 = "utf8"
	// CharsetUTF8MB4 represents 4 bytes utf8, which works the same way as utf8 in Go.
	CharsetUTF8MB4 = "utf8mb4"
)

const (
	// CollationBin is the default collation for CharsetBin.
	CollationBin = "binary"
	// CollationUTF8 is the default collation for CharsetUTF8.
	CollationUTF8 = "utf8_bin"
	// CollationUTF8MB4 is the default collation for CharsetUTF8MB4.
	CollationUTF8MB4 = "utf8mb4_bin"
)

// Simplified encoding operations for basic compatibility
const (
	OpReplace = iota
	OpEncodeNoErr
	OpDecode
)

// Encoding types for basic compatibility
const (
	EncodingTpUTF8 = iota
	EncodingTpASCII
)

// Encoding provides a minimal interface for character encoding operations
type Encoding interface {
	Transform(dst, src []byte, op int) ([]byte, error)
	Tp() int
	IsValid(s []byte) bool
}

// Collation provides basic collation information
type Collation struct {
	CharsetName string
}

// GetCollationByName returns a basic collation info
func GetCollationByName(name string) (*Collation, error) {
	// Simplified implementation - just return charset based on collation name
	if name == CollationBin {
		return &Collation{CharsetName: CharsetBin}, nil
	}
	if name == CollationUTF8 {
		return &Collation{CharsetName: CharsetUTF8}, nil
	}
	if name == CollationUTF8MB4 {
		return &Collation{CharsetName: CharsetUTF8MB4}, nil
	}
	// Default to UTF8MB4 for unknown collations
	return &Collation{CharsetName: CharsetUTF8MB4}, nil
}

// FindEncodingTakeUTF8AsNoop returns a basic encoding handler
func FindEncodingTakeUTF8AsNoop(charset string) Encoding {
	return &basicEncoding{charset: charset}
}

// FindEncoding returns a basic encoding handler
func FindEncoding(charset string) Encoding {
	return &basicEncoding{charset: charset}
}

// basicEncoding provides a minimal encoding implementation
type basicEncoding struct {
	charset string
}

func (e *basicEncoding) Transform(dst, src []byte, op int) ([]byte, error) {
	// Simplified implementation - just return source bytes for most operations
	if op == OpDecode || op == OpEncodeNoErr {
		return src, nil
	}
	if op == OpReplace {
		// For replace operations, just return source (no actual replacement)
		return src, nil
	}
	return src, nil
}

func (e *basicEncoding) Tp() int {
	if e.charset == CharsetUTF8 || e.charset == CharsetUTF8MB4 {
		return EncodingTpUTF8
	}
	return EncodingTpASCII
}

func (e *basicEncoding) IsValid(s []byte) bool {
	// Simplified implementation - always return true for basic validation
	return true
}

// EncodingUTF8MB3StrictImpl provides a basic UTF8 encoding for compatibility
var EncodingUTF8MB3StrictImpl = &basicEncoding{charset: CharsetUTF8}

// NewCollationEnabled returns whether new collation is enabled
func NewCollationEnabled() bool {
	// Stub implementation - return false for conservative behavior
	return false
}

// CharsetInfo provides basic charset information
type CharsetInfo struct {
	Name   string
	Maxlen int
}

// GetCharsetInfo returns basic charset information
func GetCharsetInfo(charset string) (*CharsetInfo, error) {
	// Set common maxlen values based on charset
	maxlen := 1 // Default for single-byte charsets
	switch charset {
	case CharsetUTF8:
		maxlen = 3
	case CharsetUTF8MB4:
		maxlen = 4
	case CharsetBin:
		maxlen = 1
	}
	return &CharsetInfo{Name: charset, Maxlen: maxlen}, nil
}