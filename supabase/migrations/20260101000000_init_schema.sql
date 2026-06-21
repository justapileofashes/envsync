-- =============================================================================
-- EnvSync :: initial schema
-- =============================================================================
-- Zero-knowledge environment-variable sync. The database stores only opaque
-- AES-256-GCM ciphertext (base64) plus encrypted-metadata for administration.
-- The team's cryptographic passphrase and derived key NEVER reach this server.
--
-- Row-Level Security is the load-bearing access control: a user may only touch
-- rows belonging to an organization they are a member of. Writes additionally
-- require an "admin" or "developer" role.
-- =============================================================================

create extension if not exists "pgcrypto";

-- -----------------------------------------------------------------------------
-- organizations
-- -----------------------------------------------------------------------------
-- One row per team. cryptographic_salt seeds client-side PBKDF2 key derivation;
-- it is non-secret by design (a salt only needs to be unique), but is scoped to
-- members so it is not world-readable.
create table if not exists public.organizations (
    id                  uuid primary key default gen_random_uuid(),
    name                text not null,
    cryptographic_salt  text not null,                 -- base64-encoded random salt
    seats_purchased     integer not null default 1 check (seats_purchased >= 0),
    credit_balance      bigint  not null default 0,    -- pre-paid credits (PayRam)
    subscription_status text not null default 'active'
                            check (subscription_status in ('active','past_due','canceled')),
    created_at          timestamptz not null default now()
);

-- -----------------------------------------------------------------------------
-- organization_members
-- -----------------------------------------------------------------------------
-- Junction between auth.users and organizations. Drives every RLS policy below.
create table if not exists public.organization_members (
    org_id     uuid not null references public.organizations(id) on delete cascade,
    user_id    uuid not null references auth.users(id) on delete cascade,
    role       text not null default 'developer'
                   check (role in ('admin','developer','viewer')),
    created_at timestamptz not null default now(),
    primary key (org_id, user_id)
);

create index if not exists organization_members_user_idx
    on public.organization_members (user_id);

-- -----------------------------------------------------------------------------
-- projects
-- -----------------------------------------------------------------------------
-- A project maps a local working directory (via .envsync.json) to a remote
-- secret store. project_id is what `envsync init <project_id>` records.
create table if not exists public.projects (
    id         uuid primary key default gen_random_uuid(),
    org_id     uuid not null references public.organizations(id) on delete cascade,
    name       text not null,
    slug       text not null,
    created_at timestamptz not null default now(),
    unique (org_id, slug)
);

create index if not exists projects_org_idx on public.projects (org_id);

-- -----------------------------------------------------------------------------
-- environments
-- -----------------------------------------------------------------------------
-- Append-only, versioned store of encrypted .env blobs. Every `envsync push`
-- inserts a new row with the next version_sequence; `envsync pull` reads the
-- highest version. ciphertext holds base64(nonce || aes-256-gcm ciphertext).
create table if not exists public.environments (
    id               uuid primary key default gen_random_uuid(),
    project_id       uuid not null references public.projects(id) on delete cascade,
    version_sequence integer not null,
    ciphertext       text not null,            -- base64 of nonce||ciphertext+tag
    checksum         text,                     -- optional sha256 of plaintext-blob, client-set
    created_by       uuid references auth.users(id) on delete set null,
    created_at       timestamptz not null default now(),
    unique (project_id, version_sequence)
);

create index if not exists environments_project_version_idx
    on public.environments (project_id, version_sequence desc);

-- =============================================================================
-- Helper functions (security definer) to avoid recursive RLS lookups.
-- =============================================================================
create or replace function public.is_org_member(target_org uuid)
returns boolean
language sql
security definer
set search_path = public
stable
as $$
    select exists (
        select 1 from public.organization_members m
        where m.org_id = target_org and m.user_id = auth.uid()
    );
$$;

create or replace function public.is_org_writer(target_org uuid)
returns boolean
language sql
security definer
set search_path = public
stable
as $$
    select exists (
        select 1 from public.organization_members m
        where m.org_id = target_org
          and m.user_id = auth.uid()
          and m.role in ('admin','developer')
    );
$$;

-- Resolve the org that owns a project (for environments policies).
create or replace function public.project_org(target_project uuid)
returns uuid
language sql
security definer
set search_path = public
stable
as $$
    select p.org_id from public.projects p where p.id = target_project;
$$;

-- =============================================================================
-- Row-Level Security
-- =============================================================================
alter table public.organizations        enable row level security;
alter table public.organization_members enable row level security;
alter table public.projects             enable row level security;
alter table public.environments         enable row level security;

-- organizations: members can read their org; only admins may update it.
drop policy if exists org_select on public.organizations;
create policy org_select on public.organizations
    for select using (public.is_org_member(id));

drop policy if exists org_update on public.organizations;
create policy org_update on public.organizations
    for update using (
        exists (
            select 1 from public.organization_members m
            where m.org_id = id and m.user_id = auth.uid() and m.role = 'admin'
        )
    );

-- organization_members: a user can always see their own membership rows and any
-- co-member of an org they belong to.
drop policy if exists members_select on public.organization_members;
create policy members_select on public.organization_members
    for select using (
        user_id = auth.uid() or public.is_org_member(org_id)
    );

-- Only admins may add/remove/modify seats.
drop policy if exists members_admin_write on public.organization_members;
create policy members_admin_write on public.organization_members
    for all using (
        exists (
            select 1 from public.organization_members m
            where m.org_id = organization_members.org_id
              and m.user_id = auth.uid()
              and m.role = 'admin'
        )
    )
    with check (
        exists (
            select 1 from public.organization_members m
            where m.org_id = organization_members.org_id
              and m.user_id = auth.uid()
              and m.role = 'admin'
        )
    );

-- projects: members read; writers (admin/developer) create & modify.
drop policy if exists projects_select on public.projects;
create policy projects_select on public.projects
    for select using (public.is_org_member(org_id));

drop policy if exists projects_write on public.projects;
create policy projects_write on public.projects
    for all using (public.is_org_writer(org_id))
    with check (public.is_org_writer(org_id));

-- environments: members read; writers append. No updates/deletes (append-only
-- audit trail) — note the absence of update/delete policies denies them.
drop policy if exists environments_select on public.environments;
create policy environments_select on public.environments
    for select using (public.is_org_member(public.project_org(project_id)));

drop policy if exists environments_insert on public.environments;
create policy environments_insert on public.environments
    for insert with check (public.is_org_writer(public.project_org(project_id)));

-- =============================================================================
-- Convenience view: latest version per project (encrypted-metadata only).
-- =============================================================================
create or replace view public.environment_latest as
    select distinct on (e.project_id)
        e.id,
        e.project_id,
        e.version_sequence,
        e.ciphertext,
        e.checksum,
        e.created_by,
        e.created_at
    from public.environments e
    order by e.project_id, e.version_sequence desc;
