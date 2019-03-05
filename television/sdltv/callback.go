package sdltv

// callback is used to wrap functions supplied to RequestCallback()

type callback struct {
	channel  chan func()
	function func()
}

func (cb *callback) dispatch() {
	if cb.function == nil {
		return
	}

	if cb.channel != nil {
		cb.channel <- cb.function
	} else {
		cb.function()
	}
}
