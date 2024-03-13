package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	_ "github.com/mattn/go-sqlite3"
)

var (
	DbName  = ".word.sqlite" // in $HOME directory
	Version = "0.1"
)

// struct to store word info
type Word struct {
	Word        string
	ZhTrans     string
	AddedCount  int
	LookupCount int
}

// struct to store word database
type WordDB struct {
	// database connection
	Db *sql.DB
}

func isValidWord(s string) bool {
	match, _ := regexp.MatchString("^[a-z]+$", s)
	return match
}

func getDb() *sql.DB {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Println(err)
		return nil
	}

	// 拼接文件路径
	DbPath := filepath.Join(homeDir, DbName)

	db, err := sql.Open("sqlite3", DbPath)
	if err != nil {
		log.Fatal(err)
	}
	return db
}

// get multiple words from arguments, split
func filterWords(s string) []string {
	// split s with ',', and trim every word, check if it's valid word
	words := strings.Split(s, ",")
	results := make([]string, 0) // Initialize results as an empty string slice

	for _, word := range words {
		word = strings.TrimSpace(word)
		word = strings.ToLower(word)
		if isValidWord(word) {
			results = append(results, word)
		}
	}
	return results
}

// init database
func (w *WordDB) Init() {
	db := w.Db

	sqlStmt := `
    CREATE TABLE word (
        word TEXT PRIMARY KEY,
        zh_trans TEXT,
        added_count INTEGER,
        lookup_count INTEGER
    );
    `
	_, err := db.Exec(sqlStmt)
	if err != nil {
		log.Printf("%q: %s\n", err, sqlStmt)
		return
	}
	log.Printf("init database")
}

// run sql query
func (w *WordDB) RunQuery(query string) {
	db := w.Db
	rows, err := db.Query(query)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
}

// add word to database
func (w *WordDB) AddWord(word string) {
	// word must match an english word
	db := w.Db

	// check if word in database
	var count int
	err := db.QueryRow("SELECT count() FROM word WHERE word=?", word).Scan(&count)
	if err != nil && err != sql.ErrNoRows {
		log.Fatal(err)
	}

	if count == 0 {
		// word not in database
		// insert word
		_, err := db.Exec("INSERT INTO word(word, zh_trans, added_count, lookup_count) values(?, ?, ?, ?)", word, "", 0, 0)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("add word '%s'", word)
	} else {
		_, err = db.Exec("UPDATE word SET added_count=added_count+1 WHERE word=?", word)
		if err != nil {
			log.Fatal(err)
		}

		log.Printf("word '%s' already in database, added_count++", word)
	}
}

// get all words
func (w *WordDB) GetAllWords() []Word {
	var words []Word
	rows, err := w.Db.Query("SELECT * FROM word")
	if err != nil {
		log.Printf("get all words error: %v", err)
		return words
	}
	defer rows.Close()

	for rows.Next() {
		var word Word
		err := rows.Scan(&word.Word, &word.ZhTrans, &word.AddedCount, &word.LookupCount)
		if err != nil {
			log.Printf("get row error: %v", err)
			continue
		}
		words = append(words, word)
	}
	return words
}

// show summary
func (w *WordDB) ShowSummary() {
	words := w.GetAllWords()
	fmt.Printf("%15s %10s %12s %-12s\n", "Word", "Added Count", "Lookup Count", "Translation")
	for _, word := range words {
		fmt.Printf("%15s %10d %12d %-12s\n",
			word.Word, word.AddedCount, word.LookupCount, word.ZhTrans)
	}
}

// delete word from database
func (w *WordDB) DelWord(word string) {
	db := w.Db

	_, err := db.Exec("DELETE FROM word WHERE word=?", word)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("del word '%s'", word)
}

// create a http service to show all words, and generate links to online dictionary
func (w *WordDB) RunWebServer() {

	http.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		var words = w.GetAllWords()
		tmpl := template.Must(template.New("").Parse(`
            <table border=1>
                <tr>
                    <th>Word</th>
                    <th>Added Count</th>
                    <th>Lookup Count</th>
                    <th>Translation</th>
                </tr>
                {{range .}}
                <tr>
                    <th><a href="/word/{{.Word}}">{{.Word}}</a></th>
                    <td>{{.AddedCount}}</td>
                    <td>{{.LookupCount}}</td>
                    <td>{{.ZhTrans}}</td>
                </tr>
                {{end}}
            </table>
        `))

		err := tmpl.Execute(rw, words)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}
	})
	// add /word to show single word
	http.HandleFunc("/word/", func(rw http.ResponseWriter, r *http.Request) {
		word := strings.TrimPrefix(r.URL.Path, "/word/")
		word = strings.TrimSuffix(word, "/")
		if word == "" {
			http.Error(rw, "word not found", http.StatusNotFound)
			return
		}
		// redirect to online dictionary
		http.Redirect(rw, r, "https://dictionary.cambridge.org/dictionary/english-chinese-simplified/"+word, http.StatusMovedPermanently)

	})
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func main() {
	init := flag.Bool("init", false, "init database")
	add := flag.String("a", "", "add new word")
	show := flag.Bool("s", false, "show summary")
	del := flag.String("d", "", "del word")
	daemon := flag.Bool("D", false, "run webserver")
	showVersion := flag.Bool("v", false, "show version")
	flag.Parse()
	// show help when run with no argument
	if flag.NFlag() == 0 {
		flag.Usage()
		return
	}

	if showVersion != nil && *showVersion {
		fmt.Printf("word version: %s\n", Version)
		return
	}

	db := getDb()
	defer db.Close()

	w := WordDB{Db: db}

	if daemon != nil && *daemon {
		w.RunWebServer()
		return
	}

	if init != nil && *init {
		w.Init()
		return
	}

	if add != nil && *add != "" {
		words := filterWords(*add)
		for _, word := range words {
			w.AddWord(word)
		}
		return
	}

	if del != nil && *del != "" {
		w.DelWord(*del)
		return
	}

	if show != nil && *show {
		w.ShowSummary()
		return
	}

}
