package flux

import (
	"context"
	"fmt"
	"os"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

type Engine struct {
	runtime wazero.Runtime
	module  api.Module
	ptr     uint32 // FluxProcessorHandle pointer
}

func NewEngine(ctx context.Context, wasmPath string) (*Engine, error) {
	wasmBytes, err := os.ReadFile(wasmPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read wasm file: %w", err)
	}

	r := wazero.NewRuntime(ctx)

	// Instantiate WASI
	wasi_snapshot_preview1.MustInstantiate(ctx, r)

	// Compile and instantiate the module
	compiled, err := r.CompileModule(ctx, wasmBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to compile wasm module: %w", err)
	}

	mod, err := r.InstantiateModule(ctx, compiled, wazero.NewModuleConfig().WithStdout(os.Stdout).WithStderr(os.Stderr))
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate wasm module: %w", err)
	}

	// Create processor
	fn := mod.ExportedFunction("flux_processor_new")
	if fn == nil {
		return nil, fmt.Errorf("flux_processor_new not exported")
	}

	results, err := fn.Call(ctx, 14) // Default 14 days baseline
	if err != nil {
		return nil, fmt.Errorf("failed to create flux processor: %w", err)
	}

	return &Engine{
		runtime: r,
		module:  mod,
		ptr:     uint32(results[0]),
	}, nil
}

func (e *Engine) Close(ctx context.Context) error {
	if e.ptr != 0 {
		fn := e.module.ExportedFunction("flux_processor_free")
		if fn != nil {
			_, _ = fn.Call(ctx, uint64(e.ptr))
		}
	}
	return e.runtime.Close(ctx)
}

func (e *Engine) WhoopToHSI(ctx context.Context, json, timezone, deviceID string) (string, error) {
	return e.callTransform(ctx, "flux_processor_process_whoop", true, json, timezone, deviceID)
}

func (e *Engine) GarminToHSI(ctx context.Context, json, timezone, deviceID string) (string, error) {
	return e.callTransform(ctx, "flux_processor_process_garmin", true, json, timezone, deviceID)
}

func (e *Engine) callTransform(ctx context.Context, funcName string, stateful bool, json, timezone, deviceID string) (string, error) {
	// Allocate and copy strings to guest memory
	jsonPtr, jsonLen, err := e.writeString(ctx, json)
	if err != nil {
		return "", err
	}
	defer e.dealloc(ctx, jsonPtr, jsonLen)

	tzPtr, tzLen, err := e.writeString(ctx, timezone)
	if err != nil {
		return "", err
	}
	defer e.dealloc(ctx, tzPtr, tzLen)

	devPtr, devLen, err := e.writeString(ctx, deviceID)
	if err != nil {
		return "", err
	}
	defer e.dealloc(ctx, devPtr, devLen)

	// Call the function
	fn := e.module.ExportedFunction(funcName)
	if fn == nil {
		return "", fmt.Errorf("function %s not exported", funcName)
	}

	var results []uint64
	if stateful {
		results, err = fn.Call(ctx, uint64(e.ptr), uint64(jsonPtr), uint64(tzPtr), uint64(devPtr))
	} else {
		results, err = fn.Call(ctx, uint64(jsonPtr), uint64(tzPtr), uint64(devPtr))
	}
	if err != nil {
		return "", fmt.Errorf("failed to call %s: %w", funcName, err)
	}

	resPtr := uint32(results[0])
	if resPtr == 0 {
		return "", e.getLastError(ctx)
	}
	defer e.freeString(ctx, resPtr)

	return e.readString(resPtr)
}

func (e *Engine) writeString(ctx context.Context, s string) (uint32, uint32, error) {
	// Adding null terminator for C strings
	nullTerminated := s + "\x00"
	size := uint32(len(nullTerminated))

	fn := e.module.ExportedFunction("alloc")
	results, err := fn.Call(ctx, uint64(size))
	if err != nil {
		return 0, 0, fmt.Errorf("alloc failed: %w", err)
	}

	ptr := uint32(results[0])
	if !e.module.Memory().Write(ptr, []byte(nullTerminated)) {
		return 0, 0, fmt.Errorf("failed to write string to memory")
	}

	return ptr, size, nil
}

func (e *Engine) dealloc(ctx context.Context, ptr, size uint32) {
	fn := e.module.ExportedFunction("dealloc")
	_, _ = fn.Call(ctx, uint64(ptr), uint64(size))
}

func (e *Engine) freeString(ctx context.Context, ptr uint32) {
	fn := e.module.ExportedFunction("flux_free_string")
	_, _ = fn.Call(ctx, uint64(ptr))
}

func (e *Engine) getLastError(ctx context.Context) error {
	fn := e.module.ExportedFunction("flux_last_error")
	results, err := fn.Call(ctx)
	if err != nil {
		return fmt.Errorf("failed to get last error: %w", err)
	}

	ptr := uint32(results[0])
	if ptr == 0 {
		return fmt.Errorf("unknown error (no error message)")
	}

	s, err := e.readString(ptr)
	if err != nil {
		return fmt.Errorf("failed to read error message: %w", err)
	}
	return fmt.Errorf("flux error: %s", s)
}

func (e *Engine) readString(ptr uint32) (string, error) {
	mem := e.module.Memory()
	buf, ok := mem.Read(ptr, mem.Size()-ptr)
	if !ok {
		return "", fmt.Errorf("failed to read from memory at %d", ptr)
	}

	// Find null terminator
	for i, b := range buf {
		if b == 0 {
			return string(buf[:i]), nil
		}
	}

	return "", fmt.Errorf("string not null-terminated")
}
