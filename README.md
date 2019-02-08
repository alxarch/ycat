# ycat

Command line processor for YAML/JSON files using [Jsonnet](https://jsonnet.org/)

## Usage
```
ycat - command line YAML/JSON processor

USAGE:
    ycat [OPTIONS] [INPUT...]
    ycat [OPTIONS] [PIPELINE...]

OPTIONS:
    -o, --out {json|j|yaml|y}    Set output format
    -h, --help                   Show help and exit

INPUT:
    [FILE...]                    Read values from file(s)
    -y, --yaml [FILE...]         Read YAML values from file(s)
    -j, --json [FILE...]         Read JSON values from file(s)
    -n, --null                   Inject a null value 
    -a, --array                  Merge values to array

PIPELINE:
    [INPUT...] [ENV...] EVAL

ENV:
    -v, --var <VAR>=<CODE>       Bind Jsonnet variable to code
              <VAR>==<VALUE>     Bind Jsonnet variable to a string value
    -i, --import <VAR>=<FILE>    Import file into a local Jsonnet variable
        --input-var <VAR>        Change the name of the input value variable (default x) 
        --max-stack <SIZE>       Jsonnet VM max stack size (default 500)

EVAL:
    -e, --eval <SNIPPET>         Process values with Jsonnet


If no INPUT is specified, values are read from stdin as YAML.
If FILE is "-" or "" values are read from stdin until EOF.
If FILE has no type option format is detected from extension.
```

## Examples

Concatenate files to a single YAML stream (type is selected from extension)

```
$ ycat foo.yaml bar.yaml baz.json
```

Concatenate files to single JSON stream (one item per-line)

```
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
$ ycat a.json b.yaml -a
```

Concatenate YAML from `a.yaml` and `b.yaml` setting key `foo` to `bar` on each top level object

```
$ ycat a.yaml b.yaml -e 'x+{foo: "bar"}'
```

Same as above with results merged into a single array

```
$ ycat a.yaml b.yaml -e 'x+{foo: "bar"}' -a
```

Add kubernetes namespace `foo` to all resources without namespace

```
$ ycat *.yaml -e 'x + { metadata +: { namespace: if "namespace" in super then super.namespace else "foo" }}'
```

Process with [jq](http://stedolan.github.io/jq/) using a pipe

```
$ ycat -oj a.yaml b.json | jq ... | ycat 
```

## Installation

Download an executable for your platform from github [releases]( https://github.com/alxarch/ycat/releases/latest).

Alternatively, assuming `$GOPATH/bin` is in `$PATH` just

```
go get github.com/alxarch/ycat/cmd/ycat
```



## Input / Output

### YAML input

Multiple YAML values separated by `---\n` are processed separately.
Value reading stops at `...\n` or `EOF`.

### YAML output

Each result value is appended to the output with `---\n` separator.

### JSON input

Multiple JSON values separated by whitespace are processed separately.
Value reading stops at `EOF`.

### JSON output

Each result value is appended into a new line of output.

## Jsonnet

[Jsonnet](https://jsonnet.org/) is a templating language from google that's really versatile in handling configuration files. Visit their site for more information.

Each value is bound to a local variable named `x` inside the snippet by default. Use `--input-var` to change the name.

To use `Jsonnet` code from a file in the snippet use `-i <VAR>=<FILE>` and the exported value will be available as
a local variable in the snippet.

## TODO

  - Add support for pretty printed output
  - Add support for reading .txt files
  - Add support for reading files as base64
  - Add support for reading files as hex
  - Add support for Jsonnet libraries
  - Add support for sorting by JSONPath