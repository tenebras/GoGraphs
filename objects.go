package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/lib/pq"
	"strconv"
	"time"
)

const (
	TTL_TO_UPDATE      = 30
	SYNC_BEFORE_VACUUM = 5
	DSN                = `user=postgres dbname=graphs password=123 port=5432 sslmode=disable`
	DB_DRIVER          = `postgres`
	DATE_FORMAT        = `2006-01-02 15:04:05`
)

type DataInsertBundle struct {
	Data    *sql.Stmt
	Meta    *sql.Stmt
	Comment *sql.Stmt
	Graph   *sql.Stmt
}

func (dib *DataInsertBundle) Close() {
	dib.Meta.Close()
	dib.Data.Close()
	dib.Comment.Close()
}

func (dib *DataInsertBundle) PrepareAll(db *sql.DB) error {
	comment, errComment := db.Prepare(`INSERT INTO comment (graph_id, ts, value, object_id) VALUES($1, $2, $3, $4)`)
	data, errData := db.Prepare(`INSERT INTO data (graph_id, ts, value, object_id) VALUES($1, $2, $3, $4)`)
	meta, errMeta := db.Prepare(`INSERT INTO meta (graph_id, ts, value, object_id) VALUES($1, $2, $3, $4)`)
	graph, errGraph := db.Prepare(`UPDATE graph SET updated_at = $1 WHERE graph_id = $2`)

	dib.Comment = comment
	dib.Meta = meta
	dib.Data = data
	dib.Graph = graph

	if errData != nil {
		panic(errData)
	} else if errMeta != nil {
		panic(errMeta)
	} else if errComment != nil {
		panic(errComment)
	} else if errGraph != nil {
		panic(errGraph)
	}

	return nil
}

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

	for idx, value := range g.Meta {
		if value.isDeleted == true {
			g.Meta[idx] = &Meta{Value: text, ObjectId: objectId, Ts: time.Now()}

			return
		}
	}

	// No empty record, add new one
	g.Meta = append(g.Meta, &Meta{Value: text, ObjectId: objectId, Ts: time.Now().Round(time.Hour)})
}

func (g *Graph) AddComment(text string, objectId int64) {
	for idx, value := range g.Comments {
		if value.isDeleted == true {
			g.Comments[idx] = &Comment{Value: text, ObjectId: objectId, Ts: time.Now().Round(time.Hour)}

			return
		}
	}

	g.Comments = append(g.Comments, &Comment{Value: text, ObjectId: objectId, Ts: time.Now()})
}

