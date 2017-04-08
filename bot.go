package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
)

type message struct {
	Text string
}

type update struct {
	Message message
}

func handler(w http.ResponseWriter, r *http.Request) {
	var u update
	res, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Fatal(err)
	}
	json.Unmarshal(res, &u)
	w.WriteHeader(http.StatusAccepted)
}

func main() {
	http.Handle("/telegramBot", http.HandlerFunc(handler))
	err := http.ListenAndServeTLS(":443", "/home/ec2-user/ssl/chained.pem", "/home/ec2-user/ssl/domain.key", nil)
	if err != nil {
		log.Fatal(err)
	}
}
