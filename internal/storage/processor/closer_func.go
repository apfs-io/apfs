package processor

type closer func() error

func (c closer) Close() error {
	return c()
}
