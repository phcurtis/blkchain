// Copyright 2018 phcurtis blkchain Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/phcurtis/fn"
)

// isolate all writes back to http.ResponseWriter to this file via
// these defined funcs.

func writeRaw(w http.ResponseWriter, data []byte) {
	if _, err := w.Write(data); err != nil {
		log.Panic(err)
	}
}

func writeJSON(w http.ResponseWriter, scode int, data, fullData []byte, verbose bool) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(scode)
	fn.LogCondMsg(verbose, "writeJSON-calledFrom:"+fn.Lvl(fn.Lpar)+
		fmt.Sprintf(":scode=%d", scode)+"\n"+string(fullData))

	if devMode {
		data = fullData
	}
	writeRaw(w, data)
}

func sendHTTPError(w http.ResponseWriter, httpScode, errCode int, errMsg, caller string) {
	hst := http.StatusText(httpScode)

	// prep stuff for internal (and verbose) details of error
	type ierrdetails struct {
		HTTPscode     int    `json:"statuscode"`
		HTTPscodeText string `json:"statuscodetext"`
		ErrCode       int    `json:"errcode"`
		ErrText       string `json:"errtext"`
		ErrMsg        string `json:"errmsg"`
		Caller        string `json:"caller"`
	}

	ibytes, ierr := json.MarshalIndent(struct {
		Details ierrdetails `json:"error"`
	}{ierrdetails{httpScode, hst, errCode, errText[errCode], errMsg, caller}}, "", "\t")
	if ierr != nil {
		log.Panic(ierr)
	}

	// prep stuff for client (limited) details of error
	type cerrdetails struct {
		HTTPscode int    `json:"clientstatuscode"`
		ClientMsg string `json:"clientmsg"`
	}
	clientMsg := fmt.Sprintf("'%d':[%s; icode=%d]", httpScode, hst, errCode)

	cbytes, cerr := json.MarshalIndent(struct {
		Details cerrdetails `json:"error"`
	}{cerrdetails{httpScode, clientMsg}}, "", "\t")
	if cerr != nil {
		log.Panic(cerr)
	}

	writeJSON(w, httpScode, cbytes, ibytes, verblvl > 0)
}
