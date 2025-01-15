package storage

type closer func() error

func (c closer) Close() error {
	return c()
}
