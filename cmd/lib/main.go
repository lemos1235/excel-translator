package main

/*
#include <stdlib.h>

// Define callback function types with void* user_data for context/reference passing
typedef void (*ProgressCallback)(char* phase, int done, int total, void* user_data);
typedef void (*ErrorCallback)(char* stage, char* error, void* user_data);

// Helper functions to call the function pointers from Go
static void call_progress(ProgressCallback cb, char* phase, int done, int total, void* user_data) {
    if (cb) cb(phase, done, total, user_data);
}

static void call_error(ErrorCallback cb, char* stage, char* error, void* user_data) {
    if (cb) cb(stage, error, user_data);
}
*/
import "C"
import (
	"context"
	"exceltranslator/pkg/config"
	"exceltranslator/pkg/runner"
	"sync"
	"unsafe"

	"github.com/pelletier/go-toml/v2"
)

var taskMap sync.Map // map[int64]context.CancelFunc

//export Translate
func Translate(
	taskID C.longlong,
	inputPath *C.char,
	outputPath *C.char,
	configToml *C.char,
	progressCB C.ProgressCallback,
	errorCB C.ErrorCallback,
	userData unsafe.Pointer,
) *C.char {
	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	id := int64(taskID)
	taskMap.Store(id, cancel)
	defer func() {
		taskMap.Delete(id)
		cancel()
	}()

	// Convert C strings to Go strings
	goInput := C.GoString(inputPath)
	goOutput := C.GoString(outputPath)
	goConfigToml := C.GoString(configToml)

	// Parse config
	var cfg config.AppConfig
	if err := toml.Unmarshal([]byte(goConfigToml), &cfg); err != nil {
		return C.CString("failed to parse config toml: " + err.Error())
	}

	// Map Go callbacks to C callbacks
	cb := runner.TranslationCallbacks{
		OnTranslated: func(original, translated string) {
			// Optional: Add OnTranslated callback if needed in the future
		},
		OnProgress: func(phase string, done, total int) {
			cPhase := C.CString(phase)
			defer C.free(unsafe.Pointer(cPhase))
			C.call_progress(progressCB, cPhase, C.int(done), C.int(total), userData)
		},
		OnError: func(stage string, err error) {
			cStage := C.CString(stage)
			cErr := C.CString(err.Error())
			defer C.free(unsafe.Pointer(cStage))
			defer C.free(unsafe.Pointer(cErr))
			C.call_error(errorCB, cStage, cErr, userData)
		},
		OnComplete: func(err error) {
			// Error handling is mostly covered by the return value or OnError
		},
	}

	err := runner.RunTranslationWithConfig(ctx, goInput, goOutput, &cfg, cb)
	if err != nil {
		// If cancelled, we might want to return a specific message or just the error
		return C.CString(err.Error())
	}

	return nil // Success
}

//export CancelTranslate
func CancelTranslate(taskID C.longlong) {
	if val, ok := taskMap.Load(int64(taskID)); ok {
		if cancel, ok := val.(context.CancelFunc); ok {
			cancel()
		}
	}
}

func main() {}
