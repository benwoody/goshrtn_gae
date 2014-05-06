package goshrtn

import (
	"crypto/rand"
	"html/template"
	"net/http"
	"strings"
	"time"

	"appengine"
	"appengine/datastore"
)

type Shorten struct {
	LongURL  string
	ShortURL string
	Date     time.Time
}

var newUrlTemplate = template.Must(template.New("url").Parse(`
<html>
  <head>
    <title>Go Shrtn</title>
  </head>
  <body>
    <form action="/new" method="POST">
      <div><input type="text"  name="longurl" cols="64"></textarea></div>
      <div><input type="submit" value="Shorten URL"></div>
    </form>

    <ul>
    {{range .}}
      <li><a href="/s/{{.ShortURL}}">{{.ShortURL}}</a> redirects to {{.LongURL}}</li>
    {{end}}
    </ul>
  </body>
</html>
`))

func init() {
	http.HandleFunc("/", handleRoot)
	http.HandleFunc("/s/", handleRedirect)
	http.HandleFunc("/new", handleNewUrl)
}

func shortenKey(c appengine.Context) *datastore.Key {
	return datastore.NewKey(c, "Shorten", "default_shorten", 0, nil)
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	q := datastore.NewQuery("Shorten").Ancestor(shortenKey(c)).Order("-Date").Limit(10)
	urls := make([]Shorten, 0, 10)
	if _, err := q.GetAll(c, &urls); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := newUrlTemplate.Execute(w, urls); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func handleRedirect(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.NotFound(w, r)
		return
	}
	short := r.URL.Path[len("/s/"):]
	c := appengine.NewContext(r)
	q := datastore.NewQuery("Shorten").Ancestor(shortenKey(c)).Filter("ShortURL =", short).Limit(1)
	url := make([]Shorten, 0, 1)
	if _, err := q.GetAll(c, &url); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for _, long := range url {
		http.Redirect(w, r, long.LongURL, http.StatusFound)
	}
}

func handleNewUrl(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.NotFound(w, r)
		return
	}

	c := appengine.NewContext(r)
	s := Shorten{
		LongURL:  checkUrl(r.FormValue("longurl")),
		ShortURL: generateShortURL(),
		Date:     time.Now(),
	}
	key := datastore.NewIncompleteKey(c, "Shorten", shortenKey(c))
	_, err := datastore.Put(c, key, &s)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusFound)
}

// Needed to clean up bad urls
func checkUrl(url string) string {
	if strings.HasPrefix(url, "http://") {
		return url
	} else {
		s := []string{"http://", url}
		return strings.Join(s, "")
	}
}

func generateShortURL() string {
	var letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

	var bytes = make([]byte, 6)
	rand.Read(bytes)
	for i, b := range bytes {
		bytes[i] = letters[b%byte(len(letters))]
	}
	return string(bytes)
}
