package desktop

import "go.lsp.dev/protocol"

func diagnosticSeverity(severity protocol.DiagnosticSeverity) string {
	switch severity {
	case protocol.DiagnosticSeverityError:
		return "error"
	case protocol.DiagnosticSeverityWarning:
		return "warning"
	case protocol.DiagnosticSeverityInformation:
		return "info"
	default:
		// Unknown severity defaults to the lowest level rather than the
		// highest — we never want to surface an unknown classification as
		// a blocking error.
		return "info"
	}
}

func diagnosticCode(code any) string {
	switch v := code.(type) {
	case string:
		return v
	default:
		return ""
	}
}
