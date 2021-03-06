package complete

import (
	"bytes"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

type Context int

const (
	None Context = iota
	UnknownKeyword
	Keyword
	FileName
	OverSized
)

var (
	keywordpat = regexp.MustCompile("^[[]([a-zA-Z0-9]+):([$%a-zA-Z0-9.-_]*)[]]$")
	keywordarg = regexp.MustCompile("^(-{1,2})([a-zA-Z0-9]*)(={0,1})([^ =]*)$")
)

type Complete struct {
	directory  string
	positional [][]string
	keyword    map[string][]string
	multiple   bool
}

func Compile(val string, dict map[string][]string) (*Complete, error) {
	c := new(Complete)
	c.keyword = make(map[string][]string)
	lis := strings.Split(val, " ")
	c.positional = make([][]string, len(lis))
	addword := func(s string) ([]string, error) {
		if strings.HasSuffix(s, "...") {
			if c.multiple {
				return nil, errors.New("cannot use multiple '...'")
			}
			c.multiple = true
			s = strings.TrimSuffix(s, "...")
		}
		if s == "_" {
			return []string{}, nil
		} else if strings.HasPrefix(s, "$") {
			key := strings.TrimPrefix(s, "$")
			if d, ok := dict[key]; ok {
				return d, nil
			} else {
				return nil, errors.New(fmt.Sprintf("no key: %s", key))
			}
		} else {
			return []string{s}, nil
		}
	}
	ind := 0
	for _, s := range lis {
		if s == "" {
			continue
		} else if keywordpat.MatchString(s) { // keyword
			fs := keywordpat.FindStringSubmatch(s)
			if fs[1] == "" {
				return c, errors.New(fmt.Sprintf("no keyword: %s", s))
			}
			if _, exist := c.keyword[fs[1]]; exist {
				return c, errors.New(fmt.Sprintf("key %s already exists", fs[1]))
			}
			if fs[2] == "" {
				c.keyword[fs[1]] = []string{}
			} else {
				ws, err := addword(fs[2])
				if err != nil {
					return c, err
				}
				c.keyword[fs[1]] = ws
			}
		} else { // positional
			ws, err := addword(s)
			if err != nil {
				return c, err
			}
			c.positional[ind] = ws
			ind++
		}
	}
	c.positional = c.positional[:ind]
	return c, nil
}

func MustCompile(val string, dict map[string][]string) *Complete {
	c, err := Compile(val, dict)
	if err != nil {
		panic(err.Error())
	}
	return c
}

func (c *Complete) String() string {
	var w bytes.Buffer
	for i, l := range c.positional {
		switch len(l) {
		case 0:
			w.WriteString("_")
		case 1:
			if l[0] == "%g" {
				w.WriteString("filename")
			} else {
				w.WriteString(l[0])
			}
		default:
			w.WriteString("[" + l[0])
			for _, s := range l[1:] {
				w.WriteString("," + s)
			}
			w.WriteString("]")
		}
		if i != len(c.positional)-1 {
			w.WriteString(" ")
		} else {
			if c.multiple {
				w.WriteString("...")
			}
			w.WriteString("\n")
		}
	}
	for k, v := range c.keyword {
		switch len(v) {
		case 0:
			w.WriteString("    -" + k + "\n")
		case 1:
			w.WriteString("    -" + k + "=")
			if v[0] == "%g" {
				w.WriteString("filename\n")
			} else {
				w.WriteString(v[0] + "\n")
			}
		default:
			w.WriteString("    -" + k + "=")
			w.WriteString("[" + v[0])
			for _, s := range v[1:] {
				w.WriteString("," + s)
			}
			w.WriteString("]\n")
		}
	}
	return w.String()
}

func (c *Complete) Complete(val string) []string {
	lis := strings.Split(val, " ")
	pos := len(lis) - 1
	v := lis[pos]
	complete := func(word string, values []string, compf func(string) string) []string {
		l := len(values)
		switch l {
		case 0:
			return []string{val}
		case 1:
			if strings.HasPrefix(values[0], "%g") {
				if !strings.HasSuffix(word, "*") {
					word += "*"
				}
				fs, err := filepath.Glob(filepath.Join(c.directory, word))
				if err != nil {
					return []string{val}
				}
				rtn := make([]string, len(fs))
				i := 0
				for _, f := range fs {
					lis[pos] = compf(f)
					rtn[i] = strings.Join(lis, " ")
					i++
				}
				return rtn[:i]
			} else {
				if strings.HasPrefix(values[0], word) {
					lis[pos] = compf(values[0])
					return []string{strings.Join(lis, " ")}
				} else {
					return []string{val}
				}
			}
		default:
			rtn := make([]string, l)
			i := 0
			for _, f := range values {
				if strings.HasPrefix(f, word) {
					lis[pos] = compf(f)
					rtn[i] = strings.Join(lis, " ")
					i++
				}
			}
			return rtn[:i]
		}
	}
	if keywordarg.MatchString(v) { // keyword
		fs := keywordarg.FindStringSubmatch(v)
		if fs[3] != "" { // keyword: = or == is entred
			key := fs[2]
			var values []string
			for k, v := range c.keyword {
				if k == key {
					values = v
					break
				}
			}
			if values == nil { // keyword: no keyword matched
				return []string{val}
			}
			pre := v[:strings.LastIndex(v, "=")+1]
			return complete(fs[4], values, func(v string) string { return fmt.Sprintf("%s%s", pre, v) })
		} else { // keyword: = nor == is not entred
			key := strings.TrimLeft(v, "-")
			rtn := make([]string, len(c.keyword))
			i := 0
			for k, v := range c.keyword {
				if strings.HasPrefix(k, key) {
					if len(v) == 0 {
						lis[pos] = fmt.Sprintf("%s%s", fs[1], k)
					} else {
						lis[pos] = fmt.Sprintf("%s%s=", fs[1], k)
					}
					rtn[i] = strings.Join(lis, " ")
					i++
				}
			}
			return rtn[:i]
		}
	} else { // positional
		cpos := pos
		for _, s := range lis {
			if keywordarg.MatchString(s) {
				cpos--
			}
		}
		if cpos >= len(c.positional) {
			if c.multiple {
				return complete(v, c.positional[len(c.positional)-1], func(v string) string { return v})
			} else {
				return []string{val}
			}
		}
		return complete(v, c.positional[cpos], func(v string) string { return v })
	}
}

func (c *Complete) CompleteWord(val string) []string {
	lis := c.Complete(val)
	rtn := make([]string, len(lis))
	sp := strings.Split(val, " ")
	pos := len(sp) - 1
	for i, s := range lis {
		rtn[i] = strings.Split(s, " ")[pos]
	}
	return rtn
}

func (c *Complete) Chdir(dir string) {
	c.directory = dir
}

func (c *Complete) Context(val string) Context {
	lis := strings.Split(val, " ")
	pos := len(lis) - 1
	v := lis[pos]
	context := func(values []string) Context {
		switch len(values) {
		case 0:
			return None
		case 1:
			if strings.HasPrefix(values[0], "%g") {
				return FileName
			} else {
				return None
			}
		default:
			return None
		}
	}
	if keywordarg.MatchString(v) { // keyword
		fs := keywordarg.FindStringSubmatch(v)
		if fs[3] != "" { // keyword: = or == is entred
			key := fs[2]
			var values []string
			for k, v := range c.keyword {
				if k == key {
					values = v
					break
				}
			}
			if values == nil { // keyword: no keyword matched
				return UnknownKeyword
			}
			return context(values)
		} else { // keyword: = nor == is not entred
			return Keyword
		}
	} else { // positional
		cpos := pos
		for _, s := range lis {
			if keywordarg.MatchString(s) {
				cpos--
			}
		}
		if cpos >= len(c.positional) {
			return OverSized
		}
		return context(c.positional[cpos])
	}
}
