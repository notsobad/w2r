package main

import (
	"context"
	"database/sql"
	"embed"
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
	"github.com/notsobad/w2r/worddb"
)

var (
	DbName  = ".word.sqlite" // in $HOME directory
	Version = "0.1"
	//go:embed words.html
	WordsHTML embed.FS
)

// struct to store word database
type WordDB struct {
	// database connection
	Db  *sql.DB
	Ctx context.Context
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

// add word to database
func (w *WordDB) AddWord(word string) {

	queries := worddb.New(w.Db)

	count, _ := queries.CountWord(w.Ctx, word)
	if count == 0 {
		_, err := queries.CreateWord(w.Ctx, worddb.CreateWordParams{Word: word, ZhTrans: sql.NullString{}})
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("add word '%s'", word)
	} else {
		err := queries.AddWordCount(w.Ctx, word)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("word '%s' already in database, added_count++", word)
	}
}

// show summary
func (w *WordDB) ShowSummary() {

	queries := worddb.New(w.Db)
	words, _ := queries.Listword(w.Ctx)

	fmt.Printf("%15s %10s %12s %-12s\n", "Word", "Added Count", "Lookup Count", "Translation")
	for _, word := range words {
		lookupCount := word.LookupCount.Int64
		if !word.LookupCount.Valid {
			lookupCount = 0
		}
		zhTrans := ""
		if word.ZhTrans.Valid {
			zhTrans = word.ZhTrans.String
		}
		fmt.Printf("%15s %10d %12d %-12s\n",
			word.Word, word.AddedCount.Int64, lookupCount, zhTrans)
	}
}

// delete word from database
func (w *WordDB) DelWord(word string) {
	db := w.Db

	queries := worddb.New(db)
	err := queries.DeleteWord(w.Ctx, word)

	if err != nil {
		log.Fatal(err)
	}
	log.Printf("del word '%s'", word)
}

// create a http service to show all words, and generate links to online dictionary
func (w *WordDB) RunWebServer(port int) {
	tmpl, err := template.ParseFS(WordsHTML, "words.html")
	if err != nil {
		// handle error
		log.Fatal(err)
	}

	http.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {

		queries := worddb.New(w.Db)
		words, _ := queries.Listword(w.Ctx)

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
	log.Printf("Start web server at http://127.0.0.1:%d", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf("127.0.0.1:%d", port), nil))
}

func main() {
	init := flag.Bool("init", false, "init database")
	add := flag.String("a", "", "add new word")
	show := flag.Bool("s", false, "show summary")
	del := flag.String("d", "", "del word")
	daemon := flag.Bool("D", false, "run webserver")
	port := flag.Int("p", 8080, "webserver port")
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
	w.Ctx = context.Background()

	if daemon != nil && *daemon {
		// port must be between 0~65535
		if *port <= 0 || *port > 65535 {
			log.Fatal("port must be between 0~65535")
		}

		w.RunWebServer(*port)
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
