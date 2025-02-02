// Copyright 2020 Thomas.Hoehenleitner [at] seerose.net
// Use of this source code is governed by a license that can be found in the LICENSE file.

// Package id List is responsible for id List managing
package id

// List management

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"strings"

	"github.com/rokath/trice/pkg/msg"
)

// NewLut returns a look-up map generated from JSON map file named fn.
func NewLut(w io.Writer, fn string) TriceIDLookUp {
	lu := make(TriceIDLookUp)
	if fn == "emptyFile" { // reserved name for tests only
		return lu
	}
	msg.FatalOnErr(lu.fromFile(fn))
	fmt.Fprintln(w, "Read ID List file", fn, "with", len(lu), "items.")
	if Verbose {
	}
	return lu
}

// NewLutLI returns a look-up map generated from JSON map file named fn.
func NewLutLI(w io.Writer, fn string) TriceIDLookUpLI {
	li := make(TriceIDLookUpLI)
	if fn == "emptyFile" { // reserved name for tests only
		return li
	}
	msg.FatalOnErr(li.fromFile(fn))
	fmt.Fprintln(w, "Read ID location information file", fn, "with", len(li), "items.")
	if Verbose {
	}
	return li
}

// newID() gets a new ID not used so far.
// The delivered id is usable as key for lu, but not added. So calling fn twice without adding to lu could give the same value back.
// It is important that lu was refreshed before with all sources to avoid finding as a new ID an ID which is already used in the source tree.
func (lu TriceIDLookUp) newID(w io.Writer, min, max TriceID, searchMethod string) TriceID {
	if Verbose {
		fmt.Fprintln(w, "IDMin=", min, "IDMax=", max, "IDMethod=", searchMethod)
	}
	switch searchMethod {
	case "random":
		return lu.newRandomID(w, min, max)
	case "upward":
		return lu.newUpwardID(min, max)
	case "downward":
		return lu.newDownwardID(min, max)
	}
	msg.Info(fmt.Sprint("ERROR:", searchMethod, "is unknown ID search method."))
	return 0
}

// newRandomID provides a random free ID inside interval [min,max].
// The delivered id is usable as key for lu, but not added. So calling fn twice without adding to lu could give the same value back.
func (lu TriceIDLookUp) newRandomID(w io.Writer, min, max TriceID) (id TriceID) {
	interval := int(max - min + 1)
	freeIDs := interval - len(lu)
	msg.FatalInfoOnFalse(freeIDs > 0, "no new ID possible, "+fmt.Sprint(min, max, len(lu)))
	wrnLimit := interval >> 2 // 25%
	msg.InfoOnTrue(freeIDs < wrnLimit, "WARNING: Less than 25% IDs free!")
	id = min + TriceID(rand.Intn(interval))
	if len(lu) == 0 {
		return
	}
	for {
	nextTry:
		for k := range lu {
			if id == k { // id used
				fmt.Fprintln(w, "ID", id, "used, next try...")
				id = min + TriceID(rand.Intn(interval))
				goto nextTry
			}
		}
		return
	}
}

// newUpwardID provides the smallest free ID inside interval [min,max].
// The delivered id is usable as key for lut, but not added. So calling fn twice without adding to lu gives the same value back.
func (lu TriceIDLookUp) newUpwardID(min, max TriceID) (id TriceID) {
	interval := int(max - min + 1)
	freeIDs := interval - len(lu)
	msg.FatalInfoOnFalse(freeIDs > 0, "no new ID possible: "+fmt.Sprint("min=", min, ", max=", max, ", used=", len(lu)))
	id = min
	if len(lu) == 0 {
		return
	}
	for {
	nextTry:
		for k := range lu {
			if id == k { // id used
				id++
				goto nextTry
			}
		}
		return
	}
}

// newDownwardID provides the biggest free ID inside interval [min,max].
// The delivered id is usable as key for lut, but not added. So calling fn twice without adding to lu gives the same value back.
func (lu TriceIDLookUp) newDownwardID(min, max TriceID) (id TriceID) {
	interval := int(max - min + 1)
	freeIDs := interval - len(lu)
	msg.FatalInfoOnFalse(freeIDs > 0, "no new ID possible: "+fmt.Sprint("min=", min, ", max=", max, ", used=", len(lu)))
	id = max
	if len(lu) == 0 {
		return
	}
	for {
	nextTry:
		for k := range lu {
			if id == k { // id used
				id--
				goto nextTry
			}
		}
		return
	}
}

