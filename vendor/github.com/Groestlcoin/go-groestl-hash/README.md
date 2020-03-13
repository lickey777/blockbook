# go-groestl-hash

Implements the Groestl hash and required functions in Go

## Usage

```go
package main

import (
	"fmt"

	"github.com/Groestlcoin/go-groestl-hash/groestl"
)

func main() {
	g, out := groestl.New(), [64]byte{}
	g.Write([]byte("Groestl"))
	g.Close(out[:], 0, 0)
	fmt.Printf("%x \n", out[:])
}
```

## License

go-groestl-hash is licensed under the [copyfree](http://copyfree.org) ISC License.

## Attribution/Credit

This entire repository is based on an original work by Nitya Sattva originally
found at https://gitlab.com/nitya-sattva/go-x11.
