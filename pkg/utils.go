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

const savesyncFolderName = "SaveSync"

var CTX = context.Background()

type File = drive.File

func getConfig() *oauth2.Config {
	config, err := google.ConfigFromJSON([]byte(internal.PublicKeys), drive.DriveFileScope)

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

	return config.Client(CTX, &token)
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

func getFileService(access_token []byte) (*drive.FilesService, error) {
	srv, err := drive.NewService(CTX, option.WithHTTPClient(getClient(getConfig(), access_token)))

	if err != nil {
		return nil, err
	}

	return srv.Files, nil
}

func homeFolder(fileService *drive.FilesService) (string, error) {
	files, err := fileService.List().
		Context(CTX).
		Fields("files(id)").
		Q(fmt.Sprintf("name = '%s' and mimeType = 'application/vnd.google-apps.folder' and trashed = false", savesyncFolderName)).
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
