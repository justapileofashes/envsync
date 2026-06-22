import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "EnvSync — Your secrets, synced. Never seen.",
  description:
    "Zero-knowledge environment-variable sync for terminal-first teams. Encrypted on your machine with AES-256-GCM; the passphrase never leaves your laptop.",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <head>
        <link rel="preconnect" href="https://fonts.googleapis.com" />
        <link
          rel="preconnect"
          href="https://fonts.gstatic.com"
          crossOrigin=""
        />
        <link
          href="https://fonts.googleapis.com/css2?family=Space+Grotesk:wght@400;500;600;700&family=JetBrains+Mono:wght@400;500;600;700&display=swap"
          rel="stylesheet"
        />
      </head>
      <body>{children}</body>
    </html>
  );
}
