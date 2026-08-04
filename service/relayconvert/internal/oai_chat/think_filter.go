package openaicompat

import "strings"

const (
	thinkOpenTag  = "<think>"
	thinkCloseTag = "</think>"
)

type ThinkTagStreamFilter struct {
	pending string
	inThink bool
}

func NewThinkTagStreamFilter() *ThinkTagStreamFilter {
	return &ThinkTagStreamFilter{}
}

func (f *ThinkTagStreamFilter) Push(delta string) string {
	if f == nil || delta == "" {
		return delta
	}
	f.pending += delta
	return f.drain(false)
}

func (f *ThinkTagStreamFilter) Flush() string {
	if f == nil {
		return ""
	}
	return f.drain(true)
}

func (f *ThinkTagStreamFilter) drain(flush bool) string {
	var out strings.Builder
	for f.pending != "" {
		if f.inThink {
			closeIdx := strings.Index(f.pending, thinkCloseTag)
			if closeIdx < 0 {
				if flush || len(f.pending) > len(thinkCloseTag)-1 {
					keep := longestSuffixPrefixLen(f.pending, thinkCloseTag)
					if keep > 0 && !flush {
						f.pending = f.pending[len(f.pending)-keep:]
					} else {
						f.pending = ""
					}
				}
				break
			}
			f.pending = f.pending[closeIdx+len(thinkCloseTag):]
			f.inThink = false
			continue
		}

		openIdx := strings.Index(f.pending, thinkOpenTag)
		if openIdx < 0 {
			emitLen := len(f.pending)
			if !flush {
				emitLen -= longestSuffixPrefixLen(f.pending, thinkOpenTag)
			}
			if emitLen <= 0 {
				break
			}
			out.WriteString(f.pending[:emitLen])
			f.pending = f.pending[emitLen:]
			continue
		}

		out.WriteString(f.pending[:openIdx])
		f.pending = f.pending[openIdx+len(thinkOpenTag):]
		f.inThink = true
	}
	return out.String()
}

func longestSuffixPrefixLen(s string, prefix string) int {
	max := len(prefix) - 1
	if len(s) < max {
		max = len(s)
	}
	for n := max; n > 0; n-- {
		if strings.HasPrefix(prefix, s[len(s)-n:]) {
			return n
		}
	}
	return 0
}

func StripThinkTags(content string) string {
	filter := NewThinkTagStreamFilter()
	return filter.Push(content) + filter.Flush()
}
