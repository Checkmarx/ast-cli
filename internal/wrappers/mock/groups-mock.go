package mock

type GroupsMockWrapper struct {
}
type Group struct{}

func (g *GroupsMockWrapper) Get(_ string) ([]Group, error) {
	return nil, nil
}
