/// This file contains an example of a golang binary that is a WASM reactor module
/// using the newly supported wasmexport directive
///
///
/// Things *don't* work smoothly yet, and there are a bunch of reasons why:
///
/// - Duplicate imports produced by go
///   - https://github.com/golang/go/issues/60525
///   - https://github.com/bytecodealliance/wasm-tools/pull/1787#issuecomment-2349304603
///   - https://go-review.googlesource.com/c/go/+/629857
/// - Anything that invokes the go runtime fails (ex. `panic()`ing, memory allocation)
///   - Even basic stuff like writing to stdout will fail
///   - No way to disable big Go scheduler, obviously
///   - https://github.com/bytecodealliance/wasmtime/blob/9afc64b4728d6e2067aa52331ff7b1d6f5275b5e/crates/wasi-preview1-component-adapter/src/lib.rs#L2743
///
/// *BUT* if you run a custom Go and a custom adapter that has the allocation code
/// in the call graph of wasi:cli/run (args_sizes_get, environ_sizes_get), you can get
/// a wasi:cli/run call to work.
///
/// Getting Go's cabi_realloc impl to work with the adapter properly *should* unlock proper memory management
/// (https://github.com/bytecodealliance/go-modules/blob/4958c04b9d15de553f235954851732f73b5dcaee/x/cabi/realloc.go)
///
/// As for wasmCloud, progress should be possible once new wash pointing to the host w/ WASI forwards-compat changes,
/// as right now the host is trying to polyfill imports like `wasi:clocks/monotonic-clock@0.2.3#now`.

package main

import (
	"unsafe"

	//	"fmt"
	//	"os"

	tstdout "github.com/mrman/go-with-exports/generated/wasi/cli/v0.2.4/terminal-stdout"

	"go.bytecodealliance.org/cm"
)

//go:wasmexport examples:go/invoke-str#call
func invoke_str() uintptr {
	ptr, len := cm.LowerString("Hello from Go!")
	addr := uint64(uintptr(unsafe.Pointer(ptr)))
	combined := addr<<32 | uint64((uint32)(len))
	return uintptr(combined)
}

//go:wasmexport examples:go/invoke-num#call
func invoke_num() int32 {
	return 42
}

//go:wasmexport wasi:cli/run@0.2.4#run
func run() uint32 {
	// fmt.Fprintln(os.Stderr, "hello world")

	stdout := tstdout.GetTerminalStdout()
	if stdout.None() {
		// HACK: this won't actually error properly, but it *would* cause a distinct failure
		return 1
	}

	// NOTE: result<_,_> is repesented by `cm` as a bool that is turned into a u32
	//
	// You can change the below return to see things fail
	return 0
}

// This is a no-op main function, as this binary will be built into a
// WASM reactor module, which is not entered at _start (AKA `main()`)
//
//go:generate go run go.bytecodealliance.org/cmd/wit-bindgen-go generate --versioned -o ./generated ../wit
func main() {}
