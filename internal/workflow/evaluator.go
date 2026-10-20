package workflow

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/apfs-io/apfs/models"
)

// EvalContext is the data an if: expression can reference.
type EvalContext struct {
	State *models.ProcessingState
	Meta  *models.Meta
}

// EvaluateIf evaluates a job's if: expression. Returns (skip bool, err error);
// skip is true when the expression evaluates to false.
//
// Every ${{ ... }} in the string is replaced by its content, so both
// "${{ a > 1 }}" and "${{ a }} > 1" work. Grammar (lowest precedence first):
//
//	or         = and { "||" and }
//	and        = unary { "&&" unary }
//	unary      = "!" unary | comparison
//	comparison = operand [ ("=="|"!="|"<"|"<="|">"|">=") operand ]
//	operand    = "(" or ")" | literal | reference
//
// References:
//
//	main.<field>               — the original file (width, height, duration,
//	                             size, bitrate, codec, content_type, type, ext)
//	main.attributes.<key>      — original file attribute
//	jobs.<id>.status           — job status ("completed", "skipped", …)
//	jobs.<id>.outputs.<key>    — job output value
//	<id>.status, <id>.outputs.<key> — short forms of the above
//
// A comparison with a missing (nil) value is false, except == / != against null.
func EvaluateIf(expr string, ctx EvalContext) (skip bool, err error) {
	if strings.TrimSpace(expr) == "" {
		return false, nil
	}
	tokens, err := tokenize(unwrapTemplates(expr))
	if err != nil {
		return false, fmt.Errorf("workflow evaluator: %w", err)
	}
	p := &exprParser{tokens: tokens, ctx: ctx}
	v, err := p.parseOr()
	if err == nil && p.pos < len(p.tokens) {
		err = fmt.Errorf("unexpected token %q", p.tokens[p.pos].text)
	}
	if err != nil {
		return false, fmt.Errorf("workflow evaluator: %w", err)
	}
	return !isTruthy(v), nil
}

var templateRe = regexp.MustCompile(`\$\{\{(.*?)\}\}`)

func unwrapTemplates(expr string) string {
	return templateRe.ReplaceAllString(expr, " $1 ")
}

// ── Tokenizer ────────────────────────────────────────────────────────────────

type tokenKind int

const (
	tokIdent tokenKind = iota
	tokNumber
	tokString
	tokOp
	tokLParen
	tokRParen
)

type token struct {
	kind tokenKind
	text string
}

func tokenize(s string) ([]token, error) {
	var out []token
	for i := 0; i < len(s); {
		c := s[i]
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			i++
		case c == '(':
			out = append(out, token{tokLParen, "("})
			i++
		case c == ')':
			out = append(out, token{tokRParen, ")"})
			i++
		case c == '\'' || c == '"':
			end := strings.IndexByte(s[i+1:], c)
			if end < 0 {
				return nil, fmt.Errorf("unterminated string at %d", i)
			}
			out = append(out, token{tokString, s[i+1 : i+1+end]})
			i += end + 2
		case strings.ContainsRune("=!<>&|", rune(c)):
			two := ""
			if i+1 < len(s) {
				two = s[i : i+2]
			}
			switch two {
			case "==", "!=", "<=", ">=", "&&", "||":
				out = append(out, token{tokOp, two})
				i += 2
				continue
			}
			if c == '<' || c == '>' || c == '!' {
				out = append(out, token{tokOp, string(c)})
				i++
				continue
			}
			return nil, fmt.Errorf("unexpected %q at %d", c, i)
		case isNumberStart(s, i):
			j := i + 1
			for j < len(s) && (isDigit(s[j]) || s[j] == '.') {
				j++
			}
			if j < len(s) && (isIdentChar(s[j]) || s[j] == '-') && c != '-' {
				// Job id starting with digits, e.g. 1080p.status.
				for j < len(s) && (isIdentChar(s[j]) || s[j] == '.' || s[j] == '-') {
					j++
				}
				out = append(out, token{tokIdent, s[i:j]})
				i = j
				continue
			}
			out = append(out, token{tokNumber, s[i:j]})
			i = j
		case isIdentChar(c):
			j := i
			for j < len(s) && (isIdentChar(s[j]) || s[j] == '.' || s[j] == '-') {
				j++
			}
			out = append(out, token{tokIdent, s[i:j]})
			i = j
		default:
			return nil, fmt.Errorf("unexpected %q at %d", c, i)
		}
	}
	return out, nil
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

