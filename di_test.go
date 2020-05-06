package injection

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type iLowLevel interface {
	Counter() int
}

type lowLevelComponent struct {
	counter int
}

func (llc lowLevelComponent) Constructor() *lowLevelComponent {
	return &lowLevelComponent{12}
}

func (llc *lowLevelComponent) Counter() int {
	return llc.counter
}

type topLevelComponent struct {
	LowLevelComponent iLowLevel `dep:""`
}

func (tlc topLevelComponent) Constructor() *topLevelComponent {
	return &topLevelComponent{tlc.LowLevelComponent}
}

func TestExample(t *testing.T) {
	dic := NewDIContainer()

	tlc := new(topLevelComponent)
	dic.Register(new(lowLevelComponent))
	dic.Register(tlc)
	dic.Populate()

	tlc = tlc.Constructor()
	assert.Equal(t, 12, tlc.LowLevelComponent.Counter())

	// use Get
	tlcPtr, _ := dic.Get("github.com/pot-code/go-injection/topLevelComponent")
	tlc = tlcPtr.(*topLevelComponent)
	assert.Equal(t, 12, tlc.LowLevelComponent.Counter())
}
