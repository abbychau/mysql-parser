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
	"fmt"
	"log"
)

// Log provides basic logging functionality for error tracing
func Log(err error) {
	if err != nil {
		// Simple logging to stderr for self-contained operation
		log.Printf("Error: %v", err)
	}
}

// ErrorEqual checks if two errors are equivalent
func ErrorEqual(err1, err2 error) bool {
	if err1 == nil && err2 == nil {
		return true
	}
	if err1 == nil || err2 == nil {
		return false
	}
	return err1.Error() == err2.Error()
}

// ErrClass represents an error class for categorizing errors
type ErrClass struct {
	Name string
}

// NewStd creates a standard error from mysql error code
func (e *ErrClass) NewStd(code uint16) error {
	return fmt.Errorf("mysql error %d", code)
}

// ClassDDL represents DDL error class
var ClassDDL = &ErrClass{Name: "DDL"}

// TError is a more sophisticated error type with additional methods
type TError struct {
	code uint16
	message string
}

func (e *TError) Error() string {
	return fmt.Sprintf("mysql error %d: %s", e.code, e.message)
}

// FastGenByArgs generates error with arguments
func (e *TError) FastGenByArgs(args ...interface{}) error {
	return fmt.Errorf("mysql error %d: %v", e.code, args)
}

// GenWithStackByArgs generates error with stack and arguments
func (e *TError) GenWithStackByArgs(args ...interface{}) error {
	return fmt.Errorf("mysql error %d: %v", e.code, args)
}

// Equal checks if errors are equal
func (e *TError) Equal(other error) bool {
	if otherT, ok := other.(*TError); ok {
		return e.code == otherT.code
	}
	return false
}

// GenWithStack generates error with stack
func (e *TError) GenWithStack(format string, args ...interface{}) error {
	return fmt.Errorf("mysql error %d: "+format, append([]interface{}{e.code}, args...)...)
}

// FastGen generates error quickly
func (e *TError) FastGen(args ...interface{}) error {
	return fmt.Errorf("mysql error %d: %v", e.code, args)
}

// Basic error functions that return TError for compatibility
func NewStd(code uint16) *TError {
	return &TError{code: code, message: fmt.Sprintf("error %d", code)}
}