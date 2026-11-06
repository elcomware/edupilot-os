package scope

type OwnerKind string

const (
	TenantOwner       OwnerKind = "tenant"
	OrganizationOwner OwnerKind = "organization"
	SchoolOwner       OwnerKind = "school"
	CampusOwner       OwnerKind = "campus"
)

// Owner stores an exact, typed ancestry. Build it from authoritative directory
// output and recheck permissions in the scoped transaction.
type Owner struct {
	kind         OwnerKind
	tenant       TenantID
	organization OrganizationID
	school       SchoolID
	campus       CampusID
}

func (c Chain) Owner(kind OwnerKind) (Owner, error) {
	owner := Owner{kind: kind, tenant: c.Tenant}
	switch kind {
	case CampusOwner:
		owner.campus = c.Campus
		fallthrough
	case SchoolOwner:
		owner.school = c.School
		fallthrough
	case OrganizationOwner:
		owner.organization = c.Organization
	case TenantOwner:
	default:
		return Owner{}, ErrUnknown
	}
	if !owner.Valid() {
		return Owner{}, ErrUnknown
	}
	return owner, nil
}
func (o Owner) Valid() bool {
	if o.tenant.IsZero() {
		return false
	}
	switch o.kind {
	case TenantOwner:
		return true
	case OrganizationOwner:
		return !o.organization.IsZero()
	case SchoolOwner:
		return !o.organization.IsZero() && !o.school.IsZero()
	case CampusOwner:
		return !o.organization.IsZero() && !o.school.IsZero() && !o.campus.IsZero()
	default:
		return false
	}
}
func (o Owner) Kind() OwnerKind              { return o.kind }
func (o Owner) Tenant() TenantID             { return o.tenant }
func (o Owner) Organization() OrganizationID { return o.organization }
func (o Owner) School() SchoolID             { return o.school }
func (o Owner) Campus() CampusID             { return o.campus }
