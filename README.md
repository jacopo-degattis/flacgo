# Flacgo

## Description

A high-level library to work with FLAC files.

## Install 

```bash
$ go get github.com/jacopo-degattis/go-flac
```

## Features

- Read FLAC metadata blocks
- High-level function to add new metadata

## Example usage

See [`examples/basic.go`](examples/basic.go) for a complete set of examples showing how to read, add, update and remove metadatas.

You can run them with:

```bash
$ go run examples/addmetadata.go
$ go run examples/removemetadata.go
$ go run examples/readmetadata.go
```

More examples will be added.

## License

Refer to [LICENSE](LICENSE)
