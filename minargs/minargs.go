package minargs

type StringArg struct {
	set   bool
	value string
}

func (sf *StringArg) Set(x string) error {
	sf.value = x
	sf.set = true
	return nil
}

func (sf *StringArg) String() string {
	return sf.value
}

func (sf *StringArg) Provided() bool {
	return sf.set
}
