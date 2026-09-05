## Epic 2: Permissions, Roles & Administration Foundation

Administrators manage who belongs and what they are allowed to do. Admins approve pending self-registered volunteers, create/edit/deactivate users, manage user groups and roles, assign qualifications, and maintain the additive action-matched permission model. The admin module is isolated and returned HTTP 403 (with hidden existence) for non-admins.

### Story 2.1: Admin Module Access Isolation

As the security system,
I want the admin module to be fully isolated from non-admin users,
So that admin functionality and even its existence stay hidden from unauthorized users.

**Acceptance Criteria:**

**Given** I am a non-admin authenticated user,
**When** I request any admin route,
**Then** the server returns HTTP 403 and no admin data is exposed (FR-19, server-side only, AD-6).

**Given** I am a non-admin (or unauthenticated) user using the SPA,
**When** the client renders,
**Then** the admin module's existence is hidden — no links, menu entries, route hints, or UI hints point to it (FR-19/AD-6)
**And** if I force-navigate to an admin path, the SPA redirects to the Dashboard (UX-DR6).

**Given** an unauthenticated user requests an admin route,
**When** access is attempted,
**Then** they are not granted content and no indication of the admin module's existence is returned (anti-enumeration of admin existence, FR-19).

**Given** the permission model (AD-12),
**When** admin actions are attempted,
**Then** each admin action maps to exactly one permission code and requires it (e.g. `admin.settings.*`, `users.approve`, `users.manage`, `roles.*`, `qualifications.manage`), enforced server-side (AD-6).

**Given** an unauthorized access attempt,
**When** the 403 is triggered,
**Then** it is emitted to structured auth logging (NFR-O1) and the UI shows "Zugriff verweigert" toast with no admin surfaces rendered (UX-DR6/UX-DR8).

### Story 2.2: Resolution of Active Permission Set

As the authorization service,
I want to resolve each user's active permission set from groups and direct grants,
So that every protected action is authorized against a current, additive, server-side permission set.

**Acceptance Criteria:**

**Given** the additive permission model (AD-12),
**When** a user's permission set is resolved,
**Then** it is the additive union of their permission-group memberships and any direct grants — there are no deny permissions, and effective permissions never subtract
**And** the set maps each action to exactly one permission code (21 base codes, AD-12).

**Given** the base role matrix,
**When** a user belongs to a base role (`helfende`, `schirrmeister`, `fuehrende`, `admin`),
**Then** their set reflects the additive role matrix (e.g. `admin` = all 21 codes; `helfende` = dashboard.view, inspection.submit) (AD-12).

**Given** a permission or membership changes (group assignment, direct grant, role edit),
**When** the change is committed,
**Then** the user's effective permission set updates on the next request — revocation takes effect immediately with no cache delay (AD-2/AD-6/FR-21/FR-22).

**Given** the User module owns identity and permission resolution,
**When** another module (Tool, Admin) needs authorization,
**Then** it consumes the permission set through the User module's port — the server validates every request against it, never the client (AD-2/AD-6).

**Given** an action lacking the required permission code,
**When** it is attempted,
**Then** the server returns HTTP 403 (AD-6).

### Story 2.3: Admin UX Foundation

As an admin,
I want a consistent, accessible, responsive admin shell,
So that I can manage users, roles, qualifications, and system config efficiently on desktop, tablet, or mobile.

**Acceptance Criteria:**

**Given** the design tokens and component library (from DESIGN.md, Epic 1),
**When** the admin module surfaces are built,
**Then** they consume the DESIGN.md tokens (light + `-dark`) and the reusable components (cards, buttons, status chips, input fields) with German microcopy (UX-DR4/5/8).

**Given** I access the admin module on mobile, tablet, or desktop,
**When** any admin surface is rendered,
**Then** it is responsive (mobile: collapsed nav to hamburger/bottom; tablet/desktop: persistent admin sidebar; >1024px: full sidebar) (UX-DR10).

**Given** the admin nav,
**When** I use it,
**Then** it exposes only the entry points my permission set allows (Nav: Übersicht · Benutzer · Rollen · Qualifikationen · Werkzeuge · Einstellungen · DSGVO) (UX-DR6/AD-6).

**Given** I interact with admin forms and lists,
**When** a state changes or an error occurs,
**Then** feedback is inline German text (loading skeletons, inline errors, empty states like "Keine ausstehenden Anträge", success confirmations) per UX-DR6/UX-DR8
**And** all interactive elements meet the accessibility floor (≥48px targets, keyboard, focus order, SR announcements) (UX-DR9).

**Given** the admin module surfaces,
**When** they are built,
**Then** they match the EXPERIENCE.md IA for the Admin-Modul and Werkzeugverwaltung surfaces and use route-gating to the contained permission codes (AD-6).

### Story 2.4: User Approval Workflow

As an admin,
I want to review and approve or reject pending self-registered volunteers,
So that only vetted volunteers gain access to the app.

**Acceptance Criteria:**

**Given** one or more `pending_approval` users exist,
**When** I open the "Verwaltung — Start" surface,
**Then** I see the pending requests (e.g. "Tim Müller — pending — self-registered") with their submitted details (Vorname, Nachname, E-Mail) (FR-20/UX-DR8).

**Given** I review a pending request,
**When** I approve it,
**Then** the user moves to `active` status and can log in (FR-20/FR-5)
**And** the user is seeded with the default `helfende` role (AD-2).

