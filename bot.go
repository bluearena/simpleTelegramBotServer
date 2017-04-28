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

	"strconv"

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

type store struct {
	Location string
	Name     string
	Category string
	Shortcut string
}

var storeP = store{"Lonsdale, North Vancouver", "Persia Foods", "vegetable & fruit", "P"}
var storeTW = store{"North Vancouver", "Taiwan", "lunch", "TW"}
var storeSF = store{"North Vancouver", "Save on Foods", "food", "SF"}
var storeTT = store{"North Vancouver", "TT Supermarket", "food", "TT"}
var storeSP = store{"North Vancouver", "Shoppers", "tool", "SP"}
var storeWM = store{"North Vancouver", "Walmart", "food", "WM"}

var allStores = []store{storeP, storeTW, storeSF, storeTT, storeSP, storeWM}
var googleClient *sheets.Service
var spreadsheetID = os.Getenv("SPREADSHEET_ID")
var writeRange = "工作表1!A:E"
var readRange = "工作表1!E:E"
var botToken = os.Getenv("BOT_TOKEN")
var prefix = "https://api.telegram.org/bot" + botToken + "/"

func (s *store) record(price float64) {
	rb := &sheets.ValueRange{
		Values: [][]interface{}{{time.Now().Format("2006-01-02"), s.Location, s.Name, s.Category, price}},
	}
	_, err := googleClient.Spreadsheets.Values.Append(spreadsheetID, writeRange, rb).ValueInputOption("RAW").Do()
	if err != nil {
		log.Fatalf("Unable to retrieve data from sheet. %v", err)
	}
}

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

func getTotal() float64 {
	resp, err := googleClient.Spreadsheets.Values.Get(spreadsheetID, readRange).Do()
	if err != nil {
		log.Fatalf("Unable to retrieve data from sheet. %v", err)
	}
	sum := 0.0
	if len(resp.Values) > 0 {
		for _, row := range resp.Values {
			tmp, err := strconv.ParseFloat(row[0].(string), 64)
			if err != nil {
				log.Fatal(err)
			}
			sum += tmp
		}
	}
	return sum
}

func reply(replyMsg string) {
	finalURL := prefix + "sendMessage?chat_id=188909374&text=" + replyMsg
	http.Get(finalURL)
}

func getHelp() string {
	msg := ""
	for _, v := range allStores {
		msg += fmt.Sprintf("%v: %v\n", v.Shortcut, v.Name)
	}
	return url.PathEscape(msg)
}

func handler(w http.ResponseWriter, r *http.Request) {
	var u update
	price := 0.0
	res, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Fatal(err)
	}
	json.Unmarshal(res, &u)
	priceSlice := strings.Split(u.Message.Text, " ")
	if len(priceSlice) > 1 {
		price, err = strconv.ParseFloat(priceSlice[len(priceSlice)-1], 64)
	}
	if err != nil {
		log.Fatal(err)
	}
	replyMsg := "done"
	switch {
	case strings.HasPrefix(u.Message.Text, "P"):
		storeP.record(price)
	case strings.HasPrefix(u.Message.Text, "TW"):
		storeTW.record(price)
	case strings.HasPrefix(u.Message.Text, "SF"):
		storeSF.record(price)
	case strings.HasPrefix(u.Message.Text, "TT"):
		storeTT.record(price)
	case strings.HasPrefix(u.Message.Text, "SP"):
		storeSP.record(price)
	case strings.HasPrefix(u.Message.Text, "WM"):
		storeWM.record(price)
	case strings.ToLower(u.Message.Text) == "total":
		replyMsg = strconv.FormatFloat(getTotal(), 'f', 2, 64)
	case strings.ToLower(u.Message.Text) == "help":
		replyMsg = getHelp()
	default:
		replyMsg = "I don't understand"
	}
	reply(replyMsg)
	w.WriteHeader(http.StatusAccepted)
}

func main() {
	googleClient = initClient()
	http.Handle("/telegramBot", http.HandlerFunc(handler))
	err := http.ListenAndServe(":8001", nil)
	if err != nil {
		log.Fatal(err)
	}
}
