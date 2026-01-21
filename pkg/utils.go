package pkg

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/Zachareee/savesync_gdrive/internal"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

var CTX = context.Background()

const savesyncFolderName = "SaveSync"

type File = drive.File

func GetConfig() *oauth2.Config {
	config, err := google.ConfigFromJSON([]byte(internal.PublicKeys), drive.DriveFileScope)

	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	return config
}

func GetClient(config *oauth2.Config, data []byte) *http.Client {
	token, err := GetToken(data)

	if err != nil {
		log.Fatalf("Unable to parse token: %v", err)
	}

	return config.Client(CTX, &token)
}

func GetToken(data []byte) (oauth2.Token, error) {
	var authtoken oauth2.Token
	err := json.Unmarshal(data, &authtoken)

	return authtoken, err
}

func CreateAuthCodeURL(redirect_uri string) string {
	config := GetConfig()
	config.RedirectURL = redirect_uri
	return config.AuthCodeURL(rand.Text(), oauth2.AccessTypeOffline)
}

func GetFileService(access_token []byte) (*drive.FilesService, error) {
	srv, err := drive.NewService(CTX, option.WithHTTPClient(GetClient(GetConfig(), access_token)))

	if err != nil {
		return nil, err
	}

	return srv.Files, nil
}

func HomeFolder(fileService *drive.FilesService) (string, error) {
	files, err := fileService.List().
		Context(CTX).
		Fields("files(id)").
		Q(fmt.Sprintf("name = '%s'", savesyncFolderName)).
		Do()

	if err != nil {
		return "", err
	}

	if len(files.Files) != 0 {
		return files.Files[0].Id, nil
	}

	folder, err := fileService.
		Create(&drive.File{Name: savesyncFolderName, MimeType: "application/vnd.google-apps.folder"}).
		Context(CTX).
		Do()

	if err != nil {
		return "", err
	}

	return folder.Id, nil
}
