package main

import (
    "embed"
    "fmt"
    "html/template"
    "log"
    "math/rand"
    "net/http"
    "sync"
    "time"
)

// Chúng ta nhúng toàn bộ thư mục templates để render
//go:embed templates
var templatesFS embed.FS

var tplIndex *template.Template
var tplBoard *template.Template

const (
    Rows    = 20
    Columns = 10
)

// Cấu trúc lưu trạng thái game Tetris
type TetrisGame struct {
    Board       [][]int // 0 = trống, >0 = có khối
    CurrentTet  *Tetromino
    Lock        sync.Mutex
}

// Cấu trúc một khối (Tetromino)
type Tetromino struct {
    Shape [][]int // Ma trận shape 4x4 (hoặc 3x3), 1 = có ô, 0 = trống
    Row   int     // Vị trí hàng hiện tại
    Col   int     // Vị trí cột hiện tại
}

// Tạo game mới
func NewTetrisGame() *TetrisGame {
    g := &TetrisGame{
        Board: make([][]int, Rows),
    }
    for i := range g.Board {
        g.Board[i] = make([]int, Columns)
    }
    g.SpawnTetromino()
    return g
}


var shapes = [][][]int{
    {
        {1, 1, 1, 1},
        {0, 0, 0, 0},
        {0, 0, 0, 0},
        {0, 0, 0, 0},
    },
    {
        {1, 1, 0, 0},
        {1, 1, 0, 0},
        {0, 0, 0, 0},
        {0, 0, 0, 0},
    },
    {
        {0, 1, 1, 0},
        {1, 1, 0, 0},
        {0, 0, 0, 0},
        {0, 0, 0, 0},
    },
    {
        {1, 1, 0, 0},
        {0, 1, 1, 0},
        {0, 0, 0, 0},
        {0, 0, 0, 0},
    },
    {
        {1, 1, 1, 0},
        {0, 1, 0, 0},
        {0, 0, 0, 0},
        {0, 0, 0, 0},
    },
}

func newRandomTetromino() *Tetromino {
    r := rand.Intn(len(shapes))
    shape := shapes[r]
    return &Tetromino{
        Shape: shape,
        Row:   0,
        Col:   Columns/2 - 2, 
    }
}


func (g *TetrisGame) SpawnTetromino() {
    g.CurrentTet = newRandomTetromino()
}


func (g *TetrisGame) getBoardForRender() [][]int {
    temp := make([][]int, Rows)
    for i := range g.Board {
        rowCopy := make([]int, Columns)
        copy(rowCopy, g.Board[i])
        temp[i] = rowCopy
    }

    for r := 0; r < 4; r++ {
        for c := 0; c < 4; c++ {
            if g.CurrentTet.Shape[r][c] == 1 {
                rr := g.CurrentTet.Row + r
                cc := g.CurrentTet.Col + c
                if rr >= 0 && rr < Rows && cc >= 0 && cc < Columns {
                    temp[rr][cc] = 2
                }
            }
        }
    }

    return temp
}

func (g *TetrisGame) MoveDown() {
    if !g.canMove(g.CurrentTet.Row+1, g.CurrentTet.Col) {
        g.lockShape()
        g.SpawnTetromino()
    } else {
        g.CurrentTet.Row++
    }
}


func (g *TetrisGame) MoveLeft() {
    if g.canMove(g.CurrentTet.Row, g.CurrentTet.Col-1) {
        g.CurrentTet.Col--
    }
}


func (g *TetrisGame) MoveRight() {
    if g.canMove(g.CurrentTet.Row, g.CurrentTet.Col+1) {
        g.CurrentTet.Col++
    }
}


func (g *TetrisGame) lockShape() {
    for r := 0; r < 4; r++ {
        for c := 0; c < 4; c++ {
            if g.CurrentTet.Shape[r][c] == 1 {
                rr := g.CurrentTet.Row + r
                cc := g.CurrentTet.Col + c
                if rr >= 0 && rr < Rows && cc >= 0 && cc < Columns {
                    g.Board[rr][cc] = 1
                }
            }
        }
    }
}


func (g *TetrisGame) canMove(newRow, newCol int) bool {
    for r := 0; r < 4; r++ {
        for c := 0; c < 4; c++ {
            if g.CurrentTet.Shape[r][c] == 1 {
                rr := newRow + r
                cc := newCol + c
                if rr < 0 || rr >= Rows || cc < 0 || cc >= Columns {
                    return false
                }
                if g.Board[rr][cc] == 1 {
                    return false
                }
            }
        }
    }
    return true
}


func (g *TetrisGame) Rotate() {
    newShape := make([][]int, 4)
    for i := range newShape {
        newShape[i] = make([]int, 4)
    }
    for r := 0; r < 4; r++ {
        for c := 0; c < 4; c++ {
            newShape[c][4-r-1] = g.CurrentTet.Shape[r][c]
        }
    }

    oldShape := g.CurrentTet.Shape
    g.CurrentTet.Shape = newShape
    if !g.canMove(g.CurrentTet.Row, g.CurrentTet.Col) {
        g.CurrentTet.Shape = oldShape
    }
}


var game = NewTetrisGame()

func main() {
    rand.Seed(time.Now().UnixNano())


    tplIndex = template.Must(template.ParseFS(templatesFS, "templates/index.html"))
    tplBoard = template.Must(template.ParseFS(templatesFS, "templates/board.html"))


    http.HandleFunc("/", handleIndex)
    http.HandleFunc("/move", handleMove)

    http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

    fmt.Println("Server chạy tại http://localhost:8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}

// Render trang index
func handleIndex(w http.ResponseWriter, r *http.Request) {
    game.Lock.Lock()
    defer game.Lock.Unlock()

    if err := tplIndex.Execute(w, nil); err != nil {
        log.Println("render index error:", err)
    }
}


func handleMove(w http.ResponseWriter, r *http.Request) {
    game.Lock.Lock()
    defer game.Lock.Unlock()

    switch r.FormValue("action") {
    case "left":
        game.MoveLeft()
    case "right":
        game.MoveRight()
    case "down":
        game.MoveDown()
    case "rotate":
        game.Rotate()
    }


    boardForRender := game.getBoardForRender()
    if err := tplBoard.Execute(w, boardForRender); err != nil {
        log.Println("render board error:", err)
    }
}
