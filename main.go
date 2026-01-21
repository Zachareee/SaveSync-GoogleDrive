package main

import (
	"runtime"
	"unsafe"

	"github.com/Zachareee/savesync_gdrive/pkg"
)

/*
#include <stdlib.h>

struct Info {
	char *name;
	char *description;
	char *author;
	char *icon_url;
};

struct FileDetails {
	char *filename;
	unsigned long long dateModified;
};
*/
import "C"

type CStr = *C.char

var pinner runtime.Pinner

//export info
func info() C.struct_Info {
	plugin := pkg.Info()

	return C.struct_Info{
		C.CString(plugin.Name),
		C.CString(plugin.Description),
		C.CString(plugin.Author),
		C.CString(plugin.Icon_url),
	}
}

//export validate
func validate(credentials, redirect_uri CStr) (CStr, CStr) {
	url, err := pkg.Validate(C.GoString(credentials), C.GoString(redirect_uri))
	if err != nil {
		return C.CString(url), C.CString(err.Error())
	}

	return nil, nil
}

//export extract_credentials
func extract_credentials(uri CStr) (CStr, CStr) {
	credentials, err := pkg.Extract_credentials(C.GoString(uri))

	if err != nil {
		return nil, C.CString(err.Error())
	}

	return C.CString(credentials), nil
}

//export read_cloud
func read_cloud(accessToken CStr) (uint64, *C.struct_FileDetails, CStr) {
	details, err := pkg.Read_cloud(C.GoString(accessToken))

	if err != nil {
		return 0, nil, C.CString(err.Error())
	}

	n := uint64(len(details))

	fileDetailsSlice := make([]C.struct_FileDetails, n)

	for i, f := range details {
		fileDetailsSlice[i] = C.struct_FileDetails{
			filename:     C.CString(f.Filename),
			dateModified: C.ulonglong(f.DateModified),
		}
	}

	detailsPtr := unsafe.SliceData(fileDetailsSlice)
	pinner.Pin(detailsPtr)

	return uint64(len(details)), (*C.struct_FileDetails)(detailsPtr), nil
}

//export upload
func upload(access_token, filename CStr, date_modified uint64, data CStr, data_length uint64) CStr {
	byteslice := unsafe.Slice((*byte)(unsafe.Pointer(data)), data_length)
	err := pkg.Upload(C.GoString(access_token), C.GoString(filename), int64(date_modified), byteslice)

	if err != nil {
		return C.CString(err.Error())
	}

	return nil
}

//export download
func download(accessToken, filename CStr) (unsafe.Pointer, uint64, CStr) {
	data, err := pkg.Download(C.GoString(accessToken), C.GoString(filename))

	if err != nil {
		return nil, 0, C.CString(err.Error())
	}
	return C.CBytes(data), uint64(len(data)), nil
}

//export free_memory
func free_memory(str *C.char) {
	C.free(unsafe.Pointer(str))
}

//export close_dll
func close_dll() {
	pinner.Unpin()
}

func main() {}