// FromJSON converts JSON byte slice to lu.
func (lu TriceIDLookUp) FromJSON(b []byte) (err error) {
	if 0 < len(b) {
		err = json.Unmarshal(b, &lu)
	}
	return
}

// FromJSON converts JSON byte slice to li.
func (li TriceIDLookUpLI) FromJSON(b []byte) (err error) {
	if 0 < len(b) {
		err = json.Unmarshal(b, &li)
	}
	return
}

// fromFile reads file fn into lut. Existing keys are overwritten, lut is extended with new keys.
func (lu TriceIDLookUp) fromFile(fn string) error {
	b, err := ioutil.ReadFile(fn)
	s := fmt.Sprintf("fn=%s, maybe need to create an empty file first? (Safety feature)", fn)
	msg.FatalInfoOnErr(err, s)
	return lu.FromJSON(b)
}

// fromFile reads file fn into lut.
func (li TriceIDLookUpLI) fromFile(fn string) error {
	b, err := ioutil.ReadFile(fn)
	if err == nil { // file found
		return li.FromJSON(b)
	}
	if Verbose {
		fmt.Println("File ", fn, "not found, not showing location information")
	}
	return nil // silently ignore non existing file
}

// AddFmtCount adds inside lu to all trice type names without format specifier count the appropriate count.
// example change:
// `map[10000:{Trice8_2 hi %03u, %5x} 10001:{TRICE16 hi %03u, %5x}]
// `map[10000:{Trice8_2 hi %03u, %5x} 10001:{TRICE16_2 hi %03u, %5x}]
func (lu TriceIDLookUp) AddFmtCount(w io.Writer) {
	for i, x := range lu {
		if strings.ContainsAny(x.Type, "0_") {
			continue
		}
		n := formatSpecifierCount(x.Strg)
		x.Type = addFormatSpecifierCount(w, x.Type, n)
		lu[i] = x
	}
}

// toJSON converts lut into JSON byte slice in human-readable form.
func (lu TriceIDLookUp) toJSON() ([]byte, error) {
	return json.MarshalIndent(lu, "", "\t")
}

// toFile writes lut into file fn as indented JSON.
func (lu TriceIDLookUp) toFile(fn string) (err error) {
	var b []byte
	b, err = lu.toJSON()
	msg.FatalOnErr(err)
	var f *os.File
	f, err = os.Create(fn)
	msg.FatalOnErr(err)
	defer func() {
		err = f.Close()
	}()
	_, err = f.Write(b)
	return
}

// reverseS returns a reversed map. If different triceID's assigned to several equal TriceFmt all of the TriceID gets it into tflus.
func (lu TriceIDLookUp) reverseS() (tflus triceFmtLookUpS) {
	tflus = make(triceFmtLookUpS)
	for id, tF := range lu {
		addID(tF, id, tflus)
	}
	return
}

// addID adds tF and id to tflus. If tF already exists inside tflus, its id slice is extended with id.
func addID(tF TriceFmt, id TriceID, tflus triceFmtLookUpS) {
	tF.Type = strings.ToUpper(tF.Type) // no distinction for lower and upper case Type
	idSlice := tflus[tF]               // If the key doesn't exist, the first value will be the default zero value.
	idSlice = append(idSlice, id)
	tflus[tF] = idSlice
}

// toFile writes lut into file fn as indented JSON.
func (lim TriceIDLookUpLI) toFile(fn string) (err error) {
	f, err := os.Create(fn)
	msg.FatalOnErr(err)
	defer func() {
		err = f.Close()
		msg.FatalOnErr(err)
	}()

	b, err := lim.toJSON()
	msg.FatalOnErr(err)

	_, err = f.Write(b)
	msg.FatalOnErr(err)
	return
}

// toJSON converts lim into JSON byte slice in human-readable form.
func (lim TriceIDLookUpLI) toJSON() ([]byte, error) {
	return json.MarshalIndent(lim, "", "\t")
}
