package knowledgeadmin

// Service provides one bounded upload validator shared by all request handlers.
type Service struct {
	Repository Repository
	uploadSlot chan struct{}
}

func NewService(r Repository) *Service {
	return &Service{Repository: r, uploadSlot: make(chan struct{}, 1)}
}
