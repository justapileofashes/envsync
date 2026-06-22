# EnvSync — Web

Marketing + admin front-end for EnvSync. **Not** a secrets tool: it never
requests the cryptographic passphrase and never decrypts `.env` values.

## Stack

- Next.js (App Router, TypeScript)
- Space Grotesk + JetBrains Mono (Google Fonts)
- Dark-only, glassmorphic, terminal aesthetic

## Develop

```sh
cd web
npm install
npm run dev        # http://localhost:3000
```

## Pages

- `/` — Landing page (`components/Landing.tsx`), ported from the
  "EnvSync Landing" Claude Design project: animated terminal (cycles
  `envsync pull` and `envsync run`), zero-knowledge schematic, six-command
  feature grid, and a PayRam prepaid pricing table.

## Security boundary

The web app handles seats, billing, project IDs, and version-history audits
only. It must **never** request/store the team passphrase or decrypt any value —
decryption is exclusively a CLI/client-machine operation.
