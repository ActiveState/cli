package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"activestate.com/devx/cli/internal/structures"

	"github.com/gorilla/mux"
)

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

func getPackageFile() []structures.Package {
	raw, err := ioutil.ReadFile("../data/packages.json")
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	packages := make([]structures.Package, 0)
	// var package Package
	err2 := json.Unmarshal([]byte(raw), &packages)

	if err2 != nil {
		log.Fatal(err2)
	}

	return packages
}
