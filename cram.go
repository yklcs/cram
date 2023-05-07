// Package cram implements various image compression methods through WASM.
package cram

import (
	"context"
	_ "embed"
	"errors"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/emscripten"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

//go:embed codecs/mozjpeg/mozjpeg.wasm
var wasm []byte

// MozJPEG returns a JPEG-encoded byte slice compressing rgb.
func MozJPEG(rgb []byte, width int, height int) ([]byte, error) {
	ctx := context.Background()
	cfg := wazero.NewRuntimeConfigCompiler()
	r := wazero.NewRuntimeWithConfig(ctx, cfg)
	defer r.Close(ctx)

	emscripten.MustInstantiate(ctx, r)
	wasi_snapshot_preview1.MustInstantiate(ctx, r)

	mod, err := r.Instantiate(ctx, wasm)
	if err != nil {
		return nil, err
	}

	alloc := mod.ExportedFunction("allocate")
	encode := mod.ExportedFunction("encode")

	insize := len(rgb)

	res, err := alloc.Call(ctx, uint64(insize)) // should this be dealloced?
	if err != nil {
		return nil, err
	}
	inptr := res[0]

	ok := mod.Memory().Write(uint32(inptr), rgb)
	if !ok {
		return nil, errors.New("error writing memory")
	}

	res, err = alloc.Call(ctx, uint64(1)) // since mozjpeg manages its own memory, allocating 1 byte is fine
	if err != nil {
		return nil, err
	}
	outptr := res[0]

	res, err = encode.Call(ctx, inptr, uint64(width), uint64(height), 3, outptr)
	if err != nil {
		return nil, err
	}
	outsize := res[0]

	outimg, ok := mod.Memory().Read(uint32(outptr), uint32(outsize))
	if !ok {
		return nil, errors.New("error reading memory")
	}

	return outimg, nil
}