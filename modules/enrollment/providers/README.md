# Academic placements

Enrollment owns a student's grade and optional course or programme for an academic year. A school may create one placement per student and year. A withdrawal preserves the placement history and its unique key. The existing Enrollment workspace remains available for school tasks.

The API authenticates the bearer session and takes its school from the principal. It requires both `students.read` and `workspace.read` to list placements, or both `students.write` and `workspace.write` to register or withdraw one.

| Request | Input | Result |
| --- | --- | --- |
| `GET /api/v1/enrollment/placements` | Optional `studentId`, `limit` (1 to 100, default 25), and `offset` | `{items,total}` including active and withdrawn records, ordered by newest academic year then ID. |
| `POST /api/v1/enrollment/placements` | `{"studentId":"…","academicYear":"2026-2027","grade":"Grade 8","course":"General"}`. Course is optional. | 201 for a new placement. An identical retry of an active placement returns 200 and its original ID. |
| `PATCH /api/v1/enrollment/placements/{id}` | `{"status":"withdrawn","version":1}` | 200 with status `withdrawn` and version incremented. An outdated version returns 409. |

An academic year must contain two consecutive four-digit years. Grade accepts 1 to 48 characters and course accepts up to 120. Reusing an existing student/year with a different grade or course returns 409. Retrying a withdrawn placement returns 409. A student missing from the verified school returns 404. A new placement for an archived student returns 409.

Enrollment asks the public Students directory to verify active status before a new placement commits. Registration, its status check, and the outbox event share the school's transaction. A PostgreSQL lookup holds a row lock until commit. SQLite already holds its write lock after reserving the placement key. A later archive preserves historical placements, and retrying an earlier successful registration returns the original active placement.

`enrollment.placed.v1` and `enrollment.withdrawn.v1` each carry the placement ID, student ID, academic year, status, version, and actor ID. Their envelope supplies the school and event ID. An idempotent retry produces no additional event.

The business rules live in `app/models/placement.go`. The Students dependency and use cases live in `app/services/placements.go`, while the adapters own HTTP and persistence. The feature suite in `tests/integration/enrollment_placements_test.go` exercises both modules through their real providers with SQLite and optionally PostgreSQL when `EDUPILOT_TEST_POSTGRES_DSN` is set.
