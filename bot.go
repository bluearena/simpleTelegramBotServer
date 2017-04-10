package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	sheets "google.golang.org/api/sheets/v4"
)

type message struct {
	Text string
}

type update struct {
	Message message
}

var googleClient *sheets.Service
var spreadsheetID = os.Getenv("SPREADSHEET_ID")
var readRange = "工作表1!A:E"
var botToken = os.Getenv("BOT_TOKEN")
var prefix = "https://api.telegram.org/bot" + botToken + "/"

func getClient(ctx context.Context, config *oauth2.Config) *http.Client {
	cacheFile, err := tokenCacheFile()
	if err != nil {
		log.Fatalf("Unable to get path to cached credential file. %v", err)
	}
	tok, err := tokenFromFile(cacheFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(cacheFile, tok)
	}
	return config.Client(ctx, tok)
}

func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var code string
	if _, err := fmt.Scan(&code); err != nil {
		log.Fatalf("Unable to read authorization code %v", err)
	}

	tok, err := config.Exchange(oauth2.NoContext, code)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web %v", err)
	}
	return tok
}

func tokenCacheFile() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	tokenCacheDir := filepath.Join(usr.HomeDir, ".credentials")
	os.MkdirAll(tokenCacheDir, 0700)
	return filepath.Join(tokenCacheDir,
		url.QueryEscape("sheets.googleapis.com-go-quickstart.json")), err
}

func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	t := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(t)
	defer f.Close()
	return t, err
}

func saveToken(file string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", file)
	f, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func initClient() *sheets.Service {
	ctx := context.Background()

	b, err := ioutil.ReadFile("client_secret.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved credentials
	// at ~/.credentials/sheets.googleapis.com-go-quickstart.json
	config, err := google.ConfigFromJSON(b, "https://www.googleapis.com/auth/spreadsheets")
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(ctx, config)

	srv, err := sheets.New(client)
	if err != nil {
		log.Fatalf("Unable to retrieve Sheets Client %v", err)
	}
	return srv
}

func record(recordData [][]interface{}) {
	rb := &sheets.ValueRange{
		Values: recordData,
	}
	_, err := googleClient.Spreadsheets.Values.Append(spreadsheetID, readRange, rb).ValueInputOption("RAW").Do()
	if err != nil {
		log.Fatalf("Unable to retrieve data from sheet. %v", err)
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	var u update
	res, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Fatal(err)
	}
	json.Unmarshal(res, &u)
	priceSlice := strings.Split(u.Message.Text, " ")
	price := priceSlice[len(priceSlice)-1]
	switch {
	case strings.HasPrefix(u.Message.Text, "P"):
		record([][]interface{}{{time.Now().Format("2006-01-02"), "Lonsdale, North Vancouver", "Persia Foods", "vegetable & fruit", price}})
	case strings.HasPrefix(u.Message.Text, "TW"):
		record([][]interface{}{{time.Now().Format("2006-01-02"), "North Vancouver", "Taiwan", "lunch", price}})
	case strings.HasPrefix(u.Message.Text, "SF"):
		record([][]interface{}{{time.Now().Format("2006-01-02"), "North Vancouver", "Save on Foods", "food", price}})
	case strings.HasPrefix(u.Message.Text, "TT"):
		record([][]interface{}{{time.Now().Format("2006-01-02"), "North Vancouver", "T&T Supermarket", "food", price}})
	case strings.HasPrefix(u.Message.Text, "SP"):
		record([][]interface{}{{time.Now().Format("2006-01-02"), "North Vancouver", "Shoppers", "food", price}})
	default:
		finalURL := prefix + "sendMessage?chat_id=188909374&text=I don't understand"
		http.Get(finalURL)
	}
	w.WriteHeader(http.StatusAccepted)
}

func main() {
	googleClient = initClient()
	http.Handle("/telegramBot", http.HandlerFunc(handler))
	err := http.ListenAndServeTLS(":443", "/home/ec2-user/ssl/chained.pem", "/home/ec2-user/ssl/domain.key", nil)
	if err != nil {
		log.Fatal(err)
	}
}
