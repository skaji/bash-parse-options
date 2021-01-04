# bash-parse-options [![](https://github.com/skaji/bash-parse-options/workflows/test/badge.svg)](https://github.com/skaji/bash-parse-options/actions)

Generate `parse_options` (aka getopt) code for bash

## Install

Download an appropriate tarball from [release page](https://github.com/skaji/bash-parse-options/releases/latest).

## Example

```console
bash-parse-options 'url|u=s@' 'timeout|t=i' 'retry|r'
```

generates the following code:

```bash
main() {
  local option_url=()
  local option_timeout=
  local option_retry=
  local argv=()
  local _argv=("$@")
  local _v
  while [[ ${#_argv[@]} -gt 0 ]]; do
    case "${_argv[0]}" in
    --url | -u | --url=* | -u=*)
      if [[ ${_argv[0]} =~ ^--url= ]]; then
        _v="${_argv[0]##--url=}"
        _argv=("${_argv[@]:1}")
      elif [[ ${_argv[0]} =~ ^-u= ]]; then
        _v="${_argv[0]##-u=}"
        _argv=("${_argv[@]:1}")
      else
        if [[ ${#_argv[@]} -eq 1 ]] || [[ ${_argv[1]} =~ ^- ]]; then
          echo "${_argv[0]} option requires an argument" >&2
          return 1
        fi
        _v="${_argv[1]}"
        _argv=("${_argv[@]:2}")
      fi
      option_url=("${option_url[@]}" "$_v")
      ;;
    --timeout | -t | --timeout=* | -t=*)
      if [[ ${_argv[0]} =~ ^--timeout= ]]; then
        _v="${_argv[0]##--timeout=}"
        _argv=("${_argv[@]:1}")
      elif [[ ${_argv[0]} =~ ^-t= ]]; then
        _v="${_argv[0]##-t=}"
        _argv=("${_argv[@]:1}")
      else
        if [[ ${#_argv[@]} -eq 1 ]] || [[ ${_argv[1]} =~ ^- ]]; then
          echo "${_argv[0]} option requires an argument" >&2
          return 1
        fi
        _v="${_argv[1]}"
        _argv=("${_argv[@]:2}")
      fi
      if [[ ! $_v =~ ^-?[0-9]+$ ]]; then
        echo "--timeout option takes only integer" >&2
        return 1
      fi
      option_timeout="$_v"
      ;;
    --retry | -r)
      option_retry=1
      _argv=("${_argv[@]:1}")
      ;;
    -[a-zA-Z0-9][a-zA-Z0-9]*)
      _v="${_argv[0]:1}"
      _argv=($(echo "$_v" | \grep -o . | \sed -e 's/^/-/') "${_argv[@]:1}")
      ;;
    -?*)
      echo "Unknown option ${_argv[0]}" >&2
      return 1
      ;;
    *)
      argv+=("${_argv[0]}")
      _argv=("${_argv[@]:1}")
      ;;
    esac
  done
  # WRITE YOUR CODE
}

main "$@"
```

## QA

### I want to use global variables instead of function local variables for parse options. What should I do?

Use `-global` option.
For example, `bash-parse-options -global 'foo'` generates the following code
so that you can use the global variable `OPTION_FOO` everywhere.

```bash
OPTION_FOO=
ARGV=()
parse_options() {
  local _argv=("$@")
  local _v
  while [[ ${#_argv[@]} -gt 0 ]]; do
    case "${_argv[0]}" in
    --foo)
      OPTION_FOO=1
      _argv=("${_argv[@]:1}")
      ;;
    -[a-zA-Z0-9][a-zA-Z0-9]*)
      _v="${_argv[0]:1}"
      _argv=($(echo "$_v" | \grep -o . | \sed -e 's/^/-/') "${_argv[@]:1}")
      ;;
    -?*)
      echo "Unknown option ${_argv[0]}" >&2
      return 1
      ;;
    *)
      ARGV+=("${_argv[0]}")
      _argv=("${_argv[@]:1}")
      ;;
    esac
  done
}
parse_options "$@"
```

## Author

Shoichi Kaji

## License

MIT
