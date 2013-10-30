package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

func HandlePush(w http.ResponseWriter, r *http.Request) {

	pushString := r.FormValue("data")
	var pushData map[string]interface{}

	json.Unmarshal([]byte(pushString), &pushData)

	if title, ok := pushData[`title`]; ok {
		graph := graphs.FindByTitle(title.(string), true)

		dataRow := DataRow{GraphId: graph.GraphId, Ts: time.Now()}

		if value, ok := pushData[`inc`]; ok {
			dataRow.Value = value.(float64)
		} else {
			dataRow.Value = 1
		}

		if c1, ok := pushData[`c1`]; ok {
			dataRow.C1 = c1.(float64)
		}

		if c2, ok := pushData[`c2`]; ok {
			dataRow.C2 = c2.(float64)
		}

		if c3, ok := pushData[`c3`]; ok {
			dataRow.C3 = c3.(float64)
		}

		if params, ok := pushData[`params`]; ok {
			dataRow.Params = params.(map[string]interface{})
		}

		pushData = nil

		graph.AddRow(&dataRow)
		fmt.Fprintf(w, `%d`, 1)
		return
	}

	fmt.Fprintf(w, `%d`, 0)
	return
}

func HandleGet(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, `Not implemented yet`)
}

func HandleInfo(w http.ResponseWriter, r *http.Request) {
	title := r.FormValue(`title`)

	response, _ := json.Marshal(graphs.FindByTitle(title, false))

	fmt.Fprintf(w, `%s`, response)
}
