package main

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"time"
)

type DataRow struct {
	DataId     int
	GraphId    int
	Ts         time.Time
	Value      float64
	Params     map[string]string
	C1, C2, C3 float64
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

func (g *Graph) ToDump() string {
	return ""
}

type GraphList struct {
	UpdatedAt time.Time
	Graphs    []*Graph
	dbConn    *sql.DB
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
	gl.Clear()

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
	for _, graph := range gl.Graphs {
		if graph.IsChanged == true {
			gl.Db().Query(graph.ToDump())
		}
	}
}
