package types

func (b *Binding) GetCommandID() string {
	if b.CommandIDFunc != nil {
		return b.CommandIDFunc()
	}
	return b.CommandID
}

func (b *Binding) GetVerb() string {
	if b.VerbFunc != nil {
		return b.VerbFunc()
	}
	return b.Verb
}

func (self *MenuItem) GetCommandID() string {
	if self.CommandIDFunc != nil {
		return self.CommandIDFunc()
	}
	return self.CommandID
}

func (self *MenuItem) GetVerb() string {
	if self.VerbFunc != nil {
		return self.VerbFunc()
	}
	return self.Verb
}
