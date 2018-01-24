// Copyright 2018 phcurtis blkchain Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/phcurtis/fn"
)

// this file contains functions related to transactions (tx) processing.

// for most of following vars see definitions in flagsStruct
var (
	blkctime    time.Duration
	blkctimestr string
	blktxmax    int
	blkfile     string
	blkfilep    *os.File
	devMode     bool
	expvars     bool
	fnlogflags  int
	srvurl      string
	srvport     int
	timeofinv   time.Time // time of invocation
	verblvl     int
)

type txStruct struct {
	ID        string `json:"id"` // 64 len hexstring of sha256.
	Key       string `json:"key"`
	Value     string `json:"value"`
	TimeStamp int64  `json:"timestamp"`
}

type errStruct struct {
	ClientErrMsg string `json:"clienterrmsg"`
}

// Blk - block struct
type Blk struct {
	PrevHash     string     `json:"prev-block-hash"` // 64 len hexstring of sha256
	BlockHash    string     `json:"block-hash"`      // 64 len hexstring of sha256
	Transactions []txStruct `json:"transactions"`
}

var (
	bcmu            sync.Mutex // block chain mutex
	blk             Blk        // a single block in a block chain
	curblktxcnt     uint64     // use with atomic cur block transaction count if 0 no active block.
	totblkappSinv   uint64     // use with atomic total blocks appended to file since invocation
	tottxappSinv    uint64     // use with atomic total transactions appended to file since invocation
	totwrtbytesSinv uint64     // use with atomic total bytes written to file since invocation
)

// expects bcmu.Lock mutext to be active
func (b *Blk) append2File() {
	defer fn.LogCondTrace(verblvl > 1)()

	// check if Transactions were already written, could happen because of timer.
	if b.Transactions == nil {
		return
	}

	// compute the current block hash
	var buf bytes.Buffer
	// if first block during this [program] invocation.
	if totblkappSinv == 0 {
		b.PrevHash = strings.Repeat("0", 64) // length of hexed sha256hash
	}
	buf.WriteString(b.PrevHash)
	for i := 0; i < len(b.Transactions); i++ {
		buf.Write([]byte(b.Transactions[i].ID[:]))
	}
	src := sha256.Sum256(buf.Bytes())
	dst := make([]byte, hex.EncodedLen(len(src)))
	hex.Encode(dst, src[:])
	b.BlockHash = string(dst[:])

	bytes1, jerr := json.Marshal(b)
	if jerr != nil {
		log.Panic("json.Marshal:" + jerr.Error()) // TODO later better error msg even though should not happen :)
		return
	}

	// adjust json that is to be added to blockchain file.
	var buf1 bytes.Buffer
	// if first block
	if totblkappSinv == 0 {
		// adjust stuff to make file json parse-able as well as making invocation name contain epoch-ts..
		if blkchainFileSize() == 0 {
			buf1.Write([]byte("{"))
		} else {
			// TODO verify '}' is last char in file probably should do at open time.
			// need to replace last char which must be and is assumed to be '}' in file with a ',' and add linefeed.
			n, err := blkfilep.Seek(blkchainFileSize()-1, os.SEEK_SET)
			if err != nil {
				log.Panic(err)
			}
			if n != blkchainFileSize()-1 {
				log.Panic(err)
			}
			buf1.Write([]byte(`,` + "\n"))
		}
		inv := fmt.Sprintf(`"invts-%d":[`, timeofinv.Unix())
		buf1.Write([]byte(inv))
	} else {
		buf1.Write([]byte(`,` + "\n"))
	}

	// write it to the blockchain file.
	buf1.Write(bytes1)
	bytes2 := buf1.Bytes()
	if _, err := blkfilep.Write(bytes2); err != nil {
		log.Panic("appendWriteError:" + err.Error())
	}

	fn.LogCondMsg(verblvl > 2, fmt.Sprintf("curblkwrtbytes:%d BlockHash:%v", len(bytes2), b.BlockHash))

	// update counters.
	atomic.AddUint64(&totblkappSinv, 1)
	atomic.AddUint64(&tottxappSinv, uint64(len(b.Transactions)))
	atomic.AddUint64(&totwrtbytesSinv, uint64(len(bytes2)))
	atomic.StoreUint64(&curblktxcnt, 0)

	// get ready for possible next block
	b.Transactions = nil
	b.PrevHash = b.BlockHash
	b.BlockHash = ""
}

