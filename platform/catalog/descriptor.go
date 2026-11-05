// Package catalog defines public workspace metadata. It knows nothing about
// Goravel or any school module's database, HTTP routes or business rules.
package catalog

type Title struct {
 EN string `json:"en"`
 FR string `json:"fr"`
}
type Descriptor struct {
 Key string `json:"key"`
 Group string `json:"group"`
 Title Title `json:"title"`
 Capabilities []string `json:"capabilities"`
}
