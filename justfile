just := env_var_or_default("JUST", "just")
just_dir := env_var_or_default("JUST_DIR", justfile_directory())
curl := env_var_or_default("CURL", "curl")
wasmtime := env_var_or_default("WASMTIME", "wasmtime")

go := env_var_or_default("GO_1_24_BIN", "go1.24.0")
wasm_tools := env_var_or_default("WASM_TOOLS", "wasm-tools")

go_output_wasm_path := env_var_or_default("GO_OUTPUT_WASM_PATH", "bin/go.raw.wasm")
go_output_wasm_embedded_path := env_var_or_default("GO_OUTPUT_WASM_EMBEDDED_PATH", "bin/go.embedded.wasm")
output_p2_wasm := env_var_or_default("OUTPUT_P2_WASM_PATH", "bin/go.wasm")

# NOTE: the reactor world doesn't work, tries to do too much
#wasi_p1_adapter_path := env_var_or_default("WASI_P1_ADAPTER_PATH", "wasi_snapshot_preview1.reactor.wasm")
wasi_p1_adapter_url := env_var_or_default("WASI_P1_ADAPTER_URL", "https://github.com/bytecodealliance/wasmtime/releases/download/v29.0.1/wasi_snapshot_preview1.reactor.wasm")

# We use a hacked adapter which has memory allocating functions in the call graph for
# wasi:cli/run no-oped (i.e. no args, no env vars)
wasi_p1_adapter_path := env_var_or_default("WASI_P1_ADAPTER_PATH", "p1.reactor.hacked.wasm")

@_default:
    {{just}} --list

# Retrieve the WASI P2 adapter
[group('setup')]
get-p2-adapter:
    if [ ! -f "{{wasi_p1_adapter_path}}" ]; then \
      {{curl}} -LO {{wasi_p1_adapter_url}}; \
    fi

[group('setup')]
bindgen:
    {{go}} run go.bytecodealliance.org/cmd/wit-bindgen-go generate --versioned -o ./generated wit

# Build the project
[group('build')]
build: build-go build-wasip2

# Build the go code
[group('build')]
build-go: bindgen
    GOOS=wasip1 GOARCH=wasm {{go}} build -buildmode=c-shared -o {{go_output_wasm_path}}

# Adapt an existing binary to WASI p2
[group('build')]
build-wasip2: get-p2-adapter
    {{wasm_tools}} component embed wit/ {{go_output_wasm_path}} -o {{go_output_wasm_embedded_path}}
    {{wasm_tools}} component new \
    -o {{output_p2_wasm}} \
    --adapt wasi_snapshot_preview1={{wasi_p1_adapter_path}} \
    {{go_output_wasm_embedded_path}}

# Run the code
[group('run')]
run:
    {{wasmtime}} -S cli {{output_p2_wasm}}
