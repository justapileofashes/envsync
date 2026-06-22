-- =============================================================================
-- EnvSync :: ephemeral "time-bomb" access grants
-- =============================================================================
-- Lets an admin/developer mint a scoped, time-limited token for a contractor or
-- freelancer. The raw token is shown to the CLI user exactly once; only its
-- SHA-256 hash is stored server-side. A grant auto-expires at expires_at and can
-- be revoked early. Enforcement lives in redeem_access_grant() below.
-- =============================================================================

create table if not exists public.access_grants (
    id          uuid primary key default gen_random_uuid(),
    project_id  uuid not null references public.projects(id) on delete cascade,
    token_hash  text not null unique,           -- sha256(raw token), hex
    role        text not null default 'read-only'
                    check (role in ('read-only','developer')),
    expires_at  timestamptz not null,
    revoked     boolean not null default false,
    created_by  uuid references auth.users(id) on delete set null,
    created_at  timestamptz not null default now()
);

create index if not exists access_grants_project_idx
    on public.access_grants (project_id);
create index if not exists access_grants_active_idx
    on public.access_grants (token_hash) where (not revoked);

alter table public.access_grants enable row level security;

-- Only org writers (admin/developer) may mint, view, or revoke grants for their
-- own projects.
drop policy if exists access_grants_select on public.access_grants;
create policy access_grants_select on public.access_grants
    for select using (public.is_org_writer(public.project_org(project_id)));

drop policy if exists access_grants_insert on public.access_grants;
create policy access_grants_insert on public.access_grants
    for insert with check (public.is_org_writer(public.project_org(project_id)));

drop policy if exists access_grants_update on public.access_grants;
create policy access_grants_update on public.access_grants
    for update using (public.is_org_writer(public.project_org(project_id)))
    with check (public.is_org_writer(public.project_org(project_id)));

-- redeem_access_grant resolves a raw token to the project it grants access to,
-- but only if the grant is unexpired and not revoked. SECURITY DEFINER so an
-- unauthenticated/limited caller can redeem without broad table access.
create or replace function public.redeem_access_grant(raw_token text)
returns table (project_id uuid, role text, expires_at timestamptz)
language sql
security definer
set search_path = public
stable
as $$
    select g.project_id, g.role, g.expires_at
    from public.access_grants g
    where g.token_hash = encode(digest(raw_token, 'sha256'), 'hex')
      and g.revoked = false
      and g.expires_at > now();
$$;
