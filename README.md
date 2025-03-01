# Curlie

If you like the interface of [HTTPie](https://httpie.org) but miss the features of [curl](https://curl.haxx.se), curlie is what you are searching for. Curlie is a frontend to `curl` that adds the ease of use of `httpie`, without compromising on features and performance. All `curl` options are exposed with syntax sugar and output formatting inspired from `httpie`.

## Install

Using [homebrew](https://brew.sh/):

```sh
brew install curlie
```

Using [webi](https://webinstall.dev/curlie/):

```sh
# macOS / Linux
curl -sS https://webinstall.dev/curlie | bash
```

```pwsh
# Windows
curl.exe -A "MS" https://webinstall.dev/curlie | powershell
```

Using [eget](https://github.com/zyedidia/eget):

```sh
# Ubuntu/Debian
eget rs/curlie -a deb --to=curlie.deb
sudo dpkg -i curlie.deb
```

Using [macports](https://www.macports.org):

```sh
sudo port install curlie
```

Using [pkg](https://man.freebsd.org/pkg/8):

```sh
pkg install curlie
```

Using [go](https://golang.org/):

```sh
go install github.com/rs/curlie@latest
```

Using [scoop](https://scoop.sh/):

```sh
scoop install curlie
```

Or download a [binary package](https://github.com/rs/curlie/releases/latest).

## Usage

Synopsis:

```sh
curlie [CURL_OPTIONS...] [METHOD] URL [ITEM [ITEM]]
```

Simple GET:

![Simple GET request example](doc/get.png)

Custom method, headers and JSON data:

![Custom PUT request with headers and JSON data example](doc/put.png)

When running interactively, `curlie` provides pretty-printed output for json. To force pretty-printed output, pass `--pretty`.

## Build

Build with [goreleaser](https://goreleaser.com) to test that all platforms compile properly.

```sh
goreleaser build --clean --snapshot
```

Or for your current platform only.

```sh
goreleaser build --clean --snapshot --single-target
```

## Differences with httpie

* Like `curl` but unlike `httpie`, headers are written on `stderr` instead of `stdout`.
* Output is not buffered, all the formatting is done on the fly so you can easily debug streamed data.
* Use the `--curl` option to print executed curl command.

## License

All source code is licensed under the [MIT License](https://raw.github.com/rs/curlie/master/LICENSE).
