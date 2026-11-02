//go:build js

package main

import (
	"errors"
	"fmt"
	"syscall/js"
)

// httpDo performs a request with the browser's fetch. Using fetch directly
// instead of net/http keeps game.wasm considerably smaller.
func httpDo(method, url string, body []byte) ([]byte, error) {
	opts := js.Global().Get("Object").New()
	opts.Set("method", method)
	if body != nil {
		headers := js.Global().Get("Object").New()
		headers.Set("Content-Type", "application/json")
		opts.Set("headers", headers)
		opts.Set("body", string(body))
	}
	if as := js.Global().Get("AbortSignal"); as.Truthy() && as.Get("timeout").Truthy() {
		opts.Set("signal", as.Call("timeout", 8000))
	}

	type result struct {
		b   []byte
		err error
	}
	ch := make(chan result, 1)
	var onResp, onBuf, onErr js.Func
	release := func() { onResp.Release(); onBuf.Release(); onErr.Release() }
	status := 0
	onErr = js.FuncOf(func(_ js.Value, args []js.Value) any {
		msg := "fetch failed"
		if len(args) > 0 {
			msg = args[0].Call("toString").String()
		}
		ch <- result{err: errors.New(msg)}
		return nil
	})
	onBuf = js.FuncOf(func(_ js.Value, args []js.Value) any {
		u8 := js.Global().Get("Uint8Array").New(args[0])
		b := make([]byte, u8.Get("length").Int())
		js.CopyBytesToGo(b, u8)
		if status != 200 {
			ch <- result{err: fmt.Errorf("HTTP %d", status)}
		} else {
			ch <- result{b: b}
		}
		return nil
	})
	onResp = js.FuncOf(func(_ js.Value, args []js.Value) any {
		status = args[0].Get("status").Int()
		args[0].Call("arrayBuffer").Call("then", onBuf).Call("catch", onErr)
		return nil
	})
	js.Global().Call("fetch", url, opts).Call("then", onResp).Call("catch", onErr)
	r := <-ch
	release()
	return r.b, r.err
}