func (g *Graph) AddRow(row *DataRow) {
	var emptyIndex int = -1

	// Find deleted row and replace it with new value (to prevent memory allocation)
	for idx, value := range g.rows {
		if emptyIndex != -1 && value.isDeleted == true {
			emptyIndex = idx
		}

		if value.ObjectId == row.ObjectId && value.Ts == row.Ts {
			fmt.Println(`Aggregate`)
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
	meta := make([]*Meta, 0)
	comments := make([]*Comment, 0)

	fmt.Printf("Vacuum: records before %d\n", len(g.rows)+len(g.Meta)+len(g.Comments))
	for _, row := range g.rows {
		if row.isDeleted == false {
			rows = append(rows, row)
		}
	}

	for _, row := range g.Meta {
		if row.isDeleted == false {
			meta = append(meta, row)
		}
	}

	for _, row := range g.Comments {
		if row.isDeleted == false {
			comments = append(comments, row)
		}
	}

	g.rows = rows
	g.Meta = meta
	g.Comments = comments

	fmt.Printf("Vacuum: records after %d\n", len(g.rows)+len(g.Meta)+len(g.Comments))
}

func (g *Graph) Store(bundle *DataInsertBundle, execVacuum bool) error {

	if err := g.storeData(bundle.Data); err != nil {
		if err := g.storeMeta(bundle.Meta); err != nil {
			if err := g.storeComments(bundle.Comment); err != nil {
				g.IsChanged = false
				g.UpdatedAt = time.Now()

				_, err := bundle.Graph.Exec(g.GraphId, g.UpdatedAt.Format(DATE_FORMAT))

				if err != nil {
					return err
				}

				if execVacuum {
					g.Vacuum()
				}
			} else {
				return err
			}
		} else {
			return err
		}
	} else {
		return err
	}

	return nil
}

// @todo Merge with already stored rows?
func (g *Graph) storeData(stmt *sql.Stmt) error {
	for _, row := range g.rows {
		if !row.isDeleted {
			_, err := stmt.Exec(g.GraphId, row.Ts.Format(DATE_FORMAT), row.Value, row.ObjectId, row.Amount)

			if err != nil {
				return err
			} else {
				row.isDeleted = true
			}
		}
	}

	return nil
}

func (g *Graph) storeMeta(stmt *sql.Stmt) error {
	for _, row := range g.Meta {
		if !row.isDeleted {
			_, err := stmt.Exec(g.GraphId, row.Ts.Format(DATE_FORMAT), row.Value, row.ObjectId)

			if err != nil {
				return err
			} else {
				row.isDeleted = true
			}
		}
	}

	return nil
}

func (g *Graph) storeComments(stmt *sql.Stmt) error {
	for _, row := range g.Comments {
		if !row.isDeleted {
			_, err := stmt.Exec(g.GraphId, row.Ts.Format(DATE_FORMAT), row.Value, row.ObjectId)

			if err != nil {
				return err
			} else {
				row.isDeleted = true
			}
		}
	}

	return nil
}

type GraphList struct {
	Graphs []*Graph
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

// @todo Find in db before create
func (gl *GraphList) Create(title string) *Graph {
	g := Graph{Title: title, AddedAt: time.Now(), UpdatedAt: time.Now(), IsChanged: true}

	rows, err := app.Db().Query(`INSERT INTO graph (title, added_at, updated_at) VALUES($1, $2, $2) RETURNING graph_id`, title, g.AddedAt)

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

// @todo remove graphs not in list
func (gl *GraphList) Sync(db *sql.DB) {
	gl.Save(db)
	fmt.Println(`Synchronize`)

	rows, err := db.Query(`SELECT graph_id, title, added_at, updated_at FROM graph`)

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
}

func (gl *GraphList) Save(db *sql.DB) {
	if len(gl.Graphs) > 0 {

		tx, err := db.Begin()
		if err != nil {
			panic(err)
		}

		var insertBundle *DataInsertBundle = &DataInsertBundle{}

		if err := insertBundle.PrepareAll(db); err != nil {
			panic(err)
		}

		for _, graph := range gl.Graphs {
			if graph.IsChanged == true {

				if err := graph.Store(insertBundle, app.IsVacuumTime); err != nil {
					tx.Rollback()
					insertBundle.Close()

					panic(err)
				}
			}
		}

		insertBundle.Close()
		tx.Commit()
	}
}

type App struct {
	Graphs             *GraphList
	Collections        *CollectionList
	IsVacuumTime       bool
	dbConn             *sql.DB
	isAutoReloaded     bool
	autoSaveTickerQuit chan bool
	autoSaveTicker     *time.Ticker
	syncCounter        int
}

func (app *App) Init() {
	app.Graphs = new(GraphList)
	app.Collections = new(CollectionList)

	app.StartAutoSync()
}

func (app *App) Sync() {
	app.IsVacuumTime = app.syncCounter == SYNC_BEFORE_VACUUM

	app.Graphs.Sync(app.Db())
	app.Collections.Sync(app.Db())

	if app.IsVacuumTime {
		app.syncCounter = 0
	} else {
		app.syncCounter++
	}
}

func (app *App) Db() *sql.DB {
	if app.dbConn == nil {
		dbConn, err := sql.Open(DB_DRIVER, DSN)

		if err != nil {
			fmt.Println(`Can't connect to database:`)
			panic(err)
		}

		app.dbConn = dbConn
	}

	return app.dbConn
}

func (app *App) StartAutoSync() {

	if app.isAutoReloaded == false {

		app.Sync()

		autoSaveTicker := time.NewTicker(10 * time.Second)
		autoSaveTickerQuit := make(chan bool)

		app.autoSaveTicker = autoSaveTicker
		app.autoSaveTickerQuit = autoSaveTickerQuit

		go func() {
			for {
				select {
				case <-app.autoSaveTicker.C:
					app.Sync()
				case <-app.autoSaveTickerQuit:
					app.autoSaveTicker.Stop()
					return
				}
			}
		}()
	}
}

func (app *App) StopAutoSync() {
	app.isAutoReloaded = false
	app.autoSaveTickerQuit <- true
}

type CollectionField struct {
	Name string
	Type string
	Size int
}

type Collection struct {
	CollectionId int64
	Title        string
	AddedAt      time.Time
	UpdatedAt    time.Time
	Fields       []*CollectionField
	ItemsCount   int64
	RawStructure string
}

func (c *Collection) DecodeFields() {
	err := json.Unmarshal([]byte(c.RawStructure), &c.Fields)

	if err != nil {
		panic(err)
	}
}

func (c *Collection) EncodeFields() string {

	var result string = `[`

	for _, field := range c.Fields {
		result += `{"name":"` + field.Name + `", "type":"` + field.Type + `", "size":` + strconv.Itoa(field.Size) + `},`
	}

	return result[0:len(result)-1] + `]`
}
func (c *Collection) ToJSON() string {
	return `{"title":"` + c.Title + `", "added_at":"` + c.AddedAt.Format(DATE_FORMAT) +
		`", "updated_at":"` + c.UpdatedAt.Format(DATE_FORMAT) +
		`", "fields":` + c.EncodeFields() + `}`
}

type CollectionList struct {
	Collections []*Collection
}

func (cl *CollectionList) Add(col *Collection) {
	idx := cl.FindIndexByTitle(col.Title)

	if idx == -1 {
		cl.Collections = append(cl.Collections, col)
	} else {
		cl.Collections[idx] = col
	}
}

func (cl *CollectionList) FindByTitle(title string) *Collection {
	for _, col := range cl.Collections {
		if col.Title == title {
			return col
		}
	}

	return nil
}

func (cl *CollectionList) FindIndexByTitle(title string) int {
	for idx, col := range cl.Collections {
		if col.Title == title {
			return idx
		}
	}

	return -1
}

func (cl *CollectionList) Sync(db *sql.DB) {
	rows, err := db.Query(`SELECT collection_id, title, added_at, updated_at, structure FROM collection`)

	if err != nil {
		panic(err)
	}

	for rows.Next() {
		col := new(Collection)

		if err := rows.Scan(&col.CollectionId, &col.Title, &col.AddedAt, &col.UpdatedAt, &col.RawStructure); err != nil {
			panic(err)
		}

		col.DecodeFields()

		cl.Add(col)
	}

	if err := rows.Err(); err != nil {
		panic(err)
	}
}

func (cl *CollectionList) ToJSON() string {
	var result string = `[`

	for _, col := range cl.Collections {
		result += col.ToJSON() + `,`
	}

	return result[0:len(result)-1] + `]`
}
