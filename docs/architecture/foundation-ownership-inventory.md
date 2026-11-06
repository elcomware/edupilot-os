# Group tenancy schema inventory

These additive tables are empty when the starter is upgraded to this schema version. The offline data upgrade and runtime cutover have separate acceptance gates. The source tables and historical migration signatures remain available for verification.

| Surface | Owner | Boundary | Identity and fields | Transfer or retention | Implementation |
|---|---|---|---|---|---|
| tenants | platform/database | control | tenant UUID, slug, state, version | Retain stable identity and lifecycle in control during moves. | platform/database/foundation_schema.go |
| database_pools | platform/database | control | pool UUID, config name, marker, driver, dedicated owner | Keep immutable physical registrations in control. Provision a fresh identity after retirement. | platform/database/foundation_schema.go |
| tenant_placements | platform/database | control | tenant UUID, pool UUID, ownership kind, epoch | Change the single active placement using a verified fenced cutover. | platform/database/foundation_schema.go |
| tenant_upgrade_runs | platform/database | control | run UUID, manifest and source hashes, stage, revision | Persist upgrade decisions across retries. Verify the manifest before resume. | platform/database/foundation_schema.go |
| tenant_upgrade_checkpoints | platform/database | control | run, pool, step, digest | Reconcile against tenant-local copy receipts. | platform/database/foundation_schema.go |
| legacy_scope_mappings | platform/database | control | old school ID, source pool and marker, distinct tenant/org/school/campus | Allocate one tenant per starter school. Preserve old references. | platform/database/foundation_schema.go |
| operator_audit_v2 | platform/database | control | operator, action, reason, tenant, run, instant | Keep operator actions in the control audit history. | platform/database/foundation_schema.go |
| identity_subjects_v2 | platform/identity | control/restricted | subject UUID, state | Never merge subjects solely by address. | platform/identity/foundation_schema.go |
| identity_memberships_v2 | platform/identity | control/restricted | tenant, subject, membership, grant version | Recheck active membership and grant version on every operation. | platform/identity/foundation_schema.go |
| identity_credentials_v2 | platform/identity | control/restricted | tenant, membership, normalized email, password hash | Keep one canonical credential per membership in control. Exclude hashes from reports. | platform/identity/foundation_schema.go |
| identity_login_aliases_v2 | platform/identity | control/restricted | tenant, realm, alias, membership | Keep legacy logins mapped to the same canonical credential. | platform/identity/foundation_schema.go |
| legacy_subject_mappings | platform/identity | control/restricted | old school and user, tenant, membership, subject | Retain explicit mappings, including collisions in old user identifiers. | platform/identity/foundation_schema.go |
| identity_context_index_v2 | platform/identity | control/restricted | tenant, school, organization, label, hierarchy version | Rebuild from authoritative topology. Index membership alone cannot grant access. | platform/identity/foundation_schema.go |
| identity_grants_v2 | platform/identity | control/restricted | tenant, membership, permission, explicit owner, expiration | Restrict initial resource owners to tenant or one indexed school. | platform/identity/foundation_schema.go |
| identity_sessions_v2 | platform/identity | control/restricted | token hash, subject, membership, tenant, context and versions | Keep bearer hashes private. Recheck authority, placement and scope before use. | platform/identity/foundation_schema.go |
| tenant_gates | modules/tenancy | tenant/data | tenant, pool, placement epoch, hierarchy revision, state | Validate placement locally in the transaction. Recreate gated state on the destination. | modules/tenancy/migrations/topology.go |
| tenant_upgrade_receipts | modules/tenancy | tenant/data | tenant, run, stage, batch, canonical digest, rows | Commit each receipt with its copied rows and reconcile against the control journal. | modules/tenancy/migrations/topology.go |
| tenancy_organizations | modules/tenancy | tenant/data | tenant and org, optional same-tenant parent | Transfer parent-first. Reject cycles during domain restore. | modules/tenancy/migrations/topology.go |
| tenancy_schools | modules/tenancy | tenant/data | tenant and school, organization, code | Transfer with the whole tenant. Preserve the institution ID across moves. | modules/tenancy/migrations/topology.go |
| tenancy_campuses | modules/tenancy | tenant/data | tenant and campus, school, local code, IANA zone | Preserve school-local codes and timezones. | modules/tenancy/migrations/topology.go |
| students_records_v2 | modules/students | school/private | tenant, school and original record ID | Names, status and original version. | modules/students/migrations/ownership_v2.go |
| enrollment_placements_v2 | modules/enrollment | school/private | tenant, school and original record ID | Placement, student ID, academic year, status and version. | modules/enrollment/migrations/ownership_v2.go |
| documents_metadata_v2 | modules/documents | school/private | tenant, school and original record ID | Private storage reference, path, format, content hash and byte length. Verify original and destination bytes. | modules/documents/migrations/metadata_v2.go |
| edu_outbox_v2 | platform/outbox | school/private | tenant, school and original record ID | Stable tenant/event key, payload version, lease, retry and delivery state. | platform/outbox/schema_v2.go |
| edu_audit_inbox_v2 | platform/outbox | school/private | tenant, school and original record ID | Stable tenant/event key and original receipt time. | platform/outbox/schema_v2.go |
| ws_tenancy_v2 | modules/tenancy | school/private | tenant, school and work item ID | Preserve original ID, private title, state and version. | platform/workspace/sqlstore/ownership_v2.go |
| ws_identity_v2 | modules/identity | school/private | tenant, school and work item ID | Preserve original ID, private title, state and version. | platform/workspace/sqlstore/ownership_v2.go |
| ws_users_v2 | modules/users | school/private | tenant, school and work item ID | Preserve original ID, private title, state and version. | platform/workspace/sqlstore/ownership_v2.go |
| ws_permissions_v2 | modules/permissions | school/private | tenant, school and work item ID | Preserve original ID, private title, state and version. | platform/workspace/sqlstore/ownership_v2.go |
| ws_school_settings_v2 | modules/school_settings | school/private | tenant, school and work item ID | Preserve original ID, private title, state and version. | platform/workspace/sqlstore/ownership_v2.go |
| ws_subscriptions_v2 | modules/subscriptions | school/private | tenant, school and work item ID | Preserve original ID, private title, state and version. | platform/workspace/sqlstore/ownership_v2.go |
| ws_admissions_v2 | modules/admissions | school/private | tenant, school and work item ID | Preserve original ID, private title, state and version. | platform/workspace/sqlstore/ownership_v2.go |
| ws_guardians_v2 | modules/guardians | school/private | tenant, school and work item ID | Preserve original ID, private title, state and version. | platform/workspace/sqlstore/ownership_v2.go |
| ws_enrollment_v2 | modules/enrollment | school/private | tenant, school and work item ID | Preserve original ID, private title, state and version. | platform/workspace/sqlstore/ownership_v2.go |
| ws_attendance_v2 | modules/attendance | school/private | tenant, school and work item ID | Preserve original ID, private title, state and version. | platform/workspace/sqlstore/ownership_v2.go |
| ws_discipline_v2 | modules/discipline | school/private | tenant, school and work item ID | Preserve original ID, private title, state and version. | platform/workspace/sqlstore/ownership_v2.go |
| ws_academics_v2 | modules/academics | school/private | tenant, school and work item ID | Preserve original ID, private title, state and version. | platform/workspace/sqlstore/ownership_v2.go |
| ws_curriculum_v2 | modules/curriculum | school/private | tenant, school and work item ID | Preserve original ID, private title, state and version. | platform/workspace/sqlstore/ownership_v2.go |
| ws_classes_v2 | modules/classes | school/private | tenant, school and work item ID | Preserve original ID, private title, state and version. | platform/workspace/sqlstore/ownership_v2.go |
| ws_timetables_v2 | modules/timetables | school/private | tenant, school and work item ID | Preserve original ID, private title, state and version. | platform/workspace/sqlstore/ownership_v2.go |
| ws_assessments_v2 | modules/assessments | school/private | tenant, school and work item ID | Preserve original ID, private title, state and version. | platform/workspace/sqlstore/ownership_v2.go |
| ws_gradebook_v2 | modules/gradebook | school/private | tenant, school and work item ID | Preserve original ID, private title, state and version. | platform/workspace/sqlstore/ownership_v2.go |
| ws_examinations_v2 | modules/examinations | school/private | tenant, school and work item ID | Preserve original ID, private title, state and version. | platform/workspace/sqlstore/ownership_v2.go |
| ws_transcripts_v2 | modules/transcripts | school/private | tenant, school and work item ID | Preserve original ID, private title, state and version. | platform/workspace/sqlstore/ownership_v2.go |
| ws_learning_v2 | modules/learning | school/private | tenant, school and work item ID | Preserve original ID, private title, state and version. | platform/workspace/sqlstore/ownership_v2.go |
| ws_fees_v2 | modules/fees | school/private | tenant, school and work item ID | Preserve original ID, private title, state and version. | platform/workspace/sqlstore/ownership_v2.go |
| ws_billing_v2 | modules/billing | school/private | tenant, school and work item ID | Preserve original ID, private title, state and version. | platform/workspace/sqlstore/ownership_v2.go |
| ws_payments_v2 | modules/payments | school/private | tenant, school and work item ID | Preserve original ID, private title, state and version. | platform/workspace/sqlstore/ownership_v2.go |
| ws_accounting_v2 | modules/accounting | school/private | tenant, school and work item ID | Preserve original ID, private title, state and version. | platform/workspace/sqlstore/ownership_v2.go |
| ws_scholarships_v2 | modules/scholarships | school/private | tenant, school and work item ID | Preserve original ID, private title, state and version. | platform/workspace/sqlstore/ownership_v2.go |
| ws_procurement_v2 | modules/procurement | school/private | tenant, school and work item ID | Preserve original ID, private title, state and version. | platform/workspace/sqlstore/ownership_v2.go |
| ws_inventory_v2 | modules/inventory | school/private | tenant, school and work item ID | Preserve original ID, private title, state and version. | platform/workspace/sqlstore/ownership_v2.go |
| ws_staff_v2 | modules/staff | school/private | tenant, school and work item ID | Preserve original ID, private title, state and version. | platform/workspace/sqlstore/ownership_v2.go |
| ws_hr_v2 | modules/hr | school/private | tenant, school and work item ID | Preserve original ID, private title, state and version. | platform/workspace/sqlstore/ownership_v2.go |
| ws_payroll_v2 | modules/payroll | school/private | tenant, school and work item ID | Preserve original ID, private title, state and version. | platform/workspace/sqlstore/ownership_v2.go |
| ws_library_v2 | modules/library | school/private | tenant, school and work item ID | Preserve original ID, private title, state and version. | platform/workspace/sqlstore/ownership_v2.go |
| ws_transport_v2 | modules/transport | school/private | tenant, school and work item ID | Preserve original ID, private title, state and version. | platform/workspace/sqlstore/ownership_v2.go |
| ws_facilities_v2 | modules/facilities | school/private | tenant, school and work item ID | Preserve original ID, private title, state and version. | platform/workspace/sqlstore/ownership_v2.go |
| ws_notifications_v2 | modules/notifications | school/private | tenant, school and work item ID | Preserve original ID, private title, state and version. | platform/workspace/sqlstore/ownership_v2.go |
| ws_messaging_v2 | modules/messaging | school/private | tenant, school and work item ID | Preserve original ID, private title, state and version. | platform/workspace/sqlstore/ownership_v2.go |
| ws_documents_v2 | modules/documents | school/private | tenant, school and work item ID | Preserve original ID, private title, state and version. | platform/workspace/sqlstore/ownership_v2.go |
| ws_workflows_v2 | modules/workflows | school/private | tenant, school and work item ID | Preserve original ID, private title, state and version. | platform/workspace/sqlstore/ownership_v2.go |
| ws_reporting_v2 | modules/reporting | school/private | tenant, school and work item ID | Preserve original ID, private title, state and version. | platform/workspace/sqlstore/ownership_v2.go |
| ws_analytics_v2 | modules/analytics | school/private | tenant, school and work item ID | Preserve original ID, private title, state and version. | platform/workspace/sqlstore/ownership_v2.go |
| ws_audit_v2 | modules/audit | school/private | tenant, school and work item ID | Preserve original ID, private title, state and version. | platform/workspace/sqlstore/ownership_v2.go |
| ws_portal_v2 | modules/portal | school/private | tenant, school and work item ID | Preserve original ID, private title, state and version. | platform/workspace/sqlstore/ownership_v2.go |
