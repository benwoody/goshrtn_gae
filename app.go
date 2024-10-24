package goshrtn

import (
	"crypto/rand"
	"errors"
	"html/template"
	"net/http"
	"net/url"
	"time"

	"appengine"
	"appengine/datastore"
)

// Shorten holds the given URL(LongURL), its shortened version (ShortURL) and
// the time it was created
type Shorten struct {
	LongURL  string
	ShortURL string
	Date     time.Time
}

// Index template for creating new Shorter URLs and displaying created
// Short URLs
var newUrlTemplate = template.Must(template.New("url").Parse(`
<html>
  <head>
    <title>Go Shrtn</title>
  </head>
  <body>
    <form action="/new" method="POST">
      <div><input type="text" name="longurl" cols="64"></textarea></div>
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

// Initialize net/http handlers and routes
func init() {
	http.HandleFunc("/", handleRoot)
	http.HandleFunc("/s/", handleRedirect)
	http.HandleFunc("/new", handleNewUrl)
}

// Handle data from the AE Datastore
func shortenKey(c appengine.Context) *datastore.Key {
	return datastore.NewKey(c, "Shorten", "default_shorten", 0, nil)
}

// Handler function for returning the Index of Short URLs
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

// Get the ShortURL from the path after /s and redirect to the LongURL
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

// Create a new ShortURL
func handleNewUrl(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.NotFound(w, r)
		return
	}

	c := appengine.NewContext(r)
	longUrl, badUrl := checkUrl(r.FormValue("longurl"))
	if badUrl != nil {
		http.Error(w, badUrl.Error(), http.StatusBadRequest)
		return
	}
	s := Shorten{
		LongURL:  longUrl,
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

// checkURL validates the URL being passed to LongURL.
// FIXME As of now, a FQD is needed to create a ShortURL.  This could be
// handled better:
//    a. Show better error message with redirect to handleRoot
//    b. If LongURL is "close" to being a URL, rebuild to FQD
func checkUrl(uri string) (string, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return "Invalid URL", err
	}
	if u.IsAbs() {
		return uri, nil
	} else {
		return "Invalid URL", errors.New("Invalid URL structure")
	}
}

// Generates a random 6 alpha string for the ShortURL
func generateShortURL() string {
	var letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

	var bytes = make([]byte, 6)
	rand.Read(bytes)
	for i, b := range bytes {
		bytes[i] = letters[b%byte(len(letters))]
	}
	return string(bytes)
}
