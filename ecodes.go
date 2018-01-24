// Copyright 2018 phcurtis blkchain Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"

	"github.com/phcurtis/fn"
)

// This file contains 'exit codes', 'error codes' and related.

// Excode ... exit codes constants
const (
	ExcodeProgramSuccess       = 0   //
	ExcodeGeneralError         = 1   //
	ExcodePanic                = 2   // verify if this is norm for a panic; happens on xubuntu sys
	ExcodeHTTPServerErr        = 3   //
	ExcodeCtrlcSignal          = 4   // control-c, or process was ended via bash> kill pid or similar
	ExcodeFileOpenErr          = 5   //
	ExcodeSystemMonitorKill    = 137 // seen using xubuntu 'system monitor' kill, json file likely will have issues AVOID!
	ExcodeCliHelpUsage         = 200 //
	ExcodeCliFlagissue         = 201 //
	ExcodeCliUnrecognizedInput = 202 //
	ExcodeCliVersionReq        = 203 //
)

// Error codes constants
const (
	ErrJSONmarshal       = 100
	ErrJSONmarshalIndent = 101
	ErrJSONunMarshal     = 102
	ErrJSONdecodeBody    = 103
	ErrJSONdecodeFile    = 104
)

var errText = map[int]string{
	ErrJSONmarshal:       "error json.Marshal",
	ErrJSONmarshalIndent: "error json.MarshalIndent",
	ErrJSONunMarshal:     "error json.Unmarshal",
	ErrJSONdecodeBody:    "error json.decodeBody",
	ErrJSONdecodeFile:    "error json.decodeFile",
}

// ErrText - returns error text for given 'code'
func ErrText(code int) string {
	text, ok := errText[code]
	if ok {
		return text
	}
	msg := fmt.Sprintf("ErrText(%d) not Defined INFORM developer", code)
	fn.LogCondMsg(true, msg+"\n")
	return msg
}
