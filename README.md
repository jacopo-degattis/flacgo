# Flacgo

## Description

A high-level library to work with FLAC files.

## Install 

```bash
$ go get github.com/jacopo-degattis/flacgo
```

## Features

- Read metadata from FLAC file.
- Add and remove metadata to/from the FLAC file.
- Add or remove cover picture to/from a FLAC file.

## Example usage

See [`examples`](examples/) folder for a complete set of examples showing how to read, add, update and remove metadatas.

You can run them with:

```bash
$ go run examples/addcoverimage.go
$ go run examples/addmetadata.go
$ go run examples/bulkaddmetadata.go
$ go run examples/overwriteoriginalfile.go
$ go run examples/readmetadata.go
$ go run examples/removecoverimage.go
$ go run examples/removemetadata.go
```

More examples will be added.

## License

Refer to [LICENSE](LICENSE)
