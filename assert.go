package main

func assert(err error) {
	if err != nil {
		panic(err)
	}
}

func assertDrain(errCh <-chan error) {
	for err := range errCh {
		assert(err)
	}
}
