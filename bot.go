package main

import (
	"fmt"
	"net/http"
)

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, r.URL.Path[1:])
}

func main() {
	http.Handle("/telegramBot", http.HandlerFunc(handler))
	http.ListenAndServeTLS(":443", "/home/ec2-user/ssl/chained.pem", "/home/ec2-user/ssl/domain.key", nil)
}
