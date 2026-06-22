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
claude_design_prompt.md   handoff brief for the web interface
```

## Web interface

The web admin/marketing front-end is **not** built here. See
[`claude_design_prompt.md`](./claude_design_prompt.md) for the complete handoff
brief. The web app handles seats, billing, project IDs, and version-history
audits only — it must **never** touch the passphrase or decrypt any value.