func isIdentChar(c byte) bool {
	return c == '_' || isDigit(c) || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isNumberStart(s string, i int) bool {
	if isDigit(s[i]) {
		return true
	}
	return s[i] == '-' && i+1 < len(s) && isDigit(s[i+1])
}

// ── Parser ───────────────────────────────────────────────────────────────────

type exprParser struct {
	tokens []token
	pos    int
	ctx    EvalContext
}

func (p *exprParser) peekOp(ops ...string) (string, bool) {
	if p.pos >= len(p.tokens) || p.tokens[p.pos].kind != tokOp {
		return "", false
	}
	for _, op := range ops {
		if p.tokens[p.pos].text == op {
			return op, true
		}
	}
	return "", false
}

func (p *exprParser) parseOr() (any, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for {
		if _, ok := p.peekOp("||"); !ok {
			return left, nil
		}
		p.pos++
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = isTruthy(left) || isTruthy(right)
	}
}

func (p *exprParser) parseAnd() (any, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	for {
		if _, ok := p.peekOp("&&"); !ok {
			return left, nil
		}
		p.pos++
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		left = isTruthy(left) && isTruthy(right)
	}
}

func (p *exprParser) parseUnary() (any, error) {
	if _, ok := p.peekOp("!"); ok {
		p.pos++
		v, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return !isTruthy(v), nil
	}
	return p.parseComparison()
}

func (p *exprParser) parseComparison() (any, error) {
	left, err := p.parseOperand()
	if err != nil {
		return nil, err
	}
	op, ok := p.peekOp("==", "!=", "<", "<=", ">", ">=")
	if !ok {
		return left, nil
	}
	p.pos++
	right, err := p.parseOperand()
	if err != nil {
		return nil, err
	}
	return compareValues(left, op, right)
}

func (p *exprParser) parseOperand() (any, error) {
	if p.pos >= len(p.tokens) {
		return nil, fmt.Errorf("unexpected end of expression")
	}
	tok := p.tokens[p.pos]
	p.pos++
	switch tok.kind {
	case tokLParen:
		v, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		if p.pos >= len(p.tokens) || p.tokens[p.pos].kind != tokRParen {
			return nil, fmt.Errorf("missing )")
		}
		p.pos++
		return v, nil
	case tokString:
		return tok.text, nil
	case tokNumber:
		n, err := strconv.ParseFloat(tok.text, 64)
		if err != nil {
			return nil, fmt.Errorf("bad number %q", tok.text)
		}
		return n, nil
	case tokIdent:
		switch tok.text {
		case "true":
			return true, nil
		case "false":
			return false, nil
		case "null", "nil":
			return nil, nil
		}
		return resolveRef(tok.text, p.ctx)
	default:
		return nil, fmt.Errorf("unexpected token %q", tok.text)
	}
}

// ── References ───────────────────────────────────────────────────────────────

// resolveRef resolves main.*, jobs.<id>.* and <id>.* references.
func resolveRef(ref string, ctx EvalContext) (any, error) {
	parts := strings.Split(ref, ".")
	if parts[0] == "main" {
		return resolveMain(parts[1:], ctx.Meta), nil
	}
	if parts[0] == "jobs" && len(parts) >= 3 {
		parts = parts[1:]
	}
	if len(parts) >= 2 && (parts[1] == "status" || parts[1] == "outputs") {
		if ctx.State == nil {
			return nil, nil
		}
		js, ok := ctx.State.Jobs[parts[0]]
		if !ok || js == nil {
			return nil, nil
		}
		if parts[1] == "status" {
			return string(js.Status), nil
		}
		if len(parts) == 3 && js.Outputs != nil {
			return normalizeValue(js.Outputs[parts[2]]), nil
		}
		return nil, nil
	}
	return nil, fmt.Errorf("unresolved value: %q", ref)
}

func resolveMain(fields []string, meta *models.Meta) any {
	if meta == nil || len(fields) == 0 {
		return nil
	}
	m := &meta.Main
	positive := func(v int64) any {
		if v <= 0 {
			return nil
		}
		return float64(v)
	}
	text := func(v string) any {
		if v == "" {
			return nil
		}
		return v
	}
	switch fields[0] {
	case "width":
		return positive(int64(m.Width))
	case "height":
		return positive(int64(m.Height))
	case "duration":
		return positive(int64(m.Duration))
	case "size":
		return positive(m.Size)
	case "bitrate":
		if n, err := strconv.ParseFloat(m.Bitrate, 64); err == nil {
			return n
		}
		return text(m.Bitrate)
	case "codec":
		return text(m.Codec)
	case "content_type":
		return text(m.ContentType)
	case "type":
		return text(string(m.Type))
	case "ext":
		return text(m.ObjectTypeExt())
	case "attributes":
		if len(fields) < 2 || m.Attributes == nil {
			return nil
		}
		return normalizeValue(m.Attributes[strings.Join(fields[1:], ".")])
	}
	return nil
}

// normalizeValue decodes raw JSON outputs (target-meta) into plain values.
func normalizeValue(v any) any {
	switch val := v.(type) {
	case json.RawMessage:
		var out any
		if json.Unmarshal(val, &out) == nil {
			return out
		}
		return string(val)
	case []byte:
		return string(val)
	}
	return v
}

// ── Values ───────────────────────────────────────────────────────────────────

// compareValues compares lval op rval. Numbers compare numerically; anything
// else compares as strings (== / != only). A nil side is false except for
// == / != against another value.
func compareValues(lval any, op string, rval any) (bool, error) {
	if lval == nil || rval == nil {
		switch op {
		case "==":
			return lval == nil && rval == nil, nil
		case "!=":
			return (lval == nil) != (rval == nil), nil
		default:
			return false, nil
		}
	}

	ln, lok := toFloat64(lval)
	rn, rok := toFloat64(rval)
	if lok && rok {
		switch op {
		case "==":
			return ln == rn, nil
		case "!=":
			return ln != rn, nil
		case "<":
			return ln < rn, nil
		case ">":
			return ln > rn, nil
		case "<=":
			return ln <= rn, nil
		case ">=":
			return ln >= rn, nil
		}
	}

	ls := fmt.Sprintf("%v", lval)
	rs := fmt.Sprintf("%v", rval)
	switch op {
	case "==":
		return ls == rs, nil
	case "!=":
		return ls != rs, nil
	default:
		return false, fmt.Errorf("unsupported operator %q for non-numeric values", op)
	}
}

func isTruthy(v any) bool {
	if v == nil {
		return false
	}
	switch val := v.(type) {
	case bool:
		return val
	case string:
		return val != "" && val != "false" && val != "0"
	case float64:
		return val != 0
	case int:
		return val != 0
	case int64:
		return val != 0
	}
	return true
}

func toFloat64(v any) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case string:
		n, err := strconv.ParseFloat(val, 64)
		return n, err == nil
	}
	return 0, false
}
