package contracts

// Lifecycle events are versioned so existing subscribers can continue reading
// their published payload after the student record evolves.
const (
	RegisteredV1 = "students.registered.v1"
	UpdatedV1    = "students.updated.v1"
	ArchivedV1   = "students.archived.v1"
)

// LifecycleEventV1 contains no student names. School scope and event identity
// are carried by the outbox envelope.
type LifecycleEventV1 struct {
	StudentID string `json:"studentId"`
	Version   int64  `json:"version"`
	Status    string `json:"status"`
	ActorID   string `json:"actorId"`
}
