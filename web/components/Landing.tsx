"use client";

import { useEffect, useRef } from "react";

// Terminal frames mirror the source design's DCLogic script.
const FRAMES: { t: string; c: "prompt" | "dim" | "ok" }[][] = [
  [
    { t: "$ envsync pull", c: "prompt" },
    { t: "→ resolving team keyring · 4 members", c: "dim" },
    { t: "→ fetching ciphertext blob · 4.2 kb", c: "dim" },
    { t: "→ decrypting locally · AES-256-GCM", c: "dim" },
    { t: "✓ pulled v14 in 187ms", c: "ok" },
  ],
  [
    { t: "$ envsync run -- npm run dev", c: "prompt" },
    { t: "→ resolving env · production", c: "dim" },
    { t: "→ injecting secrets into process memory", c: "dim" },
    { t: "✓ injected 12 variable(s) into npm (zero-disk)", c: "ok" },
  ],
];

const GITHUB_URL = "https://github.com/justapileofashes/envsync";

function lineColor(c: "prompt" | "dim" | "ok") {
  return c === "prompt" ? "#EDEDEF" : c === "ok" ? "var(--acc)" : "#7A7A85";
}

export default function Landing() {
  const rootRef = useRef<HTMLDivElement>(null);

  // Terminal typewriter animation (re-implements the design's DCLogic.run()).
  useEffect(() => {
    const term = rootRef.current?.querySelector<HTMLDivElement>(
      "#envsync-terminal"
    );
    if (!term) return;

    const cursor =
      '<span style="display:inline-block;width:7px;height:15px;background:var(--acc);margin-left:3px;vertical-align:text-bottom;animation:blink 1.05s step-end infinite"></span>';

    const renderLines = (
      shown: { text: string; c: "prompt" | "dim" | "ok" }[],
      activeIndex: number
    ) => {
      term.innerHTML = shown
        .map(
          (l, i) =>
            `<div style="white-space:pre-wrap;color:${lineColor(l.c)}">${escapeHtml(
              l.text
            )}${i === activeIndex ? cursor : ""}</div>`
        )
        .join("");
    };

    const reduced =
      typeof window !== "undefined" &&
      window.matchMedia &&
      window.matchMedia("(prefers-reduced-motion: reduce)").matches;

    if (reduced) {
      renderLines(
        FRAMES[0].map((s) => ({ text: s.t, c: s.c })),
        -1
      );
      return;
    }

    let frame = 0;
    let li = 0;
    let ci = 0;
    let shown: { text: string; c: "prompt" | "dim" | "ok" }[] = [];
    let timer: ReturnType<typeof setTimeout>;

    const step = () => {
      const script = FRAMES[frame % FRAMES.length];
      if (li >= script.length) {
        timer = setTimeout(() => {
          frame++;
          li = 0;
          ci = 0;
          shown = [];
          term.innerHTML = "";
          timer = setTimeout(step, 450);
        }, 2200);
        return;
      }
      const line = script[li];
      ci++;
      shown[li] = { text: line.t.slice(0, ci), c: line.c };
      renderLines(shown, li);
      if (ci >= line.t.length) {
        li++;
        ci = 0;
        timer = setTimeout(step, 360);
      } else {
        timer = setTimeout(step, 14 + Math.random() * 26);
      }
    };
    step();

    return () => clearTimeout(timer);
  }, []);

  // Button actions: GitHub + toast feedback.
  useEffect(() => {
    const root = rootRef.current;
    if (!root) return;

    let toastEl: HTMLDivElement | null = null;
    let toastTimer: ReturnType<typeof setTimeout>;

    const showToast = (msg: string) => {
      if (!toastEl) {
        toastEl = document.createElement("div");
        toastEl.style.cssText =
          "position:fixed;bottom:28px;left:50%;transform:translateX(-50%);z-index:80;display:flex;align-items:center;gap:11px;font-family:'JetBrains Mono',monospace;font-size:13px;color:#EDEDEF;background:rgba(16,16,19,.92);backdrop-filter:blur(14px);-webkit-backdrop-filter:blur(14px);border:1px solid var(--acc-line);padding:13px 18px;border-radius:11px;box-shadow:0 20px 50px -20px rgba(0,0,0,.9);cursor:pointer";
        toastEl.addEventListener("click", () => toastEl?.remove());
        document.body.appendChild(toastEl);
      }
      toastEl.innerHTML = `<span style="width:7px;height:7px;border-radius:50%;background:var(--acc);box-shadow:0 0 8px var(--acc)"></span>${escapeHtml(
        msg
      )}`;
      clearTimeout(toastTimer);
      toastTimer = setTimeout(() => toastEl?.remove(), 3400);
    };

    const onClick = (e: MouseEvent) => {
      const target = (e.target as HTMLElement)?.closest("[data-action]");
      if (!target) return;
      const action = target.getAttribute("data-action");
      switch (action) {
        case "github":
          window.open(GITHUB_URL, "_blank");
          break;
        case "get-started":
          showToast("Get started — install the CLI: brew install envsync");
          break;
        case "solo":
          showToast("Solo selected — free for 1 developer");
          break;
        case "team":
          showToast(
            "Opening PayRam checkout · pay with crypto · prepay seats"
          );
          break;
      }
    };

    root.addEventListener("click", onClick);
    return () => {
      root.removeEventListener("click", onClick);
      clearTimeout(toastTimer);
      toastEl?.remove();
    };
  }, []);

  return (
    <div ref={rootRef} dangerouslySetInnerHTML={{ __html: MARKUP }} />
  );
}

