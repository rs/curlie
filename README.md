# HTTPie for curl

If you like the interface of [HTTPie](https://httpie.org) but miss the features of [curl](https://curl.haxx.se), curl-httpie is what you are searching for. Curl-httpie is a drop-in replacement for `httpie` that use `curl` to perform operations. All `curl` options exposed with most syntax sugar and output formatting provided by `httpie`.

## Install

Using [homebrew](http://brew.sh/):

```
brew install rs/tap/curl-httpie
```

Or download a [binary package](https://github.com/rs/curl-httpie/releases/latest).

## Usage

Hello World:

    $ http httpie.org

Synopsis:

    $ http [curl options] [METHOD] URL [ITEM [ITEM]]

See [HTTPie doc](https://httpie.org/doc) for more examples.

## Differences with httpie

* Like `curl` but unlike `httpie`, headers are written on `stderr` instead of `stdout`.
* Output is not buffered, all the formatting is done on the fly so you can easily debug streamed data.
* User the `--curl` option to print executed curl command.
