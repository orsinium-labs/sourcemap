# sourcemap

Some tools that generate JS code produce "source maps". Source maps allow to restore the original source code from the minified ugly one. You can read more here:

+ [An Introduction to Source Maps](https://blog.teamtreehouse.com/introduction-source-maps) by Treehouse.
+ [Use a source map](https://developer.mozilla.org/en-US/docs/Tools/Debugger/How_to/Use_a_source_map) by MDN.
+ [Map Preprocessed Code to Source Code](https://developer.chrome.com/docs/devtools/javascript/source-maps/) by Chrome.

This tool finds source maps on the given webpage and restores the application source code from it.

## Build

```bash
git clone https://github.com/orsinium-labs/sourcemap.git
cd sourcemap
go build -o sourcemap .
```

## Use

Feed URLs into [stdin](https://en.wikipedia.org/wiki/Standard_streams#Standard_input_(stdin)):

```bash
echo "https://orsinium.dev/" | ./sourcemap --output=./sources
```
