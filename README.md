# yjq

YAML wrapper for [jq](https://stedolan.github.io/jq/) commandline JSON processor

```
Usage: yjq [JQ_ARG...]
       yjq [options] [YAML_FILES...] -- [JQ_ARG...]

Options:
      --cmd string    Name of the jq command (default "jq")
  -h, --help          Show usage and exit
  -i, --yaml-input    Convert YAML input to JSON
  -o, --yaml-output   Convert JSON output to YAML
```


## Usage

```
yjq [JQ_ARG...] 
```

By default all arguments are forwarded to `jq` and input/output is YAML. For example:
```
$ echo 'name: foo' | yjq '.name+="_bar"'
name: foo_bar
```

If there's a `--` argument, all arguments up to `--` are handled by `yjq`
and the rest are forwarded to `jq`.
```
yjq [options] [YAML_FILES...] -- [JQ_ARG...]
```
For example:
```
$ echo 'name: foo' | yjq -i -- '.name+="_bar"'
{
  "name": "foo_bar"
}
```

### Options

  - `-i`, `--yaml-input` Convert YAML input from `stdin` to JSON
  - `-o`, `--yaml-output` Convert JSON output to YAML
  - `--cmd <JQ>` Specify alternative jq command (default "jq")

If neither `-i|--yaml-input` nor `-o|--yaml-output` is specified both input and output is YAML.

### Arguments
  
If `stdin` is not piped, `yjq` arguments are assumed to be YAML files to pipe to `jq` as JSON. If there are no arguments, interactive input from `stdin` is used (exit by sending `EOF` with `Ctrl-D`).

## YAML Input

Multiple YAML values separated by `---` are passed to `jq` as separate values.
To combine them in an array use `jq`'s `-s|--slurp` option.


## YAML Output

Results from `jq`'s output are separated by `---` in a single YAML value stream.

## Installation

Download an executable for your platform from github [releases]( https://github.com/alxarch/yjq/releases/latest).

Alternatively, assuming `$GOPATH/bin` is in `$PATH` just

```
go get github.com/alxarch/yjq
```

## Caveats

When input is YAML `jq` argument `-R|--raw-input` is not supported and gets dropped.

When output is YAML the following `jq` arguments ara not supported and get dropped:
  - `-r|--raw-output`
  - `-a|--ascii-output`
  - `-C|--color-output`
  - `-j|--join-output`
  - `--tab`
  - `--indent`
  - `-c|--compact-output`

## TODO

  - Add support for flow style YAML output, especially in long string values
  - Intercept file arguments passed to `jq` and handle the ones with `.yaml,.yml` extension