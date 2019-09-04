package services

import (
	"github.com/gernest/wow"
	"github.com/gernest/wow/spin"
	"log"
	"os"
)

type Stepper struct {
	target *wow.Wow
	step   string
}

func NewStepper(step string) Stepper {
	s := Stepper{
		step:   step,
		target: wow.New(os.Stdout, spin.Get(spin.Dots2), " "+step),
	}
	s.target.Start()
	return s
}

func (self *Stepper) Info(text string) {
	self.target.Text(text)
}

func (self *Stepper) Complete() {
	self.target.PersistWith(spin.Get(spin.Line), " "+self.step+": Done")

}
func (self *Stepper) Fail(err string) {
	self.target.PersistWith(spin.Get(spin.Star), " "+self.step+": Fail")
	log.Fatal(err)
}
func (self *Stepper) FailWithError(message string, err error) {
	self.target.PersistWith(spin.Get(spin.Star), " "+self.step+": Fail "+message)
	log.Fatal(err)
}
