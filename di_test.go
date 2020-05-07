package injection

import (
	"log"
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
	LowLevelComponent iLowLevel        `dep:""`
	WithConstructor   *withConstructor `dep:""`
	Provided          *providedDep     `dep:""`
	ZeroValue         *zeroValueDep    `dep:""`
}

func (tlc topLevelComponent) Constructor() *topLevelComponent {
	return &tlc
}

type providedDep struct {
	name   string
	number int
}

type zeroValueDep struct {
	name   string
	number int
}

type withConstructor struct {
	number int
}

func (wc withConstructor) Constructor() *withConstructor {
	return &withConstructor{13}
}

func TestExample(t *testing.T) {
	dic := NewDIContainer()
	tlc := new(topLevelComponent)
	surplus := &providedDep{name: "surplus", number: 9}

	dic.Register(tlc)
	dic.Register(surplus)
	dic.Register(new(lowLevelComponent))
	if err := dic.Populate(); err != nil {
		log.Fatal(err)
	}

	tlc = tlc.Constructor()
	assert.Equal(t, 12, tlc.LowLevelComponent.Counter())
	assert.Equal(t, 9, tlc.Provided.number)
	assert.Equal(t, 0, tlc.ZeroValue.number)
	assert.Equal(t, 13, tlc.WithConstructor.number)

	// Get by qualified name
	tlcPtr, _ := dic.Get("github.com/pot-code/go-injection/topLevelComponent")
	tlc = tlcPtr.(*topLevelComponent)
	assert.Equal(t, 12, tlc.LowLevelComponent.Counter())

	// Get by type
	tlcPtr, _ = dic.Get(topLevelComponent{})
	tlc = tlcPtr.(*topLevelComponent)
	assert.Equal(t, 12, tlc.LowLevelComponent.Counter())
}
