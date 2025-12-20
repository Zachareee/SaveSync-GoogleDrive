package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	// "strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

func getClient(config *oauth2.Config) *http.Client {
	tok := getTokenFromWeb(config)

	return config.Client(context.Background(), tok)
}

func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)

	fmt.Printf("%v", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web %v", err)
	}

	fmt.Println(tok)
	return tok
}

var data = []byte("{\"installed\":{\"client_id\":\"487698375903-j8s33ij1pc335jc2pu6d2rb1bgrg2fqo.apps.googleusercontent.com\",\"project_id\":\"savesync-450104\",\"auth_uri\":\"https://accounts.google.com/o/oauth2/auth\",\"token_uri\":\"https://oauth2.googleapis.com/token\",\"auth_provider_x509_cert_url\":\"https://www.googleapis.com/oauth2/v1/certs\",\"client_secret\":\"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\",\"redirect_uris\":[\"http://localhost\"]}}")

func main() {
	ctx := context.Background()

	config, err := google.ConfigFromJSON(data, drive.DriveFileScope)

	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)

	srv, err := drive.NewService(ctx, option.WithHTTPClient(client))

	if err != nil {
		log.Fatalf("Unable to retrieve Drive client: %v", err)
	}

	f, err := srv.Files.Create(&drive.File{Name: "SaveSync", MimeType: "application/vnd.google-apps.folder"}).Do()
	//.Media(strings.NewReader("Hello"))

	if err != nil {
		log.Fatalf("Unable to create folder: %v", err)
	}

	fmt.Println(f.Id)

	r, err := srv.Files.List().PageSize(10).Fields(googleapi.Field("files(id, name)")).Do()

	if err != nil {
		log.Fatalf("Unable to retrieve files: %v", err)
	}

	fmt.Println("Files:")
	if len(r.Files) == 0 {
		fmt.Println("No files found.")
	} else {
		for _, i := range r.Files {
			fmt.Printf("%s (%s)\n", i.Name, i.Id)
		}
	}
}
