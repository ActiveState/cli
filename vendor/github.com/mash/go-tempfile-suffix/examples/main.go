package main

import (
	"log"
	"os"

	"github.com/mash/go-tempfile-suffix"
)

func main() {
	f, err := tempfile.TempFileWithSuffix("", "prefix", ".suffix")
	if f != nil {
		defer func() {
			f.Close()
			os.Remove(f.Name())
		}()
	}
	if err != nil {
		log.Printf("error: %s", err.Error())
		return
	}
	log.Printf("got tempfile: %s", f.Name())
	// 2015/07/31 18:37:20 got tempfile: /var/folders/gc/694vhx9n0q57vw71w6__rf4h0000gn/T/prefix905642731.suffix
}
