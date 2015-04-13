package main

import (
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"regexp"
)

type Page struct {
	Title string
	Body  []byte
}

const dataDir = "data"

var (
	templates = template.Must(template.ParseGlob("tmpl/*.html"))
	validPath = regexp.MustCompile("^/((edit|save|view)/([a-zA-Z0-9]+))|(FrontPage.html)$")
	addr      = flag.Bool("addr", false, "find open adress and print final-port.txt")
)

func (p *Page) save() error {
	filename := dataDir + "/" + p.Title + ".txt"
	return ioutil.WriteFile(filename, p.Body, 0600)
}

func loadPage(title string) (*Page, error) {
	filename := dataDir + "/" + title + ".txt"
	body, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return &Page{Title: title, Body: body}, nil
}

func viewHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)
	if err != nil {
		http.Redirect(w, r, "/edit/"+title, http.StatusFound)
		return
	}
	renderTemplate(w, "view", p)
}

func editHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)
	if err != nil {
		p = &Page{Title: title}
	}
	renderTemplate(w, "edit", p)
}

func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
	err := templates.ExecuteTemplate(w, tmpl+".html", p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func saveHandler(w http.ResponseWriter, r *http.Request, title string) {
	body := r.FormValue("body")
	p := &Page{Title: title, Body: []byte(body)}
	err := p.save()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/view/"+title, http.StatusFound)
}

func frontPageHandler(w http.ResponseWriter, r *http.Request, title string) {
	entries, err := ioutil.ReadDir(dataDir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = templates.ExecuteTemplate(w, "FrontPage.html", entries)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func makeHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/tmpl/FrontPage.html", http.StatusFound)
			return
		}
		m := validPath.FindStringSubmatch(r.URL.Path)
		if m == nil {
			fmt.Println("No match found for:", r.URL.Path)
			return
		}
		fn(w, r, m[3])
	}
}

func main() {
	http.HandleFunc("/view/", makeHandler(viewHandler))
	http.HandleFunc("/edit/", makeHandler(editHandler))
	http.HandleFunc("/save/", makeHandler(saveHandler))
	http.HandleFunc("/", makeHandler(frontPageHandler))

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	if *addr {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			log.Fatal(err)
		}
		err = ioutil.WriteFile("final-port.txt", []byte(l.Addr().String()), 0644)
		if err != nil {
			log.Fatal(err)
		}
		s := &http.Server{}
		s.Serve(l)
		return
	}

	http.ListenAndServe(":8080", nil)
}
