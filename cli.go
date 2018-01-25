// Copyright 2018 phcurtis blkchain Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

// This file contains cli related hooks to support command line options.

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/phcurtis/fn"
)

// see corresponding init() for flagsStruct variables descriptions
type flagsStruct struct {
	blkctimestr    blkCtimeStr // used as special hook to validate time specified meets min duration.
	blktxmax       int
	blkfile        string
	devmode        bool
	expvars        bool
	fnlogflags     int
	showinvdetails bool
	showversion    bool
	srvport        int
	srvurl         string
	verblvl        int
}

// these constants might belong in tx.go TBD
const (
	blkctimeMin    = time.Duration(time.Second * 5) // TODO determine min time duration allowed.
	blkctimestrDef = "1m"                           // TODO determine bckctimestr default time duration.
)

var flags flagsStruct
var flag2 *flag.FlagSet

// type to procure a 'valid' blkctime duration string via flag.Var function.
type blkCtimeStr string

func (t blkCtimeStr) String() string {
	return string(t)
}
func (t blkCtimeStr) Set(value string) error {
	dur, err := time.ParseDuration(value)
	if err != nil {
		return err
	}

	if dur < blkctimeMin {
		err = fmt.Errorf("time duration(%v) is too short; minTimeDuration(%v)",
			dur, blkctimeMin)
		return err
	}

	flags.blkctimestr = blkCtimeStr(value)
	return nil
}

func init() {

	// use a newflag set with ContinueOnError so we could allow go 'defer's to run.
	flag2 = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	err := flags.blkctimestr.Set(blkctimestrDef)
	if err != nil {
		log.Panic(err)
	}
	flag2.Var(flags.blkctimestr, "blk.ctime", "block commit time duration")

	flag2.StringVar(&flags.blkfile, "blk.file", "blkchain.json", "name of blockchain json file")
	flag2.IntVar(&flags.blktxmax, "blk.txmax", 0, "<1 =off, >0 =max transactions in a block")
	flag2.BoolVar(&flags.devmode, "devmode", false, "development mode")
	flag2.BoolVar(&flags.expvars, "expvars", false, "expose expvars (via /debug/vars)")
	flag2.IntVar(&flags.fnlogflags, "fnlogflags", fn.LflagsDef, "see fn.LogSetFlags")
	flag2.BoolVar(&flags.showinvdetails, "invdetails", false, "show invocation details")
	flag2.BoolVar(&flags.showversion, "version", false, "show version and exit")
	flag2.IntVar(&flags.srvport, "srv.port", 8080, "server port to listen on")
	flag2.StringVar(&flags.srvurl, "srv.url", "localhost", "server url")
	flag2.IntVar(&flags.verblvl, "verblvl", 0, "verbosity level")
}

func osExit(excode int) {
	if excode != ExcodeProgramSuccess {
		fmt.Fprintf(os.Stderr, fmt.Sprintf("exiting:exitcode=%d(%s) caller:%s\n",
			excode, ExcodeText(excode), fn.LvlInfoShort(fn.Lpar)))
	}
	os.Exit(excode)
}

func prelimsCLI(gotest bool) {
	flag2.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s: (Version:%s)\n", os.Args[0], Version)
		flag2.PrintDefaults()
	}

	if err := flag2.Parse(os.Args[1:]); err != nil {
		if err == flag.ErrHelp {
			osExit(ExcodeCliHelpUsage)
		}
		osExit(ExcodeCliFlagissue)
	}

	if flag2.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "unrecognized %v\nUsage of %s (Version:%s):\n",
			flag2.Args(), os.Args[0], Version)
		flag2.PrintDefaults()
		osExit(ExcodeCliUnrecognizedInput)
	}

	fn.LogSetFlags(flags.fnlogflags)

	if flags.showversion {
		fn.LogCondMsg(true, fmt.Sprintf("%s version=%s\n", os.Args[0], Version))
		osExit(ExcodeCliVersionReq)
	}

	if flags.showinvdetails || flags.verblvl > 3 {
		fn.LogCondMsg(true, fmt.Sprintf("%s version=%s\n", os.Args[0], Version))
		fn.LogCondMsg(true, fmt.Sprintf("%v\n", os.Args))
		fn.LogCondMsg(true, fmt.Sprintf("flags:%+v\n", flags))
	}

	// currently stdlib log is used for panic otherwise github.com/phcurtis/fn is used.
	log.SetPrefix("Log:")

	// set related package vars from cli (flags) vars, isolated to ease coding to a package library.
	blkctime, _ = time.ParseDuration(string(flags.blkctimestr))
	blkctimestr = string(flags.blkctimestr)
	blkfile = flags.blkfile
	blktxmax = flags.blktxmax
	devMode = flags.devmode
	expvars = flags.expvars
	fnlogflags = flags.fnlogflags
	srvport = flags.srvport
	srvurl = flags.srvurl
	verblvl = flags.verblvl
}
