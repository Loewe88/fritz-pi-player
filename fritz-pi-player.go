package main

import (
	"fmt"
//	"io/ioutil"
	"net/http"
	"os"
	"log"
	"bufio"
	"strings"
	"html/template"
)

type Station struct {
	Name string
	Logoname string
	Urlname string
	Streamlink string
}

var stationList []Station
var currentStation Station

func makeLogoname(name string) string {
	logoname := makeUrlname(name)

	//fix for RTL2
	logoname = strings.Replace(logoname, "_ii", "_2", -1)

	return logoname
}

func makeUrlname(name string) string {
	logoname := strings.ToLower(name)

	logoname = strings.Replace(logoname, " ", "_", -1)

	logoname = strings.Replace(logoname, "ä", "ae", -1)
	logoname = strings.Replace(logoname, "ö", "oe", -1)
	logoname = strings.Replace(logoname, "ü", "ue", -1)
	logoname = strings.Replace(logoname, "ß", "ss", -1)

	return logoname
}

func readStationList(hostname string) {
	file, err := os.Open("tvhd.m3u")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	var parseStation Station

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "#EXTINF") {
			parseStation.Name = line[10:]
			parseStation.Logoname = makeLogoname(line[10:])
			parseStation.Urlname = makeUrlname(line[10:])
		} else if strings.Contains(line, "rtsp://") {
			parseStation.Streamlink = line
			stationList = append(stationList, parseStation)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

}

func stationHandler(w http.ResponseWriter, r *http.Request) {
	station := getStationForUrlname(r.URL.Path[9:])
	t, _ := template.ParseFiles("ui/templates/base.html", "ui/templates/station.html")
	t.ExecuteTemplate(w, "base", station)
}

func offHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Here I'm turning off the stream later :)")
}

func searchFritzHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Here I'm planning to configure the url of the AVM Fritz! device :)")
}

func listHandler(w http.ResponseWriter, r *http.Request) {

	t, _ := template.ParseFiles("ui/templates/base.html", "ui/templates/stationlist.html")
	t.ExecuteTemplate(w, "base", stationList)
}

func getStationForUrlname(urlname string) Station {
	for i := range stationList {
		if stationList[i].Urlname == urlname {
			return stationList[i]
		}
	}

	return Station{}
}

func main() {
	fs := http.FileServer(http.Dir("ui/static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.HandleFunc("/station/", stationHandler)
	http.HandleFunc("/off/", offHandler)
	http.HandleFunc("/list/", listHandler)
	http.HandleFunc("/search-for-fritzbox/", searchFritzHandler)
	http.HandleFunc("/", listHandler)

	readStationList("fritz.repeater")

	http.ListenAndServe(":8080", nil)
}