//go:build js

package main

import "syscall/js"

// apiBase returns the origin serving the game; the API lives on the same host.
func apiBase() string {
	return js.Global().Get("location").Get("origin").String()
}

func loadStore(key string) string {
	defer func() { recover() }() // storage may be blocked (private mode)
	v := js.Global().Get("localStorage").Call("getItem", key)
	if v.IsNull() || v.IsUndefined() {
		return ""
	}
	return v.String()
}

func saveStore(key, value string) {
	defer func() { recover() }()
	js.Global().Get("localStorage").Call("setItem", key, value)
}

// askName opens the HTML name dialog defined in game.html.
func askName(current string) {
	fn := js.Global().Get("openNameDialog")
	if fn.Type() != js.TypeFunction {
		nameAnswer, nameReady = current, true
		return
	}
	nameReady = false
	cb := js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) > 0 && args[0].Type() == js.TypeString {
			nameAnswer = args[0].String()
		} else {
			nameAnswer = ""
		}
		nameReady = true
		return nil
	})
	fn.Invoke(current, cb)
}

var (
	nameAnswer string
	nameReady  bool
)

// pollName returns the dialog result once. An empty name means cancelled.
func pollName() (string, bool) {
	if !nameReady {
		return "", false
	}
	nameReady = false
	return nameAnswer, true
}
