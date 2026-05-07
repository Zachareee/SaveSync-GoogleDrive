package pkg

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"time"
)

type PluginInfo struct {
	Name, Description, Author, IconUrl string
}

type FileDetails struct {
	Tag          string
	Filename     string
	DateModified uint64
	Data         string
}

func Info() PluginInfo {
	return PluginInfo{
		Name:        "Google Drive",
		Description: "Google Drive plugin for SaveSync",
		Author:      "Zachareee",
		IconUrl:     "https://upload.wikimedia.org/wikipedia/commons/1/12/Google_Drive_icon_%282020%29.svg",
	}
}

func Validate(credentials, redirectUri string) (string, error) {
	url := createAuthCodeURL(redirectUri)
	if credentials == "" {
		return url, errors.New("No credentials provided")
	}

	token, err := getToken([]byte(credentials))
	switch {
	case err != nil:
		return url, err
	case !token.Valid():
		return url, errors.New("Token expired, please reauthenticate")
	}

	return "", nil
}

func ExtractCredentials(uri string) (string, error) {
	s, err := url.Parse(uri)
	if err != nil {
		return "", err
	}

	queries := s.Query()

	if errcode := queries.Get("error"); errcode != "" {
		return "", errors.New(errcode)
	}

	auth, err := getConfig().Exchange(context.TODO(), queries.Get("code"))
	if err != nil {
		return "", err
	}

	data, err := json.Marshal(auth)

	if err != nil {
		return "", err
	}

	return string(data), nil
}

func ReadCloud(accessToken string) ([]FileDetails, error) {
	fileService, err := getFileService([]byte(accessToken))

	if err != nil {
		return nil, err
	}

	homeFolderId, err := homeFolder(fileService)

	if err != nil {
		return nil, err
	}

	tagFolders, err := fileService.
		List().
		Context(CTX).
		Fields("files(id, name)").
		Q(fmt.Sprintf("'%s' in parents and trashed = false", homeFolderId)).
		Do()

	if err != nil {
		return nil, err
	}

	fileDetailsSlice := []FileDetails{}

	for _, folder := range tagFolders.Files {
		nestedFiles, err := fileService.
			List().
			Context(CTX).
			Fields("files(name, modifiedTime)").
			Q(fmt.Sprintf("'%s' in parents and trashed = false", folder.Id)).
			Do()

		if err != nil {
			return nil, err
		}

		for _, f := range nestedFiles.Files {
			modifiedDate, err := time.Parse(time.RFC3339, f.ModifiedTime)
			if err != nil {
				return nil, err
			}

			fileDetailsSlice = append(fileDetailsSlice, FileDetails{
				Tag:          folder.Name,
				Filename:     f.Name,
				DateModified: uint64(modifiedDate.Unix()),
			})
		}
	}

	return fileDetailsSlice, nil
}

func Upload(accessToken, tag, filename string, dateModified int64, data []byte) error {
	fileService, err := getFileService([]byte(accessToken))

	if err != nil {
		return err
	}

	tagFolderId, err := findFolder(fileService, tag)

	if err != nil {
		return err
	}

	files, err := fileService.List().
		Context(CTX).
		Fields("files(id)").
		Q(filenameTemplate(filename, tagFolderId)).
		Do()

	if err != nil {
		return err
	}

	var modifiedTime = time.Unix(int64(dateModified), 0).Format(time.RFC3339)
	var reader = bytes.NewReader(data)

	if len(files.Files) != 0 {
		_, err := fileService.
			Update(files.Files[0].Id, &File{ModifiedTime: modifiedTime}).
			Context(CTX).
			Media(reader).
			Do()

		return err
	}

	_, err = fileService.
		Create(&File{
			Name:         filename,
			ModifiedTime: modifiedTime,
			Parents:      []string{tagFolderId},
		}).
		Context(CTX).
		Media(reader).
		Do()

	return err
}

func Download(accessToken, tag, filename string) ([]byte, error) {
	fileService, err := getFileService([]byte(accessToken))

	if err != nil {
		return nil, err
	}

	file, err := findFile(fileService, tag, filename)

	if err != nil {
		return nil, err
	}

	resp, err := fileService.Get(file).Download()

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

func Remove(accessToken, tag, filename string) error {
	fileService, err := getFileService([]byte(accessToken))

	id, err := findFile(fileService, tag, filename)

	if err != nil {
		return err
	}

	return fileService.Delete(id).
		Context(CTX).
		Do()
}
