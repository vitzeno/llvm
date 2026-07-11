package parser

import "fmt"

// Diagnostic is a diagnostic message
type Diagnostic struct {
	Position
	Message string
	Severity
}

// Severity is the severity of the diagnostic message
type Severity int

const (
	Error       Severity = 1
	Warning     Severity = 2
	Information Severity = 3
	Hint        Severity = 4
)

func (s Severity) String() string {
	switch s {
	case Error:
		return "error"
	case Warning:
		return "warning"
	case Information:
		return "info"
	case Hint:
		return "hint"
	default:
		return "unknown"
	}
}

func (d *Diagnostic) String() string {
	if d.Line == 0 {
		return fmt.Sprintf("%s: %s", d.Severity, d.Message)
	}
	return fmt.Sprintf("%s Line: %d, Col: %d: %s", d.Severity, d.Line, d.Col, d.Message)
}
