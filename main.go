// Copyright 2018 phcurtis blkchain Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package main (command blkchain) implements an HTTP API server
// which accepts limited requests demoing a limited blockchain simulator.
package main

import (
	"time"

	"github.com/phcurtis/fn"
)

// Version of this program
const Version = "0.03"

func main() {
	timeofinv = time.Now() // capture time of invocation.
	prelimsCLI(false)
	onExitFunc := fn.LogCondTraceMsgp(devMode || verblvl > 0, "")

	msg, excode := APIserver()

	fn.LogCondMsg(!(devMode || verblvl > 0) &&
		excode != ExcodeProgramSuccess, msg)

	onExitFunc(&msg)
	osExit(excode)
}
