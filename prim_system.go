// Copyright 2014 SteelSeries ApS.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This package implements a basic LISP interpretor for embedding in a go program for scripting.
// This file contains the system primitive functions.

package golisp

import (
	"fmt"
	"os"
	"strings"
	"time"
)

var symbolCounts map[string]int = make(map[string]int)

func RegisterSystemPrimitives() {
	MakePrimitiveFunction("load", 1, LoadFileImpl)
	MakePrimitiveFunction("sleep", 1, SleepImpl)
	MakePrimitiveFunction("millis", 0, MillisImpl)
	MakePrimitiveFunction("write-line", 1, WriteLineImpl)
	MakePrimitiveFunction("str", -1, MakeStringImpl)
	MakePrimitiveFunction("intern", 1, InternImpl)
	MakePrimitiveFunction("time", 1, TimeImpl)
	MakePrimitiveFunction("quit", 0, QuitImpl)
	MakePrimitiveFunction("gensym", -1, GensymImpl)
	MakePrimitiveFunction("eval", -1, EvalImpl)
	MakePrimitiveFunction("global-eval", 1, GlobalEvalImpl)
}

func LoadFileImpl(args *Data, env *SymbolTableFrame) (result *Data, err error) {
	filename, err := Eval(Car(args), env)
	if err != nil {
		return
	}
	if !StringP(filename) {
		err = ProcessError("Filename must be a string", env)
		return
	}

	return ProcessFile(StringValue(filename))
}

func QuitImpl(args *Data, env *SymbolTableFrame) (result *Data, err error) {
	WriteHistoryToFile(".golisp_history")
	fmt.Printf("\n\nGoodbye.\n\n")
	os.Exit(0)
	return
}

func SleepImpl(args *Data, env *SymbolTableFrame) (result *Data, err error) {
	n, err := Eval(Car(args), env)
	if err != nil {
		return
	}
	if !IntegerP(n) {
		err = ProcessError(fmt.Sprintf("Number expected, received %s", String(n)), env)
		return
	}
	millis := IntegerValue(n)
	time.Sleep(time.Duration(millis) * time.Millisecond)
	return
}

func MillisImpl(args *Data, env *SymbolTableFrame) (result *Data, err error) {
	result = IntegerWithValue(int64(time.Now().UnixNano() / 1e6))
	return
}

func WriteLineImpl(args *Data, env *SymbolTableFrame) (result *Data, err error) {
	data, err := Eval(Car(args), env)
	if err != nil {
		return
	}
	println(PrintString(data))
	return
}

func MakeStringImpl(args *Data, env *SymbolTableFrame) (result *Data, err error) {
	pieces := make([]string, 2)
	for cell := args; NotNilP(cell); cell = Cdr(cell) {
		sexpr := Car(cell)
		s, err := Eval(sexpr, env)
		if err != nil {
			break
		}
		pieces = append(pieces, PrintString(s))
	}
	return StringWithValue(strings.Join(pieces, "")), nil
}

func TimeImpl(args *Data, env *SymbolTableFrame) (result *Data, err error) {
	fmt.Printf("Starting timer.\n")
	startTime := time.Now()

	for cell := args; NotNilP(cell); cell = Cdr(cell) {
		sexpr := Car(cell)
		result, err = Eval(sexpr, env)
		if err != nil {
			break
		}
	}

	d := time.Since(startTime)
	fmt.Printf("Stopped timer.\nTook %v to run.\n", d)
	result = IntegerWithValue(int64(d.Nanoseconds() / 1000000))
	return
}

func InternImpl(args *Data, env *SymbolTableFrame) (result *Data, err error) {
	sym, err := Eval(Car(args), env)
	if err != nil {
		return
	}
	if !StringP(sym) {
		err = ProcessError(fmt.Sprintf("intern expects a string, but received %s.", String(sym)), env)
		return
	}

	return Intern(StringValue(sym)), nil
}

func GensymImpl(args *Data, env *SymbolTableFrame) (result *Data, err error) {
	var prefix string
	if Length(args) > 1 {
		err = ProcessError(fmt.Sprintf("gensym expects 0 or 1 argument, but received %d.", Length(args)), env)
		return
	}

	if Length(args) == 0 {
		prefix = "GENSYM"
	} else {
		arg, argerr := Eval(Car(args), env)
		if argerr != nil {
			err = argerr
			return
		}
		if !StringP(arg) && !SymbolP(arg) {
			err = ProcessError(fmt.Sprintf("gensym expects a string or symbol, but recieved %s.", String(arg)), env)
		}
		prefix = StringValue(arg)
	}

	count := symbolCounts[prefix]
	if count == 0 {
		count = 1
	} else {
		count += 1
	}
	symbolCounts[prefix] = count
	result = Intern(fmt.Sprintf("%s%d", prefix, count))
	return
}

func EvalImpl(args *Data, env *SymbolTableFrame) (result *Data, err error) {
	var evalEnv *SymbolTableFrame
	sexpr, err := Eval(Car(args), env)
	if err != nil {
		return
	}

	if Length(args) != 1 && Length(args) != 2 {
		err = ProcessError(fmt.Sprintf("eval expects 1 or 2 arguments, but recieved %d.", Length(args)), env)
		return
	}
	if Length(args) == 2 {
		ev, everr := Eval(Cadr(args), env)
		if everr != nil {
			return
		}
		if !EnvironmentP(ev) {
			err = ProcessError(fmt.Sprintf("eval expects an environment as it's second argument, but recieved %s.", String(Cadr(args))), env)
			return
		}
		evalEnv = EnvironmentValue(ev)
	} else {
		evalEnv = env
	}
	return Eval(sexpr, evalEnv)
}

func GlobalEvalImpl(args *Data, env *SymbolTableFrame) (result *Data, err error) {
	sexpr, err := Eval(Car(args), env)
	if err != nil {
		return
	}

	return Eval(sexpr, Global)
}
