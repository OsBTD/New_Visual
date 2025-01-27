package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

// Artists represents the artist data structure
type Artists struct {
	Image          string   `json:"image"`
	ID             int      `json:"id"`
	Name           string   `json:"name"`
	Members        []string `json:"members"`
	CreationDate   int      `json:"creationDate"`
	FirstAlbum     string   `json:"firstAlbum"`
	RelationsURL   string   `json:"relations"`
	DatesLocations Relations
}

// Relations represents the concert dates and locations data
type Relations struct {
	ID             int                 `json:"id"`
	DatesLocations map[string][]string `json:"datesLocations"`
}

// RelationsResponse represents the API response structure
type RelationsResponse struct {
	Index []Relations `json:"index"`
}

// ErrorPage represents the data structure for error information
type ErrorPage struct {
	Code    int
	Message string
	Is405   bool
	Is404   bool
	Is500   bool
	Is403   bool
}

// fetchData makes an HTTP GET request and decodes the JSON response
func fetchData(url string, target interface{}) error {
	response, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("error making GET request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("received non-200 response code: %d", response.StatusCode)
	}
	return json.NewDecoder(response.Body).Decode(target)
}

// handleError handles error responses consistently across handlers
func handleError(w http.ResponseWriter, tmpl *template.Template, code int, message string) {
	errorPage := ErrorPage{
		Code:    code,
		Message: message,
		Is405:   code == http.StatusMethodNotAllowed,
		Is404:   code == http.StatusNotFound,
		Is500:   code == http.StatusInternalServerError,
		Is403:   code == http.StatusForbidden,
	}
	w.WriteHeader(code)
	if err := tmpl.Execute(w, errorPage); err != nil {
		log.Printf("Error executing error template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// restrict is a middleware that restricts access to specific paths, /static and /images in this case
// it takes a next(handlerfunc) and returns an http handler function that checks if our path is one of the restricted ones if so the file to parse and execute would be the 403 template and status is 403 forbidden
// if the path doesn't figure in our restricted ones the handlerfunc is returned the usual way and the file to be parsed and executed would be determined
func Restrict(next http.HandlerFunc) http.HandlerFunc {
	templates := make(map[string]*template.Template)
	templateFiles := map[string]string{
		"index":  "templates/index.html",
		"error":  "templates/error.html",
		"about":  "templates/about.html",
		"readme": "templates/readme.html",
	}

	for name, file := range templateFiles {
		tmpl, err := template.ParseFiles(file)
		if err != nil {
			log.Fatalf("Error parsing template %s: %v", name, err)
		}
		templates[name] = tmpl
	}

	return func(w http.ResponseWriter, r *http.Request) {
		restrictedPaths := []string{"/static", "/assets", "/static/assets"}
		for _, path := range restrictedPaths {
			if r.URL.Path == path || r.URL.Path == path+"/" {
				handleError(w, templates["error"], http.StatusForbidden, "Access Denied")
				return
			}
		}
		next(w, r)
	}
}

// this custom file sever allows to customize the errors in file serving
// for example if a file we're trying to serve doesn't exist or if we don't have the necessary permissions
// otherwise if a file doesn't exist for example a standard 404 error would be displayed
// it takes the root as parameter and returns a handler
// it uses os.Stat which returns meta data about a file on our os system
// and returns an error which could be of two type
// either the file doesn't exist or permissions denied
// we're using serve file instead of fileserver cause it only serves one file instead of a whole directory
func customFileServer(root string) http.Handler {
	templates := make(map[string]*template.Template)
	templateFiles := map[string]string{
		"index":  "templates/index.html",
		"error":  "templates/error.html",
		"about":  "templates/about.html",
		"readme": "templates/readme.html",
	}

	for name, file := range templateFiles {
		tmpl, err := template.ParseFiles(file)
		if err != nil {
			log.Fatalf("Error parsing template %s: %v", name, err)
		}
		templates[name] = tmpl
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Join(root, r.URL.Path)
		if _, err := os.Stat(path); err != nil {
			handleError(w, templates["error"], http.StatusNotFound, "Page not found")
			return
		}
		http.ServeFile(w, r, path)
	})
}

func main() {
	// Parse templates
	templates := make(map[string]*template.Template)
	templateFiles := map[string]string{
		"index":  "templates/index.html",
		"error":  "templates/error.html",
		"about":  "templates/about.html",
		"readme": "templates/readme.html",
	}

	for name, file := range templateFiles {
		tmpl, err := template.ParseFiles(file)
		if err != nil {
			log.Fatalf("Error parsing template %s: %v", name, err)
		}
		templates[name] = tmpl
	}

	// Fetch and prepare data
	var artists []Artists
	var relationsResponse RelationsResponse

	// Fetch relations data
	if err := fetchData("https://groupietrackers.herokuapp.com/api/relation", &relationsResponse); err != nil {
		log.Printf("Error fetching relations: %v", err)
	}

	// Fetch artists data
	if err := fetchData("https://groupietrackers.herokuapp.com/api/artists", &artists); err != nil {
		log.Printf("Error fetching artists: %v", err)
	}

	// Map relations to artists
	relationsMap := make(map[int]Relations)
	for _, relation := range relationsResponse.Index {
		relationsMap[relation.ID] = relation
	}
	for i := range artists {
		relation, found := relationsMap[artists[i].ID]
		if found {
			artists[i].DatesLocations = relation
		}
	}

	// Add The Weeknd to artists
	theWeeknd := Artists{
		Image:        "/static/assets/xo.jpeg",
		ID:           54,
		Name:         "The Weeknd",
		Members:      []string{"Abel Tesfaye"},
		CreationDate: 2009,
		FirstAlbum:   "House of baloons",
		DatesLocations: Relations{
			ID: 54,
			DatesLocations: map[string][]string{
				"new_york_usa":   {"27-11-2016", "26-11-2016"},
				"toronto_canada": {"05-09-2016", "04-09-2016"},
				"oujda_morocco":  {"02-12-2016", "01-12-2016"},
			},
		},
	}
	artists = append([]Artists{theWeeknd}, artists...)

	// Define route handlers
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			handleError(w, templates["error"], http.StatusNotFound, "Page not found")
			return
		}

		if r.Method != http.MethodGet {
			handleError(w, templates["error"], http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		if err := templates["index"].Execute(w, artists); err != nil {
			log.Printf("Error executing index template: %v", err)
			handleError(w, templates["error"], http.StatusInternalServerError, "Internal server error")
		}
	})

	http.HandleFunc("/about", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/about" {
			handleError(w, templates["error"], http.StatusNotFound, "Page not found")
			return
		}

		if r.Method != http.MethodGet {
			handleError(w, templates["error"], http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		if err := templates["about"].Execute(w, nil); err != nil {
			log.Printf("Error executing about template: %v", err)
			handleError(w, templates["error"], http.StatusInternalServerError, "Internal server error")
		}
	})

	http.HandleFunc("/readme", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/readme" {
			handleError(w, templates["error"], http.StatusNotFound, "Page not found")
			return
		}

		if r.Method != http.MethodGet {
			handleError(w, templates["error"], http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		if err := templates["readme"].Execute(w, nil); err != nil {
			log.Printf("Error executing readme template: %v", err)
			handleError(w, templates["error"], http.StatusInternalServerError, "Internal server error")
		}
	})

	// Serve static files
	http.Handle("/static/", http.StripPrefix("/static/", customFileServer("templates")))
	http.Handle("/assets/", customFileServer("templates"))

	// Start server
	port := ":8080"
	fmt.Printf("Server started at http://localhost%s\n", port)
	if err := http.ListenAndServe(port, Restrict(http.DefaultServeMux.ServeHTTP)); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
