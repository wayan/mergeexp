package merger

type MergeRef interface {
	Sha() string
	Name() string
}
