// Package detect performs lightweight, zero-config framework detection so that
// `envsync pull` can write the dotenv file under the name the local framework
// actually loads (e.g. .env.local for Next.js, .env.development for Vite).
//
// Detection is best-effort and never fatal: when nothing matches, the caller
// falls back to a plain ".env".
package detect

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// Result describes what was detected in a directory.
type Result struct {
	// Framework is a human-readable name ("Next.js", "Vite", …) or "" if unknown.
	Framework string
	// EnvFile is the conventional dotenv filename for that framework.
	EnvFile string
}

// rule pairs a framework with the dependency names and config files that signal
// it, plus the dotenv filename it conventionally loads.
type rule struct {
	framework string
	envFile   string
	deps      []string // package.json dependency keys (any match)
	files     []string // marker files in the project root (any match)
}

// rules are evaluated in order; the first match wins. More specific frameworks
// are listed before more generic ones.
var rules = []rule{
	{
		framework: "Next.js",
		envFile:   ".env.local",
		deps:      []string{"next"},
		files:     []string{"next.config.js", "next.config.ts", "next.config.mjs", "next.config.cjs"},
	},
	{
		framework: "Vite",
		envFile:   ".env.development",
		deps:      []string{"vite"},
		files:     []string{"vite.config.js", "vite.config.ts", "vite.config.mjs"},
	},
	{
		framework: "Remix",
		envFile:   ".env",
		deps:      []string{"@remix-run/dev", "@remix-run/node"},
		files:     []string{"remix.config.js"},
	},
	{
		framework: "Astro",
		envFile:   ".env",
		deps:      []string{"astro"},
		files:     []string{"astro.config.mjs", "astro.config.ts", "astro.config.js"},
	},
	{
		framework: "SvelteKit",
		envFile:   ".env",
		deps:      []string{"@sveltejs/kit"},
		files:     []string{"svelte.config.js"},
	},
	{
		framework: "Create React App",
		envFile:   ".env.development.local",
		deps:      []string{"react-scripts"},
	},
	{
		framework: "Nuxt",
		envFile:   ".env",
		deps:      []string{"nuxt", "nuxt3"},
		files:     []string{"nuxt.config.ts", "nuxt.config.js"},
	},
	{
		framework: "Django",
		envFile:   ".env",
		files:     []string{"manage.py"},
	},
	{
		framework: "Rails",
		envFile:   ".env",
		files:     []string{"config/application.rb", "Gemfile"},
	},
}

// Detect inspects dir and returns the best-guess framework + env filename.
// On no match it returns Result{Framework: "", EnvFile: ".env"}.
func Detect(dir string) Result {
	deps := readPackageDeps(dir)

	for _, r := range rules {
		for _, d := range r.deps {
			if _, ok := deps[d]; ok {
				return Result{Framework: r.framework, EnvFile: r.envFile}
			}
		}
		for _, f := range r.files {
			if fileExists(filepath.Join(dir, filepath.FromSlash(f))) {
				return Result{Framework: r.framework, EnvFile: r.envFile}
			}
		}
	}
	return Result{Framework: "", EnvFile: ".env"}
}

// readPackageDeps merges dependencies + devDependencies from package.json into a
// set. Missing or malformed package.json yields an empty set (best-effort).
func readPackageDeps(dir string) map[string]struct{} {
	out := map[string]struct{}{}
	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return out
	}
	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if json.Unmarshal(data, &pkg) != nil {
		return out
	}
	for k := range pkg.Dependencies {
		out[strings.ToLower(k)] = struct{}{}
	}
	for k := range pkg.DevDependencies {
		out[strings.ToLower(k)] = struct{}{}
	}
	return out
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
