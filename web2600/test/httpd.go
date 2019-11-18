package main

import (
	"fmt"
	"net/http"
)

func main() {
	fmt.Println("web26000 test server listening on localhost:8080")
	http.ListenAndServe(":8080", http.FileServer(http.Dir("www")))
}
