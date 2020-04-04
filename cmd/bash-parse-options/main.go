package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Global  bool
	Binding bool
	Writer  io.Writer
}

type Line struct {
	level int
	line  string
}

func (l *Line) String(space string) string {
	return strings.Repeat(space, l.level) + l.line + "\n"
}

type Lines struct {
	lines []*Line
}

func NewLines() *Lines {
	return &Lines{}
}

func (l *Lines) Lines() []*Line {
	return append([]*Line(nil), l.lines...)
}

func (l *Lines) Pushf(level int, format string, args ...interface{}) *Lines {
	line := fmt.Sprintf(format, args...)
	l.lines = append(l.lines, &Line{level: level, line: line})
	return l
}

func (l *Lines) Indent(i int) *Lines {
	for _, line := range l.lines {
		line.level += i
	}
	return l
}

func (l *Lines) AppendLines(r *Lines) *Lines {
	l.lines = append(l.lines, r.lines...)
	return l
}

type Spec struct {
	Global  bool
	Name    string
	Alias   []string
	Kind    string
	Default string
	Array   bool
}

func toOption(str string) string {
	if len(str) == 1 {
		return "-" + str
	}
	return "--" + str
}

func (s *Spec) OptionVariable() string {
	o := "option_" + strings.ReplaceAll(s.Name, "-", "_")
	if s.Global {
		return strings.ToUpper(o)
	}
	return o
}

func (s *Spec) Option() string {
	return toOption(s.Name)
}

func (s *Spec) AllOptions() []string {
	out := []string{s.Option()}
	for _, a := range s.Alias {
		out = append(out, toOption(a))
	}
	return out
}

func (s *Spec) Case() *Lines {
	if s.Kind == "bool" {
		return s.boolCase()
	}
	return s.argvCase()
}

func (s *Spec) boolCase() *Lines {
	l := NewLines()
	l.Pushf(0, "%s)", strings.Join(s.AllOptions(), " | "))
	l.Pushf(1, "%s=1", s.OptionVariable())
	l.Pushf(1, `_argv=("${_argv[@]:1}")`)
	l.Pushf(1, ";;")
	return l
}

func (s *Spec) argvCase() *Lines {
	l := NewLines()
	options := s.AllOptions()
	for _, o := range s.AllOptions() {
		options = append(options, o+"=*")
	}
	l.Pushf(0, "%s)", strings.Join(options, " | "))
	for i, o := range s.AllOptions() {
		withArgv := o + "="
		condition := "if"
		if i != 0 {
			condition = "elif"
		}
		l.Pushf(1, `%s [[ ${_argv[0]} =~ ^%s ]]; then`, condition, withArgv)
		l.Pushf(2, `_v="${_argv[0]##%s}"`, withArgv)
		l.Pushf(2, `_argv=("${_argv[@]:1}")`)
	}
	l.Pushf(1, `else`)
	l.Pushf(2, `if [[ -z ${_argv[1]} ]] || [[ ${_argv[1]} =~ ^- ]]; then`)
	l.Pushf(3, `echo "${_argv[0]} option requires an argument" >&2`)
	l.Pushf(3, `return 1`)
	l.Pushf(2, `fi`)
	l.Pushf(2, `_v="${_argv[1]}"`)
	l.Pushf(2, `_argv=("${_argv[@]:2}")`)
	l.Pushf(1, `fi`)
	if s.Kind == "int" {
		l.Pushf(1, `if [[ ! $_v =~ ^-?[0-9]+$ ]]; then`)
		l.Pushf(2, `echo "%s option takes only integer" >&2`, s.Option())
		l.Pushf(2, `return 1`)
		l.Pushf(1, `fi`)
	}
	if s.Array {
		l.Pushf(1, `%s=("${%s[@]}" "$_v")`, s.OptionVariable(), s.OptionVariable())
	} else {
		l.Pushf(1, `%s="$_v"`, s.OptionVariable())
	}
	l.Pushf(1, `;;`)
	return l
}

func Header(c *Config, ss []*Spec) *Lines {
	l := NewLines()
	if c.Global {
		for _, s := range ss {
			def := s.Default
			if s.Kind == "string" && def != "" {
				def = fmt.Sprintf(`"%s"`, def)
			}
			if s.Array {
				def = "(" + def + ")"
			}
			l.Pushf(0, `%s=%s`, s.OptionVariable(), def)
		}
		l.Pushf(0, `parse_options() {`)
	} else {
		l.Pushf(0, `main() {`)
		for _, s := range ss {
			def := s.Default
			if s.Kind == "string" && def != "" {
				def = fmt.Sprintf(`"%s"`, def)
			}
			if s.Array {
				def = "(" + def + ")"
			}
			l.Pushf(1, `local %s=%s`, s.OptionVariable(), def)
		}
		l.Pushf(1, `local argv=()`)
	}
	l.Pushf(1, `local _argv=("$@")`)
	l.Pushf(1, `local _v`)
	l.Pushf(1, `while [[ ${#_argv[@]} -gt 0 ]]; do`)
	l.Pushf(2, `case "${_argv[0]}" in`)
	return l
}

