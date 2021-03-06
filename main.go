package main

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/gorilla/mux"
)

func main() {
	muxRouter := mux.NewRouter()
	muxRouter.HandleFunc("/", getLink)
	err := http.ListenAndServe(":8080", muxRouter)
	if err != nil {
		panic(err)
	}
}

func scrape(url string) (title, description, link string) {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Make request
	response, err := client.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer response.Body.Close()

	dataInBytes, err := ioutil.ReadAll(response.Body)
	pageContent := string(dataInBytes)

	// Find a substr
	titleStartIndex := strings.Index(pageContent, "<title>")
	if titleStartIndex == -1 {
		fmt.Println("No title element found")
		os.Exit(0)
	}
	// The start index of the title is the index of the first
	// character, the < symbol. We don't want to include
	// <title> as part of the final value, so let's offset
	// the index by the number of characers in <title>
	titleStartIndex += 7

	// Find the index of the closing tag
	titleEndIndex := strings.Index(pageContent, "</title>")
	if titleEndIndex == -1 {
		fmt.Println("No closing tag for title found.")
		os.Exit(0)
	}

	// (Optional)
	// Copy the substring in to a separate variable so the
	// variables with the full document data can be garbage collected
	pageTitle := []byte(pageContent[titleStartIndex:titleEndIndex])

	// Find a substr
	descriptionStartIndex := strings.Index(pageContent, "window__info-description")
	if descriptionStartIndex == -1 {
		fmt.Println("No description element found")
		os.Exit(0)
	}
	// The start index of the title is the index of the first
	// character, the < symbol. We don't want to include
	// <title> as part of the final value, so let's offset
	// the index by the number of characers in <title>
	descriptionStartIndex += 38

	// Find the index of the closing tag
	descriptionEndIndex := strings.Index(pageContent, "(https://www.patreon.com/electromonkeys)</p>")
	if descriptionEndIndex == -1 {
		fmt.Println("No closing tag for description found.")
		os.Exit(0)
	}
	descriptionEndIndex -= 77

	// (Optional)
	// Copy the substring in to a separate variable so the
	// variables with the full document data can be garbage collected
	pageDescription := []byte(pageContent[descriptionStartIndex:descriptionEndIndex])

	pageLink := url + dnsify(string(pageTitle))

	return string(pageTitle), string(pageDescription), string(pageLink)
}

func generateRSS(title, description, link string) {
	articles := []rssItem{
		{
			Title:       title,
			Link:        link,
			Description: description,
			Image:       "https://storage.buzzsprout.com/variants/k1gf0b0yd0yqkiz0qt1t2zj4smv1/74cb75bab2243992e98fab5156007185827084cf97936f24c0c66a651388df90.jpg",
		},
	}

	rssStruct := &rss{
		Version:       "2.0",
		Title:         "Eletro Monkeys Podcast",
		Link:          "https://electro-monkeys.fr/",
		Description:   "Le podcast pour découvrir et comprendre les technologies et les concepts cloud natifs",
		LastBuildDate: time.Now().Format(time.RFC1123Z),
		Item:          articles,
	}

	data, err := xml.MarshalIndent(rssStruct, "", "    ")
	if err != nil {
		fmt.Println(err)
	}

	rssFeed := []byte(xml.Header + string(data))
	if err := ioutil.WriteFile(filepath.Join("./", "rss.xml"), rssFeed, 0644); err != nil {
		fmt.Println(err)
	}
}

type rss struct {
	Version       string `xml:"version,attr"`
	Title         string `xml:"channel>title"`
	Link          string `xml:"channel>link"`
	Description   string `xml:"channel>description"`
	LastBuildDate string `xml:"channel>lastBuildDate"`

	Item []rssItem `xml:"channel>item"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	Image       string `xml:"image"`
}

func dnsify(title string) (mp3link string) {

	// Il manque des exception, genre "ê"
	replaceSpace := strings.ReplaceAll(title, " ", "-")
	toLower := strings.ToLower(replaceSpace)
	replaceEAcute := strings.ReplaceAll(toLower, "é", "e")
	replaceEGrave := strings.ReplaceAll(replaceEAcute, "è", "e")
	replaceAAcute := strings.ReplaceAll(replaceEGrave, "à", "a")
	replaceApostrophe := strings.ReplaceAll(replaceAAcute, "'", "-")
	mp3Link := "-" + replaceApostrophe + ".mp3"

	return mp3Link
}

func getLink(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET": //check if you only connect on the page
		tpl, err := template.ParseFiles("html_template/welcome.html") //get template file
		if err != nil {
			log.Print("template parsing error: ", err)
		}
		err = tpl.Execute(w, nil) //present template file with variables
		if err != nil {
			log.Print("template executing error: ", err)
		}
	case "POST":
		r.ParseForm() //get form variables

		tpl, err := template.ParseFiles("html_template/startconv.html") //get template file
		if err != nil {
			log.Print("template parsing error: ", err)
		}

		URL := r.Form.Get("bzsptURL")

		generateRSS(scrape(URL))

		err = tpl.Execute(w, nil) //present template file with variables
		if err != nil {
			log.Print("template executing error: ", err)
		}

	default:
		fmt.Printf("Unknown HTTP method : %s", r.Method)
	}
}
