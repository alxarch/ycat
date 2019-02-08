# ycat

Comand line processor for YAML/JSON files using [Jsonnet](https://jsonnet.org/)

```
Usage: ycat [options|files...]

Options:
    -h, --help                   Show help and exit
    -y, --yaml [files...]        Read YAML values from file(s)
    -j, --json [files...]        Read JSON values from file(s)
    -n, --null                   Use null value input (no reading)
    -o, --out {json|j|yaml|y}    Set output format
        --to-json                Output JSON one value per line (same as -o json, -oj)
    -a, --array                  Merge values into an array
    -e, --eval <snippet>         Process values with Jsonnet
    -v, --var <var>=<code>       Bind Jsonnet variable to code
              <var>==<value>     Bind Jsonnet variable to a string value
    -i, --import <var>=<file>    Import file into a local Jsonnet variable
        --input-var <var>        Change the name of the input value variable (default x) 
        --max-stack <size>       Jsonnet VM max stack size (default 500)
```

### Arguments

All non-option arguments are files to read from.
Format is guessed from extension and fallsback to YAML.

If filename is `-` values are read from stdin using the last specified format or YAML
  
## Examples

Concatenate files to a single YAML stream (type is selected from extension)

```
$ ycat foo.yaml bar.yaml baz.json
```

Concatenate files to single JSON stream (one item per-line)

```
$ ycat --to-json foo.yaml bar.yaml baz.json
$ ycat -o j foo.yaml bar.yaml baz.json
```

Concatenate JSON values from `stdin` to a single YAML file

```
$ ycat -j
```

Concatenate YAML from `a.txt`, `stdin`, `b.yaml` and JSON from `a.json`, 

```
$ ycat -y a.txt - b.txt -j a.json
```

Concatenate to YAML array

```
$ ycat -a a.json b.yaml
```

Concatenate YAML from `a.yaml` and `b.yaml` setting key `foo` to `bar` on each top level object

```
$ ycat a.yaml b.yaml -e 'x+{foo: "bar"}'
```

Add kubernetes namespace `foo` to all resources without namespace

```
$ ycat -e 'x + { metadata +: { namespace: if "namespace" in super then super.namespace else "foo" }}' *.yaml
```

Process with [jq](http://stedolan.github.io/jq/) using a pipe

```
$ ycat -oj a.yaml b.json | jq ... | ycat 
```

## Installation

Download an executable for your platform from github [releases]( https://github.com/alxarch/ycat/releases/latest).

Alternatively, assuming `$GOPATH/bin` is in `$PATH` just

```
go get github.com/alxarch/ycat
```



## YAML Input

Multiple YAML values separated by `---\n` are processed separately.
Value reading stops at `...\n` or `EOF`.

## JSON Input

Multiple JSON values separated by whitespace are processed separately.
Value reading stops at `EOF`.

## YAML Output

Each result value is appended to the output with `---\n` separator.

## JSON Output

Each result value is appended into a new line of output.

## Jsonnet

[Jsonnet](https://jsonnet.org/) is a templating language from google that's really versatile in handling configuration files. Visit their site for more information.

Each value is bound to a local variable named `x` inside the snippet by default. Use `--bind` to change the name.

To use `Jsonnet` code from a file in the snippet use `-m <var>=<file>` and the exported value will be available as
a local variable in the snippet.

## TODO

  - Add support for pretty printed output
  - Add support for reading .txt files
  - Add support for reading files as base64
  - Add support for reading files as hex
  - Add support for Jsonnet libraries
  - Add support for sorting by JSONPath