func Footer(c *Config, ss []*Spec) *Lines {
	l := NewLines()
	if c.Binding {
		l.Pushf(2, `-[a-zA-Z0-9][a-zA-Z0-9]*)`)
		l.Pushf(3, `_v="${_argv[0]:1}"`)
		l.Pushf(3, `_argv=($(echo "$_v" | \grep -o . | \sed -e 's/^/-/') "${_argv[@]:1}")`)
		l.Pushf(3, `;;`)
	}
	l.Pushf(2, `-*)`)
	l.Pushf(3, `echo "Unknown option ${_argv[0]}" >&2`)
	l.Pushf(3, `return 1`)
	l.Pushf(3, `;;`)
	l.Pushf(2, `*)`)
	l.Pushf(3, `argv=("${argv[@]}" "${_argv[0]}")`)
	l.Pushf(3, `_argv=("${_argv[@]:1}")`)
	l.Pushf(3, `;;`)
	l.Pushf(2, `esac`)
	l.Pushf(1, `done`)
	if !c.Global {
		l.Pushf(1, `# WRITE YOUR CODE`)
	}
	l.Pushf(0, `}`)
	if c.Global {
		l.Pushf(0, `parse_options "$@"`)
	} else {
		l.Pushf(0, ``)
		l.Pushf(0, `main "$@"`)
	}
	return l
}

func run(c *Config, specs []*Spec) {
	const indent = "  "
	lines := NewLines()
	lines.AppendLines(Header(c, specs))
	for _, s := range specs {
		lines.AppendLines(s.Case().Indent(2))
	}
	lines.AppendLines(Footer(c, specs))
	for _, l := range lines.Lines() {
		fmt.Fprint(c.Writer, l.String(indent))
	}
}

func parseArgs(c *Config, args []string) ([]*Spec, error) {
	specs := []*Spec{}
	for _, originalArg := range args {
		arg := originalArg
		def := ""
		if defs := strings.Split(arg, ";"); len(defs) > 1 {
			arg = defs[0]
			def = defs[1]
		}
		kv := strings.Split(arg, "=")
		names := strings.Split(kv[0], "|")
		name := names[0]
		var alias []string
		if len(names) > 1 {
			alias = names[1:]
		}
		kind := "bool"
		array := false
		if len(kv) > 1 {
			kindStr := kv[1]
			if strings.HasSuffix(kindStr, "@") {
				array = true
				kindStr = kindStr[:len(kindStr)-1]
			}
			switch kindStr {
			case "s":
				kind = "string"
			case "i":
				kind = "int"
			default:
				return nil, fmt.Errorf("unknown kind in %s", originalArg)
			}
		}
		if def != "" {
			switch kind {
			case "bool":
				if def == "true" {
					def = "1"
				}
				if def == "false" || def == "0" {
					def = ""
				}
				if def != "1" && def != "" {
					return nil, fmt.Errorf("invalid default in %s", originalArg)
				}
			case "int":
				if _, err := strconv.Atoi(def); err != nil {
					return nil, fmt.Errorf("invalid default in %s", originalArg)
				}
			}
		}
		specs = append(specs, &Spec{
			Name:    name,
			Alias:   alias,
			Kind:    kind,
			Array:   array,
			Global:  c.Global,
			Default: def,
		})
	}
	return specs, nil
}

var (
	version           = "dev"
	writer  io.Writer = os.Stdout
)

var usage = `Usage: bash-parse-options [options] specs...

Options:
  -help     show ths help
  -version  show version and exit
  -global   render template that use shell's global variables
  -binding  render template that allows option binding (that is, -abc means -a, -b, -c)

Specs:
  foo        boolean --foo option
  foo|f      boolean --foo option, and it has an alias -f
  foo;true   boolena --foo option, and its defualt is true value
  foo|f|F    boolean --foo option, and it has aliases -f and -F
  bar=s      --bar option that takes string
  bar|b=s    --bar option that takes string, and it has an alias -b
  bar=s;xyz  --bar option that takes string, and its defualt is "xyz"
  bar=s@     --bar option that takes string, and it can be used multiple times
  hoge=i     --hoge option that takes integer
  hoge|h=i   --hoge option that takes integer, and it has an alias -h
  hoge=i;10  --hoge option that takes integer, and its defualt is 10
  hoge=i@    --hoge option that takes integer, and it can be used multiple times

Exmples:
  $ bash-parse-options 'foo'
  $ bash-parse-options -global -binding 'foo|f;true' 'bar|b=s' 'hoge|h=i@'
`

func main() {
	flag.Usage = func() { fmt.Fprintf(flag.CommandLine.Output(), usage) }
	global := flag.Bool("global", false, "use global variables")
	binding := flag.Bool("binding", true, "turn on option binding")
	versionFlag := flag.Bool("version", false, "show version")
	flag.Parse()

	if *versionFlag {
		fmt.Println(version)
		os.Exit(0)
	}

	c := &Config{
		Global:  *global,
		Binding: *binding,
		Writer:  writer,
	}
	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Need spec argument.")
		os.Exit(1)
	}
	specs, err := parseArgs(c, args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	run(c, specs)
}
