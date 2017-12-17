package main

import (
	"fmt"
	"net/http"
	"log"
	"bufio"
	"strings"
	"html/template"
	"os/exec"
	"syscall"
	"io"
)

type Station struct {
	Name string
	Logoname string
	Urlname string
	Streamlink string
}

type templateData struct {
	CurrentStation Station
	StationListSD []Station
	StationListHD []Station
	Streaming bool
}

var stationList []Station
var stationListSD []Station
var stationListHD []Station
var currentStation Station
var streaming bool
var omxplayerCmd *exec.Cmd

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
func readM3U(playlist io.Reader) []Station {
	var parsedStations []Station

	var parseStation Station

	scanner := bufio.NewScanner(playlist)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "#EXTINF") {
			parseStation.Name = line[10:]
			parseStation.Logoname = makeLogoname(line[10:])
			parseStation.Urlname = makeUrlname(line[10:])
		} else if strings.Contains(line, "rtsp://") {
			parseStation.Streamlink = line
			parsedStations = append(parsedStations, parseStation)
			stationList = append(stationList, parseStation)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	return parsedStations
}
func readStationList(hostname string) {


	response, e := http.Get("http://" + hostname + "/dvb/m3u/tvsd.m3u")
	if e != nil {
		log.Fatal(e)
	}
	defer response.Body.Close()

	stationListSD = readM3U(response.Body)

	response, e = http.Get("http://" + hostname + "/dvb/m3u/tvhd.m3u")
	if e != nil {
		log.Fatal(e)
	}
	defer response.Body.Close()

	stationListHD = readM3U(response.Body)



}

func killStream() {
	if streaming == true {
		log.Printf("Killing stream of station '%s'", currentStation.Name)
		syscall.Kill(-omxplayerCmd.Process.Pid, syscall.SIGKILL)
		err := omxplayerCmd.Wait()
		fmt.Println(err)

		streaming = false
	}
}
func startStream(station Station) {

	killStream()

	currentStation = station
	streaming = true

	omxplayerCmd = exec.Command("/usr/bin/omxplayer", "-o", "hdmi", station.Streamlink)
	omxplayerCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	err := omxplayerCmd.Start()
	log.Printf("Started stream of station '%s'", station.Name)
	if err != nil {
		log.Fatal(err)
	}
}

func stationHandler(w http.ResponseWriter, r *http.Request) {
	station := getStationForUrlname(r.URL.Path[9:])
	go startStream(station)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func offHandler(w http.ResponseWriter, r *http.Request) {
	killStream()
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func searchFritzHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Here I'm planning to configure the url of the AVM Fritz! device :)")
}

func listHandler(w http.ResponseWriter, r *http.Request) {

	t, _ := template.ParseFiles("ui/templates/base.html", "ui/templates/stationlist.html")
	t.ExecuteTemplate(w, "base", templateData{currentStation,stationListSD, stationListHD, streaming})
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