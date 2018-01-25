// Copyright 2018 phcurtis blkchain Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"expvar"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync/atomic"

	"github.com/phcurtis/fn"
)

// this file contains 'auxilary' stuff, no specific category.

func publishExpvars() {
	expvar.Publish("1a-blkctime-duration", expvar.Func(func() interface{} { return blkctimestr }))
	expvar.Publish("1a-blkfile", expvar.Func(func() interface{} { return blkfile }))
	expvar.Publish("1a-blktxmax", expvar.Func(func() interface{} { return blktxmax }))
	expvar.Publish("1b-curblktxcnt", expvar.Func(func() interface{} { return atomic.LoadUint64(&curblktxcnt) }))
	expvar.Publish("1b-totblkappSinv", expvar.Func(func() interface{} { return atomic.LoadUint64(&totblkappSinv) }))
	expvar.Publish("1b-tottxappSinv", expvar.Func(func() interface{} { return atomic.LoadUint64(&tottxappSinv) }))
	expvar.Publish("1b-totwrtbytesSinv", expvar.Func(func() interface{} { return atomic.LoadUint64(&totwrtbytesSinv) }))
}

func logPanic(v ...interface{}) {
	fn.LogCondMsg(true, fmt.Sprintf("err:%v calledBy:%s", v, fn.LvlInfoShort(fn.Lpar)))
	log.Panic(v)
}

func callerPar() string {
	return "callerFunc:" + fn.LvlInfoShort(fn.Lpar)
}

func decodeBody(r *http.Request, v interface{}) error {
	defer func() { _ = r.Body.Close() }()
	return json.NewDecoder(r.Body).Decode(v)
}

// likely will using for search for existing key feature.
func decodeFile(fname string, v interface{}) ([]byte, error) {
	data, err := ioutil.ReadFile(fname)
	if err == nil {
		err = json.Unmarshal(data, v)
	}
	return data, err
}
