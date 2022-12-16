package main

import (
	"crypto/sha256"
	"fmt"
	"os"

	"github.com/google/uuid"
)

func main() {
	bytes := []byte(os.Args[1])
	hash := sha256.New()
	hash.Write(bytes)

	uuid := uuid.NewHash(hash, uuid.UUID{}, bytes, 0)
	fmt.Print(uuid.String())
}
