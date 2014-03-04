package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

var (
	PORT   int = 8080
	graphs     = new(GraphList)
)

func main() {

	flag.IntVar(&PORT, "p", PORT, "server port")
	flag.Parse()

	fmt.Println(`GoGraph v0.1`)
	fmt.Println("Actions:\n - /get?tile=<graph title>&from=<Y.m.d H:i:s>&to=<optional Y.m.d H:i:s>\n - /push?title=<graph title>&object_id=<optional object id>&value=<value>&comment=<text comment>&meta=<json string with additional params>\n - /info?title=<graph title>\n\nRun on :8080")

	graphs.StartAutoSync()

	http.HandleFunc(`/push`, HandlePush)
	http.HandleFunc(`/get`, HandleGet)
	http.HandleFunc(`/info`, HandleInfo)

	http.ListenAndServe(`:`+strconv.Itoa(PORT), nil)
}

func HandlePush(w http.ResponseWriter, r *http.Request) {
	title := r.FormValue(`title`)
	meta := r.FormValue(`meta`)
	comment := r.FormValue(`comment`)
	value, _ := strconv.ParseFloat(r.FormValue(`value`), 64)
	objectId, _ := strconv.ParseInt(r.FormValue(`object_id`), 10, 64)

	if len(title) > 0 {
		graph := graphs.FindByTitle(title, true)
		dataRow := DataRow{Ts: time.Now().Round(time.Hour), Value: value, ObjectId: objectId, Amount: 1}
		graph.AddRow(&dataRow)

		if meta != `` {
			graph.AddMeta(meta, objectId)
		}

		if comment != `` {
			graph.AddComment(comment, objectId)
		}

		fmt.Println(`Add row`)
		//fmt.Printf("%+v\n", dataRow)
	} else {
		fmt.Println(`Ignore record with empty title`)
		fmt.Printf("%+v\n", r)

	}
}

func HandleGet(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, `Not implemented yet`)
}

func HandleInfo(w http.ResponseWriter, r *http.Request) {
	title := r.FormValue(`title`)

	response, _ := json.Marshal(graphs.FindByTitle(title, false))

	fmt.Fprintf(w, `%s`, response)
}
