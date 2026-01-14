package main

/*
#include <stdlib.h>

struct Info {
	char *name;
	char *description;
	char *author;
	char *icon_url;
};

struct FileDetails {
	char *tag;
	char *filename;
	unsigned long long timestamp;
};
*/
import "C"

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"net/url"
	"time"
	"unsafe"

	"fmt"
	"log"

	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
	// "google.golang.org/api/googleapi"
	// "google.golang.org/api/option"
)

type CStr = *C.char

var ctx = context.Background()

const data = `{"installed":{"client_id":"487698375903-j8s33ij1pc335jc2pu6d2rb1bgrg2fqo.apps.googleusercontent.com","project_id":"savesync-450104","auth_uri":"https://accounts.google.com/o/oauth2/auth","token_uri":"https://oauth2.googleapis.com/token","auth_provider_x509_cert_url":"https://www.googleapis.com/oauth2/v1/certs","client_secret":"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","redirect_uris":["http://localhost"]}}`

func getConfig() *oauth2.Config {
	config, err := google.ConfigFromJSON([]byte(data), drive.DriveFileScope)

	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	return config
}

func getClient(config *oauth2.Config, data []byte) *http.Client {
	token, err := getToken(data)

	if err != nil {
		log.Fatalf("Unable to parse token: %v", err)
	}

	return config.Client(ctx, &token)
}

func getToken(data []byte) (oauth2.Token, error) {
	var authtoken oauth2.Token
	err := json.Unmarshal(data, &authtoken)

	return authtoken, err
}

func createAuthCodeURL(redirect_uri string) string {
	config := getConfig()
	config.RedirectURL = redirect_uri
	return config.AuthCodeURL(rand.Text(), oauth2.AccessTypeOffline)
}

//export info
func info() C.struct_Info {
	return C.struct_Info{
		name:        C.CString("Google Drive"),
		description: C.CString("Google Drive plugin for SaveSync"),
		author:      C.CString("Zachareee"),
		icon_url:    C.CString("https://upload.wikimedia.org/wikipedia/commons/1/12/Google_Drive_icon_%282020%29.svg"),
	}
}

//export validate
func validate(credentials, redirect_uri CStr) (CStr, CStr) {
	url := C.CString(createAuthCodeURL(C.GoString(redirect_uri)))
	{
		credentials := C.GoString(credentials)
		if credentials == "" {
			return url, C.CString("No credentials provided")
		}

		token, err := getToken([]byte(credentials))
		switch {
		case err != nil:
			return url, C.CString(err.Error())
		case !token.Valid():
			return url, C.CString("Token is invalid, please reauthenticate")
		}
	}

	free_memory(url)
	return nil, nil
}

//export extract_credentials
func extract_credentials(uri CStr) (CStr, CStr) {
	s, err := url.Parse(C.GoString(uri))
	if err != nil {
		return nil, C.CString(fmt.Sprintf("Could not parse url: %+v", err))
	}

	queries := s.Query()

	if errcode := queries.Get("error"); errcode != "" {
		return nil, C.CString(fmt.Sprintf("Error while exchanging authorization code: %+v", errcode))
	}

	auth, err := getConfig().Exchange(context.TODO(), queries.Get("code"))
	if err != nil {
		return nil, C.CString(fmt.Sprintf("Unable to retrieve token from web %v", err))
	}

	data, err := json.Marshal(auth)

	if err != nil {
		fmt.Printf("Unable to convert token to json %v\n", err)
	}

	return C.CString(string(data)), nil
}

func getFileService(access_token []byte) (*drive.FilesService, error) {
	srv, err := drive.NewService(ctx, option.WithHTTPClient(getClient(getConfig(), access_token)))

	if err != nil {
		return nil, err
	}

	return srv.Files, nil
}

//export read_cloud
func read_cloud() (uint64, *C.struct_FileDetails) {
	return 0, nil
}

//export upload
func upload(access_token, tag, filename CStr, date_modified uint64, data CStr, data_length uint64) CStr {
	files, err := getFileService([]byte(C.GoString(access_token)))

	if err != nil {
		return C.CString(err.Error())
	}

	tagstring := C.GoString(tag)
	savesync_folder, err := files.Create(&drive.File{Name: "SaveSync", MimeType: "application/vnd.google-apps.folder"}).Context(ctx).Do()

	if err != nil {
		return C.CString(err.Error())
	}

	folder, err := files.Create(&drive.File{Name: tagstring, MimeType: "application/vnd.google-apps.folder", Parents: []string{savesync_folder.Id}}).Context(ctx).Do()

	if err != nil {
		return C.CString(err.Error())
	}

	byteslice := unsafe.Slice((*byte)(unsafe.Pointer(data)), data_length)

	_, err = files.
		Create(&drive.File{
			Name:         C.GoString(filename),
			ModifiedTime: time.Unix(int64(date_modified), 0).Format(time.RFC3339),
			Parents:      []string{folder.Id},
		}).
		Context(ctx).
		Media(bytes.NewReader(byteslice)).
		Do()

	if err != nil {
		return C.CString(err.Error())
	}
	return nil
}

//export download
func download(tag, filename CStr) {}

//export free_memory
func free_memory(str *C.char) {
	C.free(unsafe.Pointer(str))
}

//	func main() {
//		client := getClient(getConfig())
//
//		srv, err := drive.NewService(ctx, option.WithHTTPClient(client))
//
//		if err != nil {
//			log.Fatalf("Unable to retrieve Drive client: %v", err)
//		}
//
//		f, err := srv.Files.Create(&drive.File{Name: "SaveSync", MimeType: "application/vnd.google-apps.folder"}).Do()
//		//.Media(strings.NewReader("Hello"))
//
//		if err != nil {
//			log.Fatalf("Unable to create folder: %v", err)
//		}
//
//		fmt.Println(f.Id)
//
//		r, err := srv.Files.List().PageSize(10).Fields(googleapi.Field("files(id, name)")).Do()
//
//		if err != nil {
//			log.Fatalf("Unable to retrieve files: %v", err)
//		}
//
//		fmt.Println("Files:")
//		if len(r.Files) == 0 {
//			fmt.Println("No files found.")
//		} else {
//			for _, i := range r.Files {
//				fmt.Printf("%s (%s)\n", i.Name, i.Id)
//			}
//		}
//	}
func main() {}
