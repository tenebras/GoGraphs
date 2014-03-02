package main

import (
	"database/sql"
	/*"encoding/json"*/
	"fmt"
	_ "github.com/lib/pq"
	"time"
)

const (
	TTL_TO_UPDATE      = 30
	SYNC_BEFORE_VALUUM = 5
)

type Meta struct {
	Value     string
	ObjectId  int64
	Ts        time.Time
	isDeleted bool
}

type Comment struct {
	Value     string
	ObjectId  int64
	Ts        time.Time
	isDeleted bool
}

type DataRow struct {
	DataId    int
	GraphId   int
	Ts        time.Time
	Value     float64
	ObjectId  int64
	Amount    int
	isDeleted bool
}

type Graph struct {
	GraphId   int
	Title     string
	AddedAt   time.Time
	UpdatedAt time.Time
	rows      []*DataRow
	IsChanged bool
	Meta      []*Meta
	Comments  []*Comment
}

func (g *Graph) AddMeta(text string, objectId int64) {
	g.Meta = append(g.Meta, &Meta{Value: text, ObjectId: objectId, Ts: time.Now()})
}

func (g *Graph) AddComment(text string, objectId int64) {
	g.Comments = append(g.Comments, &Comment{Value: text, ObjectId: objectId, Ts: time.Now()})
}

func (g *Graph) AddRow(row *DataRow) {
	row.GraphId = g.GraphId
	var emptyIndex int = -1

	// Find deleted row and replace it with new value (to prevent memory allocation)
	for idx, value := range g.rows {
		if emptyIndex != -1 && value.isDeleted == true {
			emptyIndex = idx
		}

		if value.ObjectId == row.ObjectId && value.Ts == row.Ts {
			fmt.Println("Aggregate")
			g.rows[idx].Amount += 1
			g.rows[idx].Value += row.Value
			g.IsChanged = true

			return
		}
	}

	if emptyIndex != -1 {
		g.rows[emptyIndex] = row
	} else {
		// No empty records in slice, add new
		g.rows = append(g.rows, row)
	}
	g.IsChanged = true
}

func (g *Graph) Vacuum() {
	rows := make([]*DataRow, 0)
	fmt.Printf("Execute vacuum, records before %d\n", len(g.rows))
	for _, row := range g.rows {
		if row.isDeleted == false {
			rows = append(rows, row)
		}
	}

	g.rows = rows

	fmt.Printf("Records after: %d\n", len(g.rows))
}

func (g *Graph) Dump(stmt *sql.Stmt, execVacuum bool) error {

	for idx, row := range g.rows {
		if !row.isDeleted {
			_, err := stmt.Exec(row.GraphId, row.Ts, row.Value, row.ObjectId)

			if err != nil {
				return err
			} else {
				row.isDeleted = true
				fmt.Printf("Deleted: %+v\n", g.rows[idx])
			}
		}
	}

	g.IsChanged = false

	if execVacuum {
		g.Vacuum()
	}

	return nil
}

type GraphList struct {
	UpdatedAt          time.Time
	Graphs             []*Graph
	dbConn             *sql.DB
	isAutoReloaded     bool
	autoSaveTickerQuit chan bool
	autoSaveTicker     *time.Ticker
	syncCounter        int
}

func (gl *GraphList) Db() *sql.DB {
	if gl.dbConn == nil {
		dbConn, err := sql.Open("postgres", "user=postgres dbname=graphs password=123 port=5432")

		if err != nil {
			fmt.Println("Can't connect to database:")
			panic(err)
		}

		gl.dbConn = dbConn
	}

	return gl.dbConn
}

func (gl *GraphList) StartAutoSync() {

	if gl.isAutoReloaded == false {

		gl.Sync()

		autoSaveTicker := time.NewTicker(10 * time.Second)
		autoSaveTickerQuit := make(chan bool)

		gl.autoSaveTicker = autoSaveTicker
		gl.autoSaveTickerQuit = autoSaveTickerQuit

		go func() {
			for {
				select {
				case <-gl.autoSaveTicker.C:
					gl.Sync()
				case <-gl.autoSaveTickerQuit:
					gl.autoSaveTicker.Stop()
					return
				}
			}
		}()
	}
}

func (gl *GraphList) StopAutoSync() {
	gl.isAutoReloaded = false
	gl.autoSaveTickerQuit <- true
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

func (gl *GraphList) FindIndexByTitle(title string) int {
	for idx, value := range gl.Graphs {
		if value.Title == title {
			return idx
		}
	}

	return -1
}

func (gl *GraphList) Replace(idx int, graph *Graph) {
	gl.Graphs[idx] = graph
}

func (gl *GraphList) Merge(idx int, graph *Graph) {
	gl.Graphs[idx].Title = graph.Title
	gl.Graphs[idx].AddedAt = graph.AddedAt
	gl.Graphs[idx].UpdatedAt = graph.UpdatedAt
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

func (gl *GraphList) Sync() {
	gl.Save()

	fmt.Println(`Synchronize`)

	rows, err := gl.Db().Query(`SELECT graph_id, title, added_at, updated_at FROM graph`)

	if err != nil {
		panic(err)
	}

	for rows.Next() {
		g := Graph{}

		if err := rows.Scan(&g.GraphId, &g.Title, &g.AddedAt, &g.UpdatedAt); err != nil {
			panic(err)
		}

		if idx := gl.FindIndexByTitle(g.Title); idx == -1 {
			gl.Add(&g)
		} else {
			gl.Merge(idx, &g)
		}
	}

	if err := rows.Err(); err != nil {
		panic(err)
	}

	gl.UpdatedAt = time.Now()
}

func (gl *GraphList) Save() {
	fmt.Printf("Save %d\n", gl.syncCounter)
	vacuum := gl.syncCounter == SYNC_BEFORE_VALUUM

	if len(gl.Graphs) > 0 {

		tx, err := gl.Db().Begin()
		if err != nil {
			panic(err)
		}

		stmt, err := gl.Db().Prepare("INSERT INTO data (graph_id, ts, value, object_id) VALUES($1, $2, $3, $4)")

		if err != nil {
			panic(err)
		}

		for _, graph := range gl.Graphs {
			if graph.IsChanged == true {
				if err := graph.Dump(stmt, vacuum); err != nil {
					tx.Rollback()
					stmt.Close()
					panic(err)
				}
			}
		}

		tx.Commit()
	}

	if vacuum {
		gl.syncCounter = 0
	} else {
		gl.syncCounter += 1
	}
}