func (b *Blk) flush() {
	defer fn.LogCondTrace(verblvl > 2)()
	bcmu.Lock()
	defer bcmu.Unlock()

	b.append2File()
}

var flushtimer *time.Timer

func (b *Blk) setTimerFlushBlk() {
	defer fn.LogCondTrace(verblvl > 3)()
	flushtimer = time.AfterFunc(blkctime, b.flush)
}

func stopTimerFlushBlkll() {
	defer fn.LogCondTrace(verblvl > 3)()
	if flushtimer != nil {
		flushtimer.Stop()
	}
}

// version of 'stopTimerFlushBlk' which wraps call with bcmu (mutex).
func stopTimerFlushBlk() {
	defer fn.LogCondTrace(verblvl > 3)()
	bcmu.Lock()
	defer bcmu.Unlock()
	stopTimerFlushBlkll()
}

// add a tranaction to the current block if none init a new block.
func (tx *txStruct) addToBlock() {
	defer fn.LogCondTrace(verblvl > 2)()
	bcmu.Lock()
	defer bcmu.Unlock()

	blk.Transactions = append(blk.Transactions, *tx)
	lenbc := len(blk.Transactions)
	atomic.StoreUint64(&curblktxcnt, uint64(len(blk.Transactions)))

	// if first transaction in a block and max transactions in a block is not 1.
	if lenbc == 1 && blktxmax != 1 {
		blk.setTimerFlushBlk()
	}

	// if max tranactions in a block is 'on' i.e. >0.
	if blktxmax > 0 {
		if lenbc == blktxmax {
			// if timer was set
			if blktxmax != 1 {
				stopTimerFlushBlkll()
			}
			blk.append2File()
		}
	}
}

// hashTx - hashes a given tranaction
func (tx *txStruct) hashTx() {
	defer fn.LogCondTrace(verblvl > 2)()
	tim := fmt.Sprintf("%v", tx.TimeStamp)
	src := sha256.Sum256([]byte(tx.Key + tx.Value + tim))
	dst := make([]byte, hex.EncodedLen(len(src)))
	hex.Encode(dst, src[:])
	tx.ID = string(dst[:])

	fn.LogCondMsg(verblvl > 3, fmt.Sprintf("tx.Key=%v tx.ID=%v\n", tx.Key, tx.ID))
}

// cepTx - client entry point for: /tx?key=keyname&value=valuestring.
func cepTx(w http.ResponseWriter, r *http.Request) {
	defer fn.LogCondTrace(devMode || verblvl > 2)()
	key := r.FormValue("key")
	val := r.FormValue("value")
	if key == "" || val == "" {
		msg := fmt.Sprintf("Error: both tranaction key and value must be set; key=%q value=%q", key, val)
		bytes, jerr := json.Marshal(errStruct{ClientErrMsg: msg})
		if jerr != nil {
			sendHTTPError(w, http.StatusInternalServerError, ErrJSONmarshal,
				"error JSON marshal of transaction", callerPar())
			return
		}
		writeJSON(w, http.StatusBadRequest, bytes, bytes, true)
		return
	}

	tx := &txStruct{Key: key, Value: val, TimeStamp: time.Now().Unix()}
	tx.hashTx()
	bytes, jerr := json.Marshal(tx)
	if jerr != nil {
		sendHTTPError(w, http.StatusInternalServerError, ErrJSONmarshal,
			"error JSON marshal of transaction", callerPar())
		return
	}
	tx.addToBlock()

	writeJSON(w, http.StatusCreated, bytes, bytes, verblvl > 2)
}

// cepSearchTx - client entry point for: /searchtx?key=keyname.
func cepSearchTx(w http.ResponseWriter, r *http.Request) {
	defer fn.LogCondTrace(devMode || verblvl > 2)()

	msg := fmt.Sprintf("Error: not implemented yet")
	bytes, jerr := json.Marshal(errStruct{ClientErrMsg: msg})
	if jerr != nil {
		sendHTTPError(w, http.StatusInternalServerError, ErrJSONmarshal,
			"error JSON marshal of transaction", callerPar())
		return
	}
	writeJSON(w, http.StatusNotImplemented, bytes, bytes, true)
	return

	//key := r.FormValue("key")
	//if key == "" {
	//	log.Println("search transaction key missing")
	//	return
	//}
}

