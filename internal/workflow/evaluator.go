package workflow

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/apfs-io/apfs/models"
)

// EvaluateIf evaluates a job's if: expression against the current
// ProcessingState. Returns (skip bool, err error).
//
// Supported expression syntax (subset of GitHub Actions):
//
//	${{ <expr> }}
//
// where <expr> is one of:
//
//	<jobID>.outputs.<key>                 — value from a completed job's outputs
//	<jobID>.outputs.<key> <op> <literal>  — comparison: == != < > <= >=
//	<jobID>.status == 'completed'         — job status check
//	true / false                          — literals
//
// The function returns skip=true when the expression evaluates to false,
// meaning the job should be skipped.
func EvaluateIf(expr string, state *models.ProcessingState) (skip bool, err error) {
	if expr == "" {
		return false, nil // no condition → do not skip
	}

	// Unwrap ${{ ... }}
	inner := strings.TrimSpace(expr)
	if strings.HasPrefix(inner, "${{") && strings.HasSuffix(inner, "}}") {
		inner = strings.TrimSpace(inner[3 : len(inner)-2])
	}

	result, err := evalExpr(inner, state)
	if err != nil {
		return false, fmt.Errorf("workflow evaluator: %w", err)
	}
	return !result, nil
}

// evalExpr evaluates a boolean expression string.
func evalExpr(expr string, state *models.ProcessingState) (bool, error) {
	expr = strings.TrimSpace(expr)
	switch expr {
	case "true":
		return true, nil
	case "false":
		return false, nil
	}

	// Binary comparison: <lhs> <op> <rhs>
	for _, op := range []string{"==", "!=", "<=", ">=", "<", ">"} {
		idx := strings.Index(expr, op)
		if idx < 0 {
			continue
		}
		lhsStr := strings.TrimSpace(expr[:idx])
		rhsStr := strings.TrimSpace(expr[idx+len(op):])

		lval, err := resolveValue(lhsStr, state)
		if err != nil {
			return false, err
		}
		rval, err := resolveValue(rhsStr, state)
		if err != nil {
			return false, err
		}
		return compareValues(lval, op, rval)
	}

	// No operator: treat as truthy value
	v, err := resolveValue(expr, state)
	if err != nil {
		return false, err
	}
	return isTruthy(v), nil
}

// resolveValue resolves a value token. Supports:
//   - quoted strings: 'foo' or "foo"
//   - numeric literals
//   - boolean literals
//   - job references: jobID.outputs.key, jobID.status
func resolveValue(token string, state *models.ProcessingState) (any, error) {
	token = strings.TrimSpace(token)

	// Quoted string
	if (strings.HasPrefix(token, "'") && strings.HasSuffix(token, "'")) ||
		(strings.HasPrefix(token, "\"") && strings.HasSuffix(token, "\"")) {
		return token[1 : len(token)-1], nil
	}

	// Numeric
	if n, err := strconv.ParseFloat(token, 64); err == nil {
		return n, nil
	}

	// Boolean
	switch token {
	case "true":
		return true, nil
	case "false":
		return false, nil
	case "null", "nil":
		return nil, nil
	}

	// Job reference: <jobID>.<field>[.<subfield>]
	if state != nil {
		parts := strings.SplitN(token, ".", 3)
		if len(parts) >= 2 {
			jobID := parts[0]
			js, ok := state.Jobs[jobID]
			if !ok {
				return nil, nil
			}
			switch parts[1] {
			case "status":
				return string(js.Status), nil
			case "outputs":
				if len(parts) == 3 && js.Outputs != nil {
					return js.Outputs[parts[2]], nil
				}
			}
		}
	}

	return nil, fmt.Errorf("unresolved value: %q", token)
}

// compareValues compares lval op rval. Both values are normalised to float64
// when possible for numeric comparisons.
func compareValues(lval any, op string, rval any) (bool, error) {
	// Try numeric comparison
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

	// String comparison
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
