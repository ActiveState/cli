package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
)

type Packages struct {
	packages []Package
}

type Package struct {
	name         string `json:"name"`
	dependencies []string
	url          string `json:"url"`
}

func main() {
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/get", Index)
	log.Fatal(http.ListenAndServe(":8080", router))
}

func Index(w http.ResponseWriter, r *http.Request) {
	packages := getPackageFile()
	s, _ := json.Marshal(packages)
	fmt.Fprintf(w, "package, %q", s)
}

func getPackageFile() Packages {
	raw, err := ioutil.ReadFile("../data/packages.json")
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	var packages Packages
	// var package Package
	json.Unmarshal([]byte(raw), &packages)

	return packages
}
