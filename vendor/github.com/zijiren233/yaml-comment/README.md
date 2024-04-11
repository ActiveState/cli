# yaml-comment
golang yaml comment encoder

# How to use
```go
package main

import (
    "fmt"
    "github.com/zijiren233/yaml-comment"
)

type Config struct {
    Name string `yaml:"name,omitempty" hc:"this is head comment"`
    Age  int    `yaml:"age,omitempty" lc:"this is line comment" fc:"this is foot comment"`
}

func main() {
    c := Config{
        Name: "comment",
        Age:  18,
    }
    data, err := yamlcomment.Marshal(c)
	if err != nil {
		panic(err)
	}
    fmt.Println(string(data))
}
```

# Output
```yaml
# this is head comment
name: comment
age: 18 # this is line comment
# this is foot comment
```