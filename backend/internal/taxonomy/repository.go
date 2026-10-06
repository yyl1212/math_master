package taxonomy

import "context"

type Repository interface {
	ListTopics(context.Context, Query) (Page[TopicSummary], error)
	ReadTopic(context.Context, string) (TopicDetail, error)
	ListTopicKnowledge(context.Context, string, Query) (Page[KnowledgeSummary], error)
}
