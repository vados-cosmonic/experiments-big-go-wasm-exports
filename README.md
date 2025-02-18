# Experiment: Big Go w/ WASM exports

Now that Go 1.24 has WASM exports via `//go:wasmexport`, this repo explores how much further we can get w/ building WASI P2 components.

## Results

We get a little further but not very far -- two huge caveats:

- We need a custom Go binary w/ changes that [fix a bug with duplicate imports](https://go-review.googlesource.com/c/go/+/629857)
- We need a hacked adapter that avoids trying to allocate (anything that triggers the go runtime will panic, trying to start a goroutine and allocate)
  - The relevant failure is [in the adapter, due to `cabi_realloc` not yet working from Go](https://github.com/bytecodealliance/wasmtime/blob/9afc64b4728d6e2067aa52331ff7b1d6f5275b5e/crates/wasi-preview1-component-adapter/src/lib.rs#L2745)

## Setup

You'll need two things for this build to "work":

- Custom build of `go` w/ the duplicate imports fixed
- Hacked adapter stored locally (there's already one here)

## What's wrong

As far as I can tell, a couple things (at least):

### Symptom: `State::magic1`, `State::magic2` are invalid, just after being set

- `cabi_realloc` on the Go side *seems* to work fine
  - memory is allocated, alignment looks right, if you allocate twice you get an address that is exactly the requested allocation size apart,
- The memory that gets returned and used by `*mut State` seems to be getting cleared/not properly set by the adapter, and this *only* triggers when using something that triggers the stdlib

Example stacktrace:

```
    0: error while executing at wasm backtrace:
           0: 0x18d41a - wit-component:adapter:wasi_snapshot_preview1!wasi_snapshot_preview1::macros::assert_fail::h4142ed9f78c145a2
           1: 0x18dab8 - wit-component:adapter:wasi_snapshot_preview1!args_sizes_get
           2: 0x190de2 - wit-component:shim!adapt-wasi_snapshot_preview1-args_sizes_get
           3: 0x68a6f - <unknown>!runtime.args_sizes_get
           4: 0x68ff5 - <unknown>!runtime.goenvs
           5: 0x772ca - <unknown>!runtime.schedinit
           6: 0xe779e - <unknown>!runtime.rt0_go
           7: 0xe7eed - <unknown>!_rt0_wasm_wasip1
           8: 0xe7ef9 - <unknown>!_rt0_wasm_wasip1_lib
    1: wasm trap: wasm `unreachable` instruction executed
```

Does something in Golang runtime setup clear out/zero out allocated memory, or are changes that were made in the adapter somehow not making it *right* after initialization?

Note that the same `assert`s placed after `State::new` do *not* trigger -- so somehow `State` is properly initialized (i.e. `state.magic1`, `state.magic2` are set) *before* returning, then once back in `State::with` they are *not* set.

### Symptom: `import allocator already set`
  - You can get here if you do the dangerous thing and disable the state magic checks
  - It *seems* that Go is calling imports while inside another import (or possibly at the same time as something else is trying?)
  - Trying to replace the `Cell` in a `Mutex` results in unsupported section error

Exmaple stacktrace:

```
unreachable executed at adapter line 2926: import allocator already set

Error: failed to run main module `bin/go.wasm`

Caused by:
    0: error while executing at wasm backtrace:
           0: 0x2537d8 - wit-component:adapter:wasi_snapshot_preview1!args_sizes_get
           1: 0x2575d3 - wit-component:shim!adapt-wasi_snapshot_preview1-args_sizes_get
           2: 0x7858c - <unknown>!runtime.args_sizes_get
           3: 0x78b12 - <unknown>!runtime.goenvs
           4: 0x870c3 - <unknown>!runtime.schedinit
           5: 0x100c41 - <unknown>!runtime.rt0_go
           6: 0x1038c0 - <unknown>!_rt0_wasm_wasip1
           7: 0x1038cc - <unknown>!_rt0_wasm_wasip1_lib
    1: wasm trap: wasm `unreachable` instruction executed
```

This *might* be fixable by increasing the number of import allocators available and naming them somehow to avoid collisions/someone taking an allocator that a specific operation was expecting to be present. It's not clear this *should* be how it works though.
