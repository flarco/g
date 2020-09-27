package stacktrace_test

import (
	"github.com/palantir/stacktrace"
)

type PublicObj struct{}
type privateObj struct{}
type ptrObj struct{}

func startDoing() error {
	return stacktrace.NewError("%s %s %s %s", "failed", "to", "start", "doing")
}

func (PublicObj) DoPublic(err error) error {
	return stacktrace.Propagate(err, "")
}

func (PublicObj) doPrivate(err error) error {
	return stacktrace.Propagate(err, "")
}

func (privateObj) DoPublic(err error) error {
	return stacktrace.Propagate(err, "")
}

func (privateObj) doPrivate(err error) error {
	return stacktrace.Propagate(err, "")
}

func (*ptrObj) doPtr(err error) error {
	return stacktrace.Propagate(err, "pointedly")
}

func doClosure(err error) error {
	return func() error {
		return stacktrace.Propagate(err, "so closed")
	}()
}
