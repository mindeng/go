package minargs

type StringFlag struct {
	set   bool
	value string
}

func (sf *StringFlag) Set(x string) error {
	sf.value = x
	sf.set = true
	return nil
}

func (sf *StringFlag) String() string {
	return sf.value
}

func (sf *StringFlag) Provided() bool {
	return sf.set
}
