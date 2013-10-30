package main

import (
	"fmt"
	"net/http"
)

var graphs = new(GraphList)

func main() {
	fmt.Println(`GoGraph v0.1`)
	fmt.Println("Actions:\n - /get\n - /push?data=<json string with params>\n - /info?title=<graph title>\n\nRun on :8080")

	graphs.StartAutoReload()

	http.HandleFunc(`/push`, HandlePush)
	http.HandleFunc(`/get`, HandleGet)
	http.HandleFunc(`/info`, HandleInfo)

	http.ListenAndServe(`:8080`, nil)
}
