# EnvSync

Zero-knowledge, terminal-first environment-variable synchronization for small
engineering teams.

EnvSync encrypts your `.env` files **on your machine** with AES-256-GCM, using a
key derived (PBKDF2-HMAC-SHA256) from a team passphrase that **never leaves your
computer**. The backend (Supabase) only ever stores opaque ciphertext plus
encrypted metadata — it can never read your secrets.

Repo: https://github.com/justapileofashes/envsync

## Install

```sh
go install github.com/justapileofashes/envsync@latest
# or build locally:
go build -o envsync .
```

## Usage

```sh
envsync login                 # authenticate + set the team passphrase (local only)
envsync init <project_id>     # link this directory to a remote project
envsync push                  # encrypt and upload the local .env as a new version
envsync pull                  # download and decrypt the latest .env (backs up to .env.bak)
envsync run -- npm run dev     # run a command with secrets injected into RAM (zero-disk)
```

### Zero-disk injection (`run`)

`envsync run -- <command>` decrypts the latest environment and injects the
variables straight into the child process's environment. The secrets are
**never written to disk** — they live only in RAM for the command's lifetime
and disappear when it exits. Child exit codes are propagated transparently.

```sh
envsync run -- go run main.go
envsync run -- next dev
```

### Smart framework auto-detection

`envsync pull` inspects the project (`package.json` deps + config files) and
writes the dotenv file under the name the framework actually loads:

| Detected            | Output file                |
|---------------------|----------------------------|
| Next.js             | `.env.local`               |
| Vite                | `.env.development`         |
| Create React App    | `.env.development.local`   |
| Remix / Astro / SvelteKit / Nuxt / Django / Rails | `.env` |

Override with `--out <file>`, or disable detection with `--no-detect`.

### Terminal diffing (`diff`)

`envsync diff` compares your local env file against the latest remote version
with a color-coded report — `+` local-only, `-` missing locally, `~` value
differs. Values are masked by default; `--values` reveals them, `--exit-code`
makes it non-zero on drift (handy in CI).

### Local override merging (`.env.override`)

Keep a personal `.env.override` (e.g. your own test database). `envsync pull`
merges it on top of the team values — your overrides win, and they are **never**
pushed back to the cloud. Disable with `--no-override`.

### Schema validation & typo-squashing (`.env.schema`)

Commit a `.env.schema` and `envsync push` validates against it, blocking pushes
that are missing required keys, break a `prefix=`/`regex=` constraint, or look
like a typo of a required key (`DATABSE_URL` → "did you mean DATABASE_URL?").

```
DATABASE_URL required
STRIPE_KEY   required prefix=sk_
PORT         optional regex=^[0-9]+$
```

Bypass with `--skip-schema`.

### Leak guard (`protect`)

`envsync protect` installs a git pre-commit hook that blocks accidental commits
of `.env` files (allowing `.env.example` / `.env.schema` / `.env.sample`).
Remove it with `envsync protect --uninstall`.

### Ephemeral access grants (`grant`)

`envsync grant --role read-only --expires 48h` mints a scoped, time-limited
token for a contractor. Only the token's SHA-256 hash is stored; the raw token
is shown once. The server enforces expiry/revocation via the
`redeem_access_grant()` function (see `supabase/migrations`).

`login` stores the Supabase JWT and the passphrase in
`~/.envsync/credentials.json` (mode `0600`). `init` writes `.envsync.json` in the
current directory mapping it to the project and caching the org salt.

## Security model

- **Client-side encryption.** AES-256-GCM; a fresh random 12-byte nonce per
  push, prepended to the ciphertext.
- **Key derivation.** PBKDF2-HMAC-SHA256, 600,000 iterations, per-org salt.
- **Zero-knowledge.** The passphrase and plaintext never cross the network. The
  server holds only `base64(nonce || ciphertext+tag)` and metadata.
- **Append-only audit.** Every push is a new `version_sequence`; history is never
  mutated. Row-Level Security scopes all access to org members.

## Layout

```
cmd/                  Cobra commands (root, login, init, push, pull)
internal/crypto       AES-256-GCM + PBKDF2 engine
internal/api          Supabase Auth + PostgREST client
internal/env          .env parsing, formatting, backup
internal/config       local credentials + workspace state
supabase/migrations   Postgres schema + RLS policies
```

## Web interface

The marketing + admin front-end lives in [`web/`](./web) (Next.js). The landing
page is implemented in `web/components/Landing.tsx` — dark, glassmorphic,
terminal-aesthetic, with an animated `envsync pull` / `envsync run` terminal and
a PayRam prepaid pricing table.

```sh
cd web && npm install && npm run dev   # http://localhost:3000
```

The web app handles seats, billing, project IDs, and version-history audits
only — it must **never** touch the passphrase or decrypt any value.
