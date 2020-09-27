package stacktrace_test

import (
	"github.com/palantir/stacktrace"
)

const (
	EcodeInvalidVillain = stacktrace.ErrorCode(iota)
	EcodeNoSuchPseudo
	EcodeNotFastEnough
	EcodeTimeIsIllusion
	EcodeNotImplemented
)
