bitflags-go
===========
This package is a simple wrapper for working with [bit field](https://en.wikipedia.org/wiki/Bit_field) in Go.

Go 1.18+ required, because generics.

Example code:

```go
package main

import (
	"fmt"
	"github.com/imacks/bitflags-go"
)

// enum type has to be integer type, such as byte, int, etc. Can be unsigned.
type fruits int

const (
	apple fruits = 1<<iota // 1
	orange	// 2
	banana	// 4
)

func main() {
	var basket fruits
	basket = bitflags.Set(basket, apple)
	// has apple? true
	fmt.Printf("has apple? %t\n", bitflags.Has(basket, apple))
	basket = bitflags.Del(basket, apple)
	// has apple? false
	fmt.Printf("has apple? %t\n", bitflags.Has(basket, apple))
	basket = bitflags.Toggle(basket, banana)
	// has banana? true
	fmt.Printf("has banana? %t\n", bitflags.Has(basket, banana))
}
```