**Given** I reject a pending request,
**When** I confirm rejection,
**Then** the pending record is removed and the user cannot register/log in again with that pending state (FR-20)
**And** the rejection is recorded to audit logging (NFR-O1/AD-8).

**Given** the approval surface is empty,
**When** I open it,
**Then** the empty state reads "Keine ausstehenden Anträge" (UX-DR6/UX-DR8).

**Given** a user was approved,
**When** they attempt to log in,
**Then** login succeeds with their resolved permissions, per AD-2/AD-6.

### Story 2.5: Role & Permission-Group Management

As an admin,
I want to create and edit named permission groups (roles) by checking the permission codes they grant,
So that I can tailor what groups of volunteers can do without duplicating per-user grants.

**Acceptance Criteria:**

**Given** the permission model,
**When** I open the "Rollen" surface,
**Then** I see the roles including the four base roles (`helfende`, `schirrmeister`, `fuehrende`, `admin`) and any custom named groups (AD-12).

**Given** I create or edit a role,
**When** I use the role editor,
**Then** each permission is an additive checkbox (checked = granted; there are no deny permissions), covering the 21 base codes (FR-6/AD-12)
**And** I can save the role with a name and its checked permission set.

**Given** I edit a role,
**When** I change its checks and save,
**Then** affected users' effective permission sets update immediately on the next request (AD-2/AD-6/FR-6).

**Given** the role model constraints (AD-12),
**When** I manage roles,
**Then** groups are flat (no groups-in-groups in V1), and base roles are editable while remaining the named starting point for the matrix.

**Given** I add a new role,
**When** I create it,
**Then** it is available to assign to users (e.g. a `schirrmeister`-like custom role) via the user editor (FR-6).

### Story 2.6: User & Group Administration

As an admin,
I want to create, edit, and deactivate user accounts and manage user groups,
So that I can keep the volunteer directory current and control access precisely.

**Acceptance Criteria:**

**Given** the "Benutzer" surface,
**When** I open it,
**Then** I see the user list with each user's status (`aktiv`, `pending`, `deaktiviert`) (FR-21/UX-DR8).

**Given** I edit a user,
**When** I open a user detail,
**Then** I can assign roles and user groups (e.g. "Gruppe Ost"), add direct permission grants, and see their qualification assignments (FR-21/UX-DR5).

**Given** I create or edit a user,
**When** I change their profile/group membership,
**Then** the change persists and their resolved permission set updates immediately (AD-2/FR-21).

**Given** I create a user group,
**When** I set its name,
**Then** I can assign users to it, and membership grants the group's permissions (organisational groups grant no access, AD-12).

**Given** I deactivate a user account,
**When** I confirm deactivation,
**Then** the user's status becomes `deaktiviert`, they cannot authenticate at all ("→ Sofort kein Login", FR-21/UX-DR8), and the action is audited (NFR-O1).

**Given** I remove a user from a group,
**When** the removal is saved,
**Then** any permissions inherited from that group are revoked immediately on the user's next request (FR-21/AD-2/AD-6).

**Given** a user is deactivated,
**When** they attempt to log in,
**Then** authentication is rejected, consistent with Story 1.4.

### Story 2.7: Qualification Management

As an admin,
I want to manage qualifications and assign them to volunteers,
So that only volunteers with the required qualifications can be scheduled for inspection/service tasks.

**Acceptance Criteria:**

**Given** the "Qualifikationen" surface,
**When** I open it,
**Then** I see the qualification list with per-qualification status indicators (`Gültig`, `Bald ablaufend`, `Abgelaufen`, or `Unbegrenzt` for never-expiring qualifications) (FR-22/AD-7/UX-DR8).

**Given** I create or edit a qualification,
**When** I save it,
**Then** it persists with a name and either a fixed validity period **or no expiry at all** (unbegrenzt gültig, lasts forever) (FR-22/AD-7).

**Given** a qualification is unbegrenzt (no expiry),
**When** eligibility is checked,
**Then** it never expires — the "Bald ablaufend"/"Abgelaufen" states never apply and assignments retain inspection rights indefinitely.

**Given** a qualification has a fixed validity period,
**When** its expiry date is reached,
**Then** the volunteer's eligibility for tasks requiring it is revoked (AD-7).

**Given** I assign a qualification to a volunteer,
**When** I save the assignment,
**Then** it is reflected immediately in the volunteer's profile and in scheduling eligibility (FR-22/AD-7).

**Given** I remove a qualification from a volunteer,
**When** the removal is saved,
**Then** eligibility for tasks requiring it is revoked immediately (FR-22/AD-2/AD-6).

### Story 2.8: Admin One-Time-Password (OTP) Issuance

As an admin,
I want to create a one-time password for a user's account and transmit it securely out-of-band,
So that a user locked out while SMTP is unconfigured can regain access.

**Acceptance Criteria:**

**Given** an account exists (`active` or `deactivated`),
**When** I open the user detail in "Benutzer",
**Then** I can generate a one-time password for that account (FR-21/Story 1.8).

**Given** I generate the one-time password,
**When** I confirm,
**Then** the account is flagged so the next login **forces** a mandatory password change (Story 1.8/FR-2)
**And** the one-time password is shown once, then never recoverable, and must be transmitted outside the system in a secure way (never emailed) (Story 1.8/NFR-S4).

**Given** the one-time password exists,
**When** it is used to log in,
**Then** login succeeds only with the mandatory forced password change, after which the one-time password is invalid (single-use) (Story 1.8).

**Given** I generate an OTP,
**When** the action completes,
**Then** it is audited with actor, timestamp, and target user (NFR-O1).
