package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/lib/pq"
	"time"
)

const TTL_TO_UPDATE = 10

type DataRow struct {
	DataId     int
	GraphId    int
	Ts         time.Time
	Value      float64
	Params     map[string]interface{}
	C1, C2, C3 float64
}

func (dr *DataRow) ParamsAsJSON() string {
	if len(dr.Params) == 0 {
		return `{}`
	}

	result, _ := json.Marshal(dr.Params)
	return string(result)
}

type Graph struct {
	GraphId   int
	Title     string
	AddedAt   time.Time
	UpdatedAt time.Time
	rows      []*DataRow
	IsChanged bool
}

func (g *Graph) AddRow(row *DataRow) {
	g.rows = append(g.rows, row)
	g.IsChanged = true

	fmt.Printf(`%+v`, row)
}

func (g *Graph) Dump(db *sql.DB) {

	var i int = 0
	var values []interface{}
	var sql string = `INSERT INTO data (graph_id, ts, value, params, c1, c2, c3) VALUES`

	for _, row := range g.rows {
		sql += fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d), ", i+1, i+2, i+3, i+4, i+5, i+6, i+7)
		values = append(values, row.GraphId, row.Ts, row.Value, row.ParamsAsJSON(), row.C1, row.C2, row.C3)
		i += 7
	}
	fmt.Println(sql)

	_, err := db.Query(sql[0:len(sql)-2], values...)

	if err != nil {
		panic(err)
	}
}

type GraphList struct {
	UpdatedAt          time.Time
	Graphs             []*Graph
	dbConn             *sql.DB
	isAutoReloaded     bool
	autoSaveTickerQuit chan bool
	autoSaveTicker     *time.Ticker
}

func (gl *GraphList) Db() *sql.DB {
	if gl.dbConn == nil {
		dbConn, err := sql.Open("postgres", "user=postgres dbname=gographs password=123 port=5433")

		if err != nil {
			fmt.Println("Can't connect to database:")
			panic(err)
		}

		gl.dbConn = dbConn
	}

	return gl.dbConn
}

func (gl *GraphList) StartAutoReload() {

	if gl.isAutoReloaded == false {

		gl.Reload()

		autoSaveTicker := time.NewTicker(10 * time.Second)
		autoSaveTickerQuit := make(chan bool)

		gl.autoSaveTicker = autoSaveTicker
		gl.autoSaveTickerQuit = autoSaveTickerQuit

		go func() {
			for {
				select {
				case <-gl.autoSaveTicker.C:
					gl.Reload()
				case <-gl.autoSaveTickerQuit:
					gl.autoSaveTicker.Stop()
					return
				}
			}
		}()
	}
}

func (gl *GraphList) StopAutoReload() {
	gl.isAutoReloaded = false
	gl.autoSaveTickerQuit <- true
}

func (gl *GraphList) Clear() {
	gl.Graphs = make([]*Graph, 0)
}

func (gl *GraphList) Add(graph *Graph) {
	gl.Graphs = append(gl.Graphs, graph)
}

func (gl *GraphList) FindByTitle(title string, autoCreate bool) *Graph {
	for _, value := range gl.Graphs {
		if value.Title == title {
			return value
		}
	}

	if autoCreate {
		return gl.Create(title)
	}

	return nil
}

func (gl *GraphList) Create(title string) *Graph {
	g := Graph{Title: title, AddedAt: time.Now(), UpdatedAt: time.Now(), IsChanged: true}

	rows, err := gl.Db().Query(`INSERT INTO graph (title, added_at, updated_at) VALUES($1, $2, $2) RETURNING graph_id`, title, g.AddedAt)

	if err != nil {
		panic(err)
	}

	for rows.Next() {
		rows.Scan(&g.GraphId)
	}

	if g.GraphId != 0 {
		g.IsChanged = false
	}

	gl.Add(&g)
	return &g
}

func (gl *GraphList) Reload() {
	gl.Save()
	gl.Clear()

	fmt.Println(`Reload`)

	rows, err := gl.Db().Query(`SELECT graph_id, title, added_at, updated_at FROM graph`)

	if err != nil {
		panic(err)
	}

	for rows.Next() {
		g := Graph{}

		if err := rows.Scan(&g.GraphId, &g.Title, &g.AddedAt, &g.UpdatedAt); err != nil {
			panic(err)
		}

		gl.Add(&g)
	}

	if err := rows.Err(); err != nil {
		panic(err)
	}

	gl.UpdatedAt = time.Now()
}

func (gl *GraphList) Save() {
	fmt.Println("Save")

	for _, graph := range gl.Graphs {
		if graph.IsChanged == true {
			graph.Dump(gl.Db())
		}
	}
}
