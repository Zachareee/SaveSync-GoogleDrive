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
		C.CString(plugin.IconUrl),
	}
}

//export validate
func validate(credentials, redirectUri CStr) (CStr, CStr) {
	url, err := pkg.Validate(C.GoString(credentials), C.GoString(redirectUri))
	if err != nil {
		return C.CString(url), C.CString(err.Error())
	}

	return nil, nil
}

//export extract_credentials
func extract_credentials(uri CStr) (CStr, CStr) {
	credentials, err := pkg.ExtractCredentials(C.GoString(uri))

	if err != nil {
		return nil, C.CString(err.Error())
	}

	return C.CString(credentials), nil
}

//export read_cloud
func read_cloud(accessToken CStr) (*C.struct_FileDetails, uint64, CStr) {
	details, err := pkg.ReadCloud(C.GoString(accessToken))

	if err != nil {
		return nil, 0, C.CString(err.Error())
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

	return (*C.struct_FileDetails)(detailsPtr), uint64(len(details)), nil
}

//export upload
func upload(accessToken, filename CStr, dateModified uint64, data CStr, dataLength uint64) CStr {
	byteslice := unsafe.Slice((*byte)(unsafe.Pointer(data)), dataLength)
	err := pkg.Upload(C.GoString(accessToken), C.GoString(filename), int64(dateModified), byteslice)

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

//export remove
func remove(accessToken, filename CStr) CStr {
	err := pkg.Remove(C.GoString(accessToken), C.GoString(filename))

	if err != nil {
		return C.CString(err.Error())
	}
	return nil
}

//export free_info
func free_info(CStr, CStr, CStr, CStr) {}

//export free_string
func free_string(str CStr) {
	C.free(unsafe.Pointer(str))
}

//export free_file_details
func free_file_details(uint64, *C.struct_FileDetails) {
	pinner.Unpin()
}

func main() {}
