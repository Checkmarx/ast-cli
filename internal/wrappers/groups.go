package wrappers

type GroupsWrapper interface {
	Get(groupName string) ([]Group, error)
}