function escapeHtml(s: string) {
  return s
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;");
}

// Static markup ported verbatim from "EnvSync Landing.dc.html" (the user's
// Claude Design project), with DC-specific tags removed: the terminal <sc-for>
// becomes #envsync-terminal (filled by the effect above), button handlers become
// data-action attributes, and the toast is injected by JS.
const MARKUP = `
<div style="min-height:100vh;position:relative;overflow:hidden;background:radial-gradient(1100px 520px at 72% -8%, var(--acc-glow), transparent 70%), radial-gradient(900px 500px at 8% 4%, rgba(255,255,255,.025), transparent 70%), #0A0A0B">

  <div style="position:absolute;inset:0;pointer-events:none;background-image:linear-gradient(rgba(255,255,255,var(--grid-line)) 1px, transparent 1px), linear-gradient(90deg, rgba(255,255,255,var(--grid-line)) 1px, transparent 1px);background-size:44px 44px;mask-image:radial-gradient(circle at 60% 18%, #000 30%, transparent 78%)"></div>

  <!-- NAV -->
  <nav style="position:sticky;top:0;z-index:50;display:flex;align-items:center;justify-content:space-between;padding:16px 36px;border-bottom:1px solid rgba(255,255,255,.07);background:rgba(10,10,11,.72);backdrop-filter:blur(14px);-webkit-backdrop-filter:blur(14px)">
    <div style="display:flex;align-items:center;gap:9px;font-family:'JetBrains Mono',monospace;font-weight:600;font-size:16px;letter-spacing:-.01em">
      <span style="color:var(--acc);text-shadow:0 0 12px var(--acc-line)">$</span><span>envsync</span>
    </div>
    <div style="display:flex;align-items:center;gap:30px;font-family:'JetBrains Mono',monospace;font-size:13.5px;color:#8A8A93">
      <span class="es-navlink">docs</span>
      <span class="es-navlink">security</span>
      <span class="es-navlink">pricing</span>
      <span class="es-navlink">changelog</span>
    </div>
    <div style="display:flex;align-items:center;gap:12px">
      <button data-action="github" class="es-btn es-btn-ghost" style="font-family:'JetBrains Mono',monospace;font-size:13px;color:#C6C6CC;background:transparent;border:1px solid rgba(255,255,255,.1);padding:8px 14px;border-radius:8px;cursor:pointer">GitHub ↗</button>
      <button data-action="get-started" class="es-btn es-btn-primary" style="font-family:'JetBrains Mono',monospace;font-size:13px;font-weight:600;color:#04130D;background:var(--acc);border:none;padding:9px 16px;border-radius:8px;cursor:pointer;box-shadow:0 0 0 1px var(--acc-line), 0 8px 24px -10px var(--acc-line)">Get started</button>
    </div>
  </nav>

  <!-- HERO -->
  <section style="position:relative;max-width:1200px;margin:0 auto;padding:72px 36px 40px;display:grid;grid-template-columns:1.04fr 1fr;gap:54px;align-items:center">
    <div>
      <div style="display:inline-flex;align-items:center;gap:9px;font-family:'JetBrains Mono',monospace;font-size:12px;color:var(--acc);background:var(--acc-dim);border:1px solid var(--acc-line);padding:6px 12px;border-radius:999px;margin-bottom:26px">
        <span style="width:6px;height:6px;border-radius:50%;background:var(--acc);box-shadow:0 0 8px var(--acc)"></span>
        zero-knowledge by architecture
      </div>
      <h1 style="font-size:56px;line-height:1.04;font-weight:600;letter-spacing:-.03em;margin:0 0 22px">Your secrets,<br>synced.<br><span style="color:var(--acc);text-shadow:0 0 36px var(--acc-line)">Never seen.</span></h1>
      <p style="font-size:18px;line-height:1.55;color:#9A9AA3;max-width:480px;margin:0 0 32px;text-wrap:pretty">Encrypted on your machine with AES-256-GCM. The team passphrase derives the key locally and <span style="color:#D8D8DE">never leaves your laptop</span> — our servers only ever hold opaque ciphertext.</p>
      <div style="display:flex;align-items:center;gap:14px;margin-bottom:34px">
        <button data-action="get-started" class="es-btn es-btn-primary" style="font-family:'JetBrains Mono',monospace;font-size:14px;font-weight:600;color:#04130D;background:var(--acc);border:none;padding:13px 22px;border-radius:10px;cursor:pointer;box-shadow:0 0 0 1px var(--acc-line),0 12px 32px -12px var(--acc-line)">Get started — free</button>
        <button data-action="github" class="es-btn es-btn-ghost" style="font-family:'JetBrains Mono',monospace;font-size:14px;color:#D8D8DE;background:rgba(255,255,255,.04);border:1px solid rgba(255,255,255,.12);padding:13px 20px;border-radius:10px;cursor:pointer;display:inline-flex;gap:8px;align-items:center">★ View on GitHub</button>
      </div>
      <div style="display:flex;flex-wrap:wrap;gap:8px">
        <span style="font-family:'JetBrains Mono',monospace;font-size:11.5px;color:#7A7A85;border:1px solid rgba(255,255,255,.08);padding:5px 10px;border-radius:7px">AES-256-GCM</span>
        <span style="font-family:'JetBrains Mono',monospace;font-size:11.5px;color:#7A7A85;border:1px solid rgba(255,255,255,.08);padding:5px 10px;border-radius:7px">PBKDF2 · 600k</span>
        <span style="font-family:'JetBrains Mono',monospace;font-size:11.5px;color:#7A7A85;border:1px solid rgba(255,255,255,.08);padding:5px 10px;border-radius:7px">0 plaintext bytes stored</span>
      </div>
    </div>

    <!-- TERMINAL -->
    <div style="position:relative">
      <div style="position:absolute;inset:-1px;border-radius:14px;background:linear-gradient(160deg, var(--acc-line), transparent 55%);opacity:.5;filter:blur(1px)"></div>
      <div style="position:relative;background:rgba(16,16,19,.82);backdrop-filter:blur(16px);-webkit-backdrop-filter:blur(16px);border:1px solid rgba(255,255,255,.1);border-radius:14px;overflow:hidden;box-shadow:0 40px 90px -40px rgba(0,0,0,.9)">
        <div style="display:flex;align-items:center;gap:8px;padding:13px 16px;border-bottom:1px solid rgba(255,255,255,.07);background:rgba(255,255,255,.02)">
          <span style="width:11px;height:11px;border-radius:50%;background:#FF5F57"></span>
          <span style="width:11px;height:11px;border-radius:50%;background:#FEBC2E"></span>
          <span style="width:11px;height:11px;border-radius:50%;background:#28C840"></span>
          <span style="margin-left:10px;font-family:'JetBrains Mono',monospace;font-size:12px;color:#6A6A73">zsh — envsync — 80×24</span>
        </div>
        <div id="envsync-terminal" style="padding:20px 20px 26px;font-family:'JetBrains Mono',monospace;font-size:13.5px;line-height:1.85;min-height:176px"></div>
      </div>
      <div style="position:absolute;bottom:-16px;right:18px;display:flex;align-items:center;gap:8px;font-family:'JetBrains Mono',monospace;font-size:11.5px;color:var(--acc);background:#0E0E11;border:1px solid var(--acc-line);padding:7px 12px;border-radius:8px;box-shadow:0 10px 30px -12px rgba(0,0,0,.8)">▲ 187ms · sub-200ms pull</div>
    </div>
  </section>

  <!-- STAT STRIP -->
  <section style="max-width:1200px;margin:24px auto 0;padding:0 36px">
    <div style="display:grid;grid-template-columns:repeat(4,1fr);border:1px solid rgba(255,255,255,.08);border-radius:14px;overflow:hidden;background:rgba(255,255,255,.015)">
      <div style="padding:22px 24px;border-right:1px solid rgba(255,255,255,.07)"><div style="font-family:'JetBrains Mono',monospace;font-size:28px;font-weight:600;color:#EDEDEF">187<span style="font-size:15px;color:#7A7A85">ms</span></div><div style="font-family:'JetBrains Mono',monospace;font-size:12px;color:#7A7A85;margin-top:4px">median pull</div></div>
      <div style="padding:22px 24px;border-right:1px solid rgba(255,255,255,.07)"><div style="font-family:'JetBrains Mono',monospace;font-size:28px;font-weight:600;color:#EDEDEF">4.2<span style="font-size:15px;color:#7A7A85">kb</span></div><div style="font-family:'JetBrains Mono',monospace;font-size:12px;color:#7A7A85;margin-top:4px">avg ciphertext blob</div></div>
      <div style="padding:22px 24px;border-right:1px solid rgba(255,255,255,.07)"><div style="font-family:'JetBrains Mono',monospace;font-size:28px;font-weight:600;color:var(--acc)">0</div><div style="font-family:'JetBrains Mono',monospace;font-size:12px;color:#7A7A85;margin-top:4px">plaintext bytes on server</div></div>
      <div style="padding:22px 24px"><div style="font-family:'JetBrains Mono',monospace;font-size:28px;font-weight:600;color:#EDEDEF">∞</div><div style="font-family:'JetBrains Mono',monospace;font-size:12px;color:#7A7A85;margin-top:4px">versioned history</div></div>
    </div>
  </section>

  <!-- ZERO KNOWLEDGE SCHEMATIC -->
  <section style="max-width:1200px;margin:96px auto 0;padding:0 36px">
    <div style="text-align:center;margin-bottom:48px">
      <div style="font-family:'JetBrains Mono',monospace;font-size:12px;color:var(--acc);letter-spacing:.08em;text-transform:uppercase;margin-bottom:14px">// trust model</div>
      <h2 style="font-size:40px;font-weight:600;letter-spacing:-.025em;margin:0 0 12px">The server can't read your secrets.<br>Not won't — <span style="color:var(--acc)">can't</span>.</h2>
      <p style="color:#9A9AA3;font-size:16px;max-width:560px;margin:0 auto;text-wrap:pretty">Encryption and decryption happen exclusively on developer machines. The passphrase and plaintext never cross the network.</p>
    </div>

    <div style="position:relative;background:rgba(16,16,19,.5);backdrop-filter:blur(12px);-webkit-backdrop-filter:blur(12px);border:1px solid rgba(255,255,255,.08);border-radius:18px;padding:40px 36px;display:grid;grid-template-columns:1fr auto 1fr;gap:0;align-items:stretch">

      <div style="padding-right:38px">
        <div style="display:inline-flex;align-items:center;gap:8px;font-family:'JetBrains Mono',monospace;font-size:11px;color:var(--acc);background:var(--acc-dim);border:1px solid var(--acc-line);padding:5px 11px;border-radius:999px;margin-bottom:22px">◇ YOUR MACHINE</div>
        <div style="display:flex;flex-direction:column;gap:14px">
          <div style="background:rgba(255,255,255,.025);border:1px solid rgba(255,255,255,.09);border-radius:10px;padding:14px 16px"><div style="font-family:'JetBrains Mono',monospace;font-size:13px;color:#EDEDEF">.env</div><div style="font-family:'JetBrains Mono',monospace;font-size:11.5px;color:#7A7A85;margin-top:3px">DATABASE_URL=postgres://…</div></div>
          <div style="display:flex;justify-content:center;color:var(--acc);font-size:18px">↓</div>
          <div style="background:var(--acc-dim);border:1px solid var(--acc-line);border-radius:10px;padding:14px 16px"><div style="font-family:'JetBrains Mono',monospace;font-size:13px;color:var(--acc)">⊕ PBKDF2 → AES-256-GCM</div><div style="font-family:'JetBrains Mono',monospace;font-size:11.5px;color:#9A9AA3;margin-top:3px">key derived from team passphrase</div></div>
          <div style="display:flex;justify-content:center;color:var(--acc);font-size:18px">↓</div>
          <div style="background:rgba(255,255,255,.025);border:1px solid rgba(255,255,255,.09);border-radius:10px;padding:14px 16px"><div style="font-family:'JetBrains Mono',monospace;font-size:13px;color:#EDEDEF">base64 ciphertext</div><div style="font-family:'JetBrains Mono',monospace;font-size:11.5px;color:#7A7A85;margin-top:3px;word-break:break-all">k4Hn0p…9f3a2c==</div></div>
        </div>
      </div>

      <div style="position:relative;width:1px;display:flex;flex-direction:column;align-items:center;justify-content:center">
        <div style="position:absolute;inset:0;border-left:1.5px dashed rgba(255,90,90,.45)"></div>
        <div style="position:absolute;top:50%;left:50%;transform:translate(-50%,-50%) rotate(90deg);white-space:nowrap;font-family:'JetBrains Mono',monospace;font-size:10.5px;letter-spacing:.18em;color:#FF7A7A;background:#0E0E11;padding:6px 10px;border:1px solid rgba(255,90,90,.3);border-radius:6px">⛔ PLAINTEXT NEVER CROSSES</div>
      </div>

      <div style="padding-left:38px;display:flex;flex-direction:column;justify-content:center">
        <div style="display:inline-flex;align-self:flex-start;align-items:center;gap:8px;font-family:'JetBrains Mono',monospace;font-size:11px;color:#9A9AA3;background:rgba(255,255,255,.04);border:1px solid rgba(255,255,255,.1);padding:5px 11px;border-radius:999px;margin-bottom:22px">▢ SUPABASE — STORAGE ONLY</div>
        <div style="background:rgba(255,255,255,.02);border:1px solid rgba(255,255,255,.09);border-radius:12px;padding:20px">
          <div style="display:flex;align-items:center;gap:10px;margin-bottom:14px"><span style="font-size:22px">🔒</span><div><div style="font-family:'JetBrains Mono',monospace;font-size:13px;color:#EDEDEF">stores: ciphertext + metadata</div><div style="font-family:'JetBrains Mono',monospace;font-size:11.5px;color:#7A7A85">versions · authors · timestamps · checksums</div></div></div>
          <div style="height:1px;background:rgba(255,255,255,.07);margin:14px 0"></div>
          <div style="display:flex;align-items:center;gap:8px;font-family:'JetBrains Mono',monospace;font-size:12.5px;color:#FF7A7A"><span>👁</span> cannot read a single secret value</div>
        </div>
        <div style="margin-top:22px;font-family:'JetBrains Mono',monospace;font-size:12px;color:#7A7A85;line-height:1.6">A teammate runs <span style="color:var(--acc)">envsync pull</span> → blob downloads → decrypts <span style="color:#D8D8DE">on their machine</span>, never here.</div>
      </div>
    </div>
  </section>

  <!-- FEATURES -->
  <section style="max-width:1200px;margin:96px auto 0;padding:0 36px">
    <div style="margin-bottom:36px">
      <div style="font-family:'JetBrains Mono',monospace;font-size:12px;color:var(--acc);letter-spacing:.08em;text-transform:uppercase;margin-bottom:12px">// the cli does the heavy lifting</div>
      <h2 style="font-size:38px;font-weight:600;letter-spacing:-.025em;margin:0">Six commands that change<br>how your team ships secrets.</h2>
    </div>
    <div style="display:grid;grid-template-columns:repeat(3,1fr);gap:18px">

      <div class="es-card es-card-flagship" style="position:relative;background:linear-gradient(180deg, var(--acc-glow), rgba(20,20,23,.6));backdrop-filter:blur(12px);-webkit-backdrop-filter:blur(12px);border:1px solid var(--acc-line);border-radius:14px;padding:24px">
        <div style="position:absolute;top:18px;right:18px;font-family:'JetBrains Mono',monospace;font-size:10px;color:var(--acc);background:var(--acc-dim);border:1px solid var(--acc-line);padding:3px 8px;border-radius:999px">FLAGSHIP</div>
        <div style="width:38px;height:38px;border-radius:10px;background:var(--acc-dim);border:1px solid var(--acc-line);display:flex;align-items:center;justify-content:center;margin-bottom:18px"><svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="var(--acc)" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M13 2L3 14h7l-1 8 10-12h-7z"/></svg></div>
        <div style="font-size:16px;font-weight:600;margin-bottom:8px">Zero-disk injection</div>
        <div style="font-size:13.5px;color:#8A8A93;line-height:1.55;margin-bottom:16px">Secrets live in RAM, never touch disk. Injected straight into the process you run.</div>
        <div style="font-family:'JetBrains Mono',monospace;font-size:12px;color:var(--acc);background:rgba(0,0,0,.35);border:1px solid var(--acc-line);border-radius:8px;padding:9px 12px"><span style="color:#5A5A63">$ </span>envsync run -- npm run dev</div>
      </div>

      <div class="es-card" style="background:rgba(20,20,23,.55);backdrop-filter:blur(12px);-webkit-backdrop-filter:blur(12px);border:1px solid rgba(255,255,255,.08);border-radius:14px;padding:24px">
        <div style="width:38px;height:38px;border-radius:10px;background:var(--acc-dim);border:1px solid var(--acc-line);display:flex;align-items:center;justify-content:center;margin-bottom:18px"><svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="var(--acc)" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M5 4h9l5 5v11"/><path d="M5 9h6M5 13h4M5 17h8"/></svg></div>
        <div style="font-size:16px;font-weight:600;margin-bottom:8px">Terminal diffing</div>
        <div style="font-size:13.5px;color:#8A8A93;line-height:1.55;margin-bottom:16px">See exactly which keys drift between environments — values stay masked.</div>
        <div style="font-family:'JetBrains Mono',monospace;font-size:12px;color:#C6C6CC;background:rgba(0,0,0,.3);border:1px solid rgba(255,255,255,.09);border-radius:8px;padding:9px 12px"><span style="color:#5A5A63">$ </span>envsync diff</div>
      </div>

      <div class="es-card" style="background:rgba(20,20,23,.55);backdrop-filter:blur(12px);-webkit-backdrop-filter:blur(12px);border:1px solid rgba(255,255,255,.08);border-radius:14px;padding:24px">
        <div style="width:38px;height:38px;border-radius:10px;background:var(--acc-dim);border:1px solid var(--acc-line);display:flex;align-items:center;justify-content:center;margin-bottom:18px"><svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="var(--acc)" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M9 12l2 2 4-4"/><path d="M21 12c0 5-3.5 7.5-8.5 9C7.5 19.5 4 17 4 12V6l8.5-3L21 6z"/></svg></div>
        <div style="font-size:16px;font-weight:600;margin-bottom:8px">Schema validation</div>
        <div style="font-size:13.5px;color:#8A8A93;line-height:1.55;margin-bottom:16px">Block a misspelled <span style="font-family:'JetBrains Mono',monospace;color:#FF7A7A">DATABSE_URL</span> before it ever ships.</div>
        <div style="font-family:'JetBrains Mono',monospace;font-size:12px;color:#C6C6CC;background:rgba(0,0,0,.3);border:1px solid rgba(255,255,255,.09);border-radius:8px;padding:9px 12px"><span style="color:#5A5A63">$ </span>envsync push --schema</div>
      </div>

      <div class="es-card" style="background:rgba(20,20,23,.55);backdrop-filter:blur(12px);-webkit-backdrop-filter:blur(12px);border:1px solid rgba(255,255,255,.08);border-radius:14px;padding:24px">
        <div style="width:38px;height:38px;border-radius:10px;background:var(--acc-dim);border:1px solid var(--acc-line);display:flex;align-items:center;justify-content:center;margin-bottom:18px"><svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="var(--acc)" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><rect x="4" y="10" width="16" height="11" rx="2"/><path d="M8 10V7a4 4 0 0 1 8 0v3"/></svg></div>
        <div style="font-size:16px;font-weight:600;margin-bottom:8px">Leak guard</div>
        <div style="font-size:13.5px;color:#8A8A93;line-height:1.55;margin-bottom:16px">A git hook that flat-out refuses to commit your <span style="font-family:'JetBrains Mono',monospace;color:#C6C6CC">.env</span>.</div>
        <div style="font-family:'JetBrains Mono',monospace;font-size:12px;color:#C6C6CC;background:rgba(0,0,0,.3);border:1px solid rgba(255,255,255,.09);border-radius:8px;padding:9px 12px"><span style="color:#5A5A63">$ </span>envsync protect</div>
      </div>

      <div class="es-card" style="background:rgba(20,20,23,.55);backdrop-filter:blur(12px);-webkit-backdrop-filter:blur(12px);border:1px solid rgba(255,255,255,.08);border-radius:14px;padding:24px">
        <div style="width:38px;height:38px;border-radius:10px;background:var(--acc-dim);border:1px solid var(--acc-line);display:flex;align-items:center;justify-content:center;margin-bottom:18px"><svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="var(--acc)" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="13" r="8"/><path d="M12 9v4l2.5 2.5"/><path d="M9 2h6M5.5 5.5L4 4M18.5 5.5L20 4"/></svg></div>
        <div style="font-size:16px;font-weight:600;margin-bottom:8px">Ephemeral access</div>
        <div style="font-size:13.5px;color:#8A8A93;line-height:1.55;margin-bottom:16px">Hire a contractor for the weekend — access self-destructs Monday.</div>
        <div style="font-family:'JetBrains Mono',monospace;font-size:12px;color:#C6C6CC;background:rgba(0,0,0,.3);border:1px solid rgba(255,255,255,.09);border-radius:8px;padding:9px 12px"><span style="color:#5A5A63">$ </span>envsync grant --expires 48h</div>
      </div>

      <div class="es-card" style="background:rgba(20,20,23,.55);backdrop-filter:blur(12px);-webkit-backdrop-filter:blur(12px);border:1px solid rgba(255,255,255,.08);border-radius:14px;padding:24px">
        <div style="width:38px;height:38px;border-radius:10px;background:var(--acc-dim);border:1px solid var(--acc-line);display:flex;align-items:center;justify-content:center;margin-bottom:18px"><svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="var(--acc)" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M12 3a9 9 0 1 0 9 9"/><path d="M12 7v5l3 2"/><path d="M16 3h5v5"/><path d="M21 3l-6 6"/></svg></div>
        <div style="font-size:16px;font-weight:600;margin-bottom:8px">Smart auto-detection</div>
        <div style="font-size:13.5px;color:#8A8A93;line-height:1.55;margin-bottom:16px">Knows it's Next.js, writes <span style="font-family:'JetBrains Mono',monospace;color:#C6C6CC">.env.local</span>. Zero config.</div>
        <div style="font-family:'JetBrains Mono',monospace;font-size:12px;color:#C6C6CC;background:rgba(0,0,0,.3);border:1px solid rgba(255,255,255,.09);border-radius:8px;padding:9px 12px"><span style="color:#5A5A63">$ </span>envsync pull</div>
      </div>
    </div>

    <div style="margin-top:18px;display:flex;flex-wrap:wrap;align-items:center;gap:10px;background:rgba(255,255,255,.015);border:1px solid rgba(255,255,255,.07);border-radius:14px;padding:16px 20px">
      <span style="font-family:'JetBrains Mono',monospace;font-size:11px;color:#5A5A63;letter-spacing:.06em">PLUS</span>
      <span style="font-family:'JetBrains Mono',monospace;font-size:12.5px;color:#9A9AA3;border:1px solid rgba(255,255,255,.08);padding:5px 11px;border-radius:7px">↺ versioned history</span>
      <span style="font-family:'JetBrains Mono',monospace;font-size:12.5px;color:#9A9AA3;border:1px solid rgba(255,255,255,.08);padding:5px 11px;border-radius:7px">⊟ append-only audit trail</span>
      <span style="font-family:'JetBrains Mono',monospace;font-size:12.5px;color:#9A9AA3;border:1px solid rgba(255,255,255,.08);padding:5px 11px;border-radius:7px">⊕ local .env.override merging</span>
      <span style="font-family:'JetBrains Mono',monospace;font-size:12.5px;color:#9A9AA3;border:1px solid rgba(255,255,255,.08);padding:5px 11px;border-radius:7px">⚡ instant team onboarding</span>
    </div>
  </section>

  <!-- PRICING -->
  <section style="max-width:1200px;margin:96px auto 0;padding:0 36px">
    <div style="text-align:center;margin-bottom:44px">
      <div style="font-family:'JetBrains Mono',monospace;font-size:12px;color:var(--acc);letter-spacing:.08em;text-transform:uppercase;margin-bottom:12px">// pricing</div>
      <h2 style="font-size:40px;font-weight:600;letter-spacing:-.025em;margin:0 0 10px">Prepay seats. Pay in crypto.</h2>
      <p style="color:#9A9AA3;font-size:16px;margin:0">Credit-based, no recurring card. Top up via PayRam, spend down as your team grows.</p>
    </div>
    <div style="display:grid;grid-template-columns:1fr 1.15fr;gap:22px;max-width:840px;margin:0 auto">
      <div style="background:rgba(20,20,23,.5);backdrop-filter:blur(12px);-webkit-backdrop-filter:blur(12px);border:1px solid rgba(255,255,255,.08);border-radius:16px;padding:30px">
        <div style="font-family:'JetBrains Mono',monospace;font-size:13px;color:#9A9AA3;margin-bottom:6px">Solo</div>
        <div style="display:flex;align-items:baseline;gap:6px;margin-bottom:6px"><span style="font-size:44px;font-weight:600;letter-spacing:-.02em">$0</span><span style="font-family:'JetBrains Mono',monospace;font-size:13px;color:#7A7A85">/ forever</span></div>
        <div style="font-size:13.5px;color:#8A8A93;margin-bottom:24px">For one developer. Unlimited projects.</div>
        <button data-action="solo" class="es-btn es-btn-ghost" style="width:100%;font-family:'JetBrains Mono',monospace;font-size:13.5px;color:#D8D8DE;background:rgba(255,255,255,.04);border:1px solid rgba(255,255,255,.14);padding:12px;border-radius:10px;cursor:pointer;margin-bottom:24px">Start solo</button>
        <div style="display:flex;flex-direction:column;gap:11px;font-size:13.5px;color:#B6B6BC">
          <div style="display:flex;gap:9px"><span style="color:var(--acc)">✓</span> 1 seat</div>
          <div style="display:flex;gap:9px"><span style="color:var(--acc)">✓</span> Unlimited projects & versions</div>
          <div style="display:flex;gap:9px"><span style="color:var(--acc)">✓</span> Full audit history</div>
          <div style="display:flex;gap:9px"><span style="color:#5A5A63">—</span> <span style="color:#7A7A85">Community support</span></div>
        </div>
      </div>
      <div style="position:relative;background:linear-gradient(180deg, rgba(0,229,160,.06), rgba(20,20,23,.6));backdrop-filter:blur(12px);-webkit-backdrop-filter:blur(12px);border:1px solid var(--acc-line);border-radius:16px;padding:30px;box-shadow:0 30px 80px -40px var(--acc-line)">
        <div style="position:absolute;top:20px;right:20px;font-family:'JetBrains Mono',monospace;font-size:10.5px;color:#04130D;background:var(--acc);padding:4px 10px;border-radius:999px;font-weight:600">PREPAID</div>
        <div style="font-family:'JetBrains Mono',monospace;font-size:13px;color:var(--acc);margin-bottom:6px">Team</div>
        <div style="display:flex;align-items:baseline;gap:6px;margin-bottom:6px"><span style="font-size:44px;font-weight:600;letter-spacing:-.02em">$8</span><span style="font-family:'JetBrains Mono',monospace;font-size:13px;color:#7A7A85">/ seat · billed in credits</span></div>
        <div style="font-size:13.5px;color:#9A9AA3;margin-bottom:24px">Buy credits once, allocate seats as you hire. No subscription, no card on file.</div>
        <button data-action="team" class="es-btn es-btn-primary" style="width:100%;display:flex;align-items:center;justify-content:center;gap:9px;font-family:'JetBrains Mono',monospace;font-size:14px;font-weight:600;color:#04130D;background:var(--acc);border:none;padding:13px;border-radius:10px;cursor:pointer;margin-bottom:24px;box-shadow:0 0 0 1px var(--acc-line),0 12px 30px -14px var(--acc-line)"><span style="font-size:15px">⬡</span> Pay with crypto via PayRam</button>
        <div style="display:flex;flex-direction:column;gap:11px;font-size:13.5px;color:#D2D2D8">
          <div style="display:flex;gap:9px"><span style="color:var(--acc)">✓</span> Unlimited seats — prepay per seat</div>
          <div style="display:flex;gap:9px"><span style="color:var(--acc)">✓</span> Role-based access · admin / dev / viewer</div>
          <div style="display:flex;gap:9px"><span style="color:var(--acc)">✓</span> Org-wide audit & version pinning</div>
          <div style="display:flex;gap:9px"><span style="color:var(--acc)">✓</span> BTC · ETH · USDC · 30+ chains</div>
        </div>
      </div>
    </div>
  </section>

  <!-- FOOTER -->
  <footer style="max-width:1200px;margin:100px auto 0;padding:40px 36px 56px;border-top:1px solid rgba(255,255,255,.07);display:grid;grid-template-columns:1.5fr 1fr 1fr 1fr;gap:30px">
    <div>
      <div style="display:flex;align-items:center;gap:9px;font-family:'JetBrains Mono',monospace;font-weight:600;font-size:15px;margin-bottom:12px"><span style="color:var(--acc)">$</span> envsync</div>
      <div style="font-size:13px;color:#7A7A85;line-height:1.6;max-width:240px">Zero-knowledge environment sync for terminal-first teams. Your secrets never leave your machine in the clear.</div>
    </div>
    <div style="font-family:'JetBrains Mono',monospace;font-size:13px;display:flex;flex-direction:column;gap:11px"><div style="color:#5A5A63;font-size:11px;letter-spacing:.06em;text-transform:uppercase;margin-bottom:4px">Product</div><span class="es-foot" style="color:#9A9AA3">CLI docs</span><span class="es-foot" style="color:#9A9AA3">Pricing</span><span class="es-foot" style="color:#9A9AA3">Changelog</span></div>
    <div style="font-family:'JetBrains Mono',monospace;font-size:13px;display:flex;flex-direction:column;gap:11px"><div style="color:#5A5A63;font-size:11px;letter-spacing:.06em;text-transform:uppercase;margin-bottom:4px">Trust</div><span style="color:var(--acc);cursor:pointer">Security model ↗</span><span class="es-foot" style="color:#9A9AA3">Threat model</span><span class="es-foot" style="color:#9A9AA3">Audit reports</span></div>
    <div style="font-family:'JetBrains Mono',monospace;font-size:13px;display:flex;flex-direction:column;gap:11px"><div style="color:#5A5A63;font-size:11px;letter-spacing:.06em;text-transform:uppercase;margin-bottom:4px">Connect</div><span data-action="github" class="es-foot" style="color:#9A9AA3">GitHub ↗</span><span class="es-foot" style="color:#9A9AA3">Discord</span><span class="es-foot" style="color:#9A9AA3">status</span></div>
  </footer>

</div>
`;
