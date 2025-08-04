package kafka

// TopicType тип для перечисления типов топиков
type TopicType string

// перечисление типов топиков
const (
	Multiple TopicType = "multiple"
	Consume  TopicType = "consume"
	Produce  TopicType = "produce"
)
