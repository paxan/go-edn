# Package edn

This [EDN][edn] encoding/decoding package is inspired and remixed from
Go's excellent `encoding/json` package.

[![GoDoc](https://godoc.org/github.com/paxan/go-edn?status.png)](https://godoc.org/github.com/paxan/go-edn)

**Warning:** currently, it supports the following:

 * `Marshal` function that encodes a Go value into EDN.
 * `TextMarshaler`-implementing objects can be marshaled.
 * `Encoder` for writing EDN objects to an output stream.

Please inspect the project's issues to see what is missing or buggy.

[edn]: https://github.com/edn-format/edn/blob/master/README.md

## Sets, Symbols and Keywords

Go lacks EDN sets, symbols and keywords. This package awkwardly
attempts to remedy this deficiency by implementing: `Set`, `Symbol`, `Keyword`.

These types are not fully fleshed out and their programming interface
will need to be improved, and **will certainly change.**

## Bytes

Go `[]byte` objects will be serialized like so:

    #base64 "YW55ICsgb2xkICYgZGF0YQ=="

**Note:** I've taken a liberty here, and chose `#base64` tag. This may change
to something else when EDN spec gets updated to accommodate byte array objects.
