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
const googleFolderMime = "application/vnd.google-apps.folder"

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

func createAuthCodeURL(redirectUri string) string {
	config := getConfig()
	config.RedirectURL = redirectUri
	return config.AuthCodeURL(rand.Text(), oauth2.AccessTypeOffline)
}

func getFileService(accessToken []byte) (*drive.FilesService, error) {
	srv, err := drive.NewService(CTX, option.WithHTTPClient(getClient(getConfig(), accessToken)))

	if err != nil {
		return nil, err
	}

	return srv.Files, nil
}

func homeFolder(fileService *drive.FilesService) (string, error) {
	files, err := fileService.List().
		Context(CTX).
		Fields("files(id)").
		Q(fmt.Sprintf("name = '%s' and mimeType = '%s' and trashed = false", savesyncFolderName, googleFolderMime)).
		Do()

	if err != nil {
		return "", err
	}

	if len(files.Files) != 0 {
		return files.Files[0].Id, nil
	}

	folder, err := fileService.
		Create(&drive.File{Name: savesyncFolderName, MimeType: googleFolderMime}).
		Context(CTX).
		Do()

	if err != nil {
		return "", err
	}

	return folder.Id, nil
}

func folderTemplate(foldername, folderId string) string {
	return fmt.Sprintf("name = '%s' and '%s' in parents and mimeType == '%s' and trashed = false", foldername, folderId, googleFolderMime)
}

func filenameTemplate(filename, folderId string) string {
	return fmt.Sprintf("name = '%s' and '%s' in parents and mimeType != '%s' and trashed = false", filename, folderId, googleFolderMime)
}

func findFolder(fileService *drive.FilesService, tag string) (string, error) {
	home, err := homeFolder(fileService)

	if err != nil {
		return "", err
	}

	folders, err := fileService.List().
		Context(CTX).
		Fields("files(id)").
		Q(folderTemplate(tag, home)).
		Do()

	if err != nil {
		return "", err
	}

	if len(folders.Files) != 0 {
		return folders.Files[0].Id, nil
	}

	createdFolder, err := fileService.Create(&File{
		Name:     tag,
		Parents:  []string{home},
		MimeType: googleFolderMime,
	}).
		Context(CTX).
		Do()

	return createdFolder.Id, err
}

func findFile(fileService *drive.FilesService, tag, filename string) (string, error) {
	tagfolder, err := findFolder(fileService, tag)

	if err != nil {
		return "", err
	}

	files, err := fileService.List().
		Context(CTX).
		Fields("files(id)").
		Q(filenameTemplate(filename, tagfolder)).
		Do()

	if err != nil {
		return "", err
	}

	if len(files.Files) == 0 {
		return "", fmt.Errorf("File not found: %s", filename)
	}

	return files.Files[0].Id, nil
}