// set up routes to be http served.
func routesSetup() *http.Server {
	defer fn.LogCondTrace(verblvl > 2)()
	r := mux.NewRouter().StrictSlash(false)

	if expvars {
		publishExpvars()
		r.PathPrefix("/debug/vars").Handler(http.DefaultServeMux)
	}

	r.HandleFunc("/tx", cepTx)
	r.HandleFunc("/searchtx", cepSearchTx)

	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", srvurl, srvport),
		Handler: r,
	}
	return server
}

// signal codes.
const (
	sigTerminate = 1
	sigSrvErr    = 2
)

var signalCh = make(chan int)

func catchProcessTerminate() {
	defer fn.LogCondTrace(verblvl > 2)()
	// catch 'process terminate' including ctrl-c so a smooth shutdown is possible
	go func() {
		termCh := make(chan os.Signal)
		signal.Notify(termCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
		for {
			select {
			case <-termCh:
				signalCh <- sigTerminate
			}
		}
	}()
}

func blkchainFileSize() (cursize int64) {
	fi, err := os.Stat(blkfile)
	if err != nil {
		log.Panic(err)
	}
	return fi.Size()
}

func blkchainFileStat(msg string) {
	size := blkchainFileSize()
	fn.LogCondMsg(true, fmt.Sprintf("blkfile:%q %sSize:%d (MiB:%.4f)\n",
		blkfile, msg, size, float64(size)/(1024.0*1024)))
	return
}

// APIserver - a small(limited) http api server that records 'transactions' in a blockchain.
func APIserver() (msg string, excode int) {
	defer fn.LogCondTrace(verblvl > 1)()
	catchProcessTerminate()
	excode = ExcodeGeneralError // for now no way to exit except by an error

	var err error
	//blkfilep, err = os.OpenFile(blkfile, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0600)
	blkfilep, err = os.OpenFile(blkfile, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return fmt.Sprintf("error opening file:%q err=%v", blkfile, err), ExcodeFileOpenErr
	}
	defer func() { _ = blkfilep.Close() }()

	if verblvl > 1 {
		blkchainFileStat("opening")
	}
	server := routesSetup()
	var srvmsg string
	go func(srvmsg *string) {
		fn.LogCondMsg(verblvl > 0, fmt.Sprintf("calling server.ListenAndServe:Addr=%q\n", server.Addr))
		if err := server.ListenAndServe(); err != nil {
			*srvmsg = err.Error()
			signalCh <- sigSrvErr
		}
	}(&srvmsg)

	code := <-signalCh
	switch code {
	case sigTerminate:
		if err := server.Shutdown(nil); err != nil {
			log.Panic(err)
		}
		stopTimerFlushBlk()
		blk.flush()
		msg = "exiting: due to 'terminate' signal"
		excode = ExcodeCtrlcSignal
	case sigSrvErr:
		msg = "exiting:srvErr: " + srvmsg
		fn.LogCondMsg(verblvl > 0, msg)
		excode = ExcodeHTTPServerErr
	default:
		log.Panic(fmt.Sprintf("unrecognized code:%v\n", code))
	}

	// add closing json syntax if any blocks where written on this invocation.
	bytesAdd := "]}"
	if totblkappSinv > 0 {
		if _, err := blkfilep.Write([]byte(bytesAdd)); err != nil {
			log.Panic("appendWriteError:" + err.Error())
		}
	}
	atomic.AddUint64(&totwrtbytesSinv, uint64(len(bytesAdd)))

	if verblvl > 1 {
		fn.LogCondMsg(true, fmt.Sprintf("blocks-Sinvappended:%d\n", atomic.LoadUint64(&totblkappSinv)))
		fn.LogCondMsg(true, fmt.Sprintf("trans--Sinvappended:%d\n", atomic.LoadUint64(&tottxappSinv)))
		fn.LogCondMsg(true, fmt.Sprintf("bytes--Sinvappended:%d\n", atomic.LoadUint64(&totwrtbytesSinv)))
		blkchainFileStat("closing")
	}

	return msg, excode
}
