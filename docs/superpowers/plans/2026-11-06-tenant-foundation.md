# Tenant foundation delivery plan

Use the [tenancy and placement specification](../../product/tenancy-and-placement.md) and [release board](../../release/status.md).

| Order | Acceptance |
|---|---|
| 1. Installation | Register Core, Auth and Tenancy through Goravel. Apply module-owned Blueprint schemas. Verify the local receipt and physical marker on each pool. Refuse unknown nonempty databases. |
| 2. Current-school access | Authenticate through the school guard. Verify the school and current grants on each protected request, including revocation. |
| 3. Hierarchy | Persist group, organization, school, campus and additional scoped relationships. Verify topology, versioning, uniqueness and isolation on both engines. |
| 4. Context and grants | Resolve permitted contexts, implement scoped grants and switch context across web, desktop and mobile. Cover denial, expiry, inheritance and stale sessions. |
| 5. Tenant relocation | Validate source and target, fence writers, verify backup restoration, resume interrupted copies, compare records and events, then switch placement atomically. Test reverse moves on synthetic tenants. |
| 6. Administration | Deliver platform and tenant administrator pages with separate authorization and real API-backed controls. |
