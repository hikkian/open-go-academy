# Dependency license inventory (local metadata only)

Snapshot: 2026-10-16. This is an inventory, not legal advice or an SPDX conclusion. It reads package manifests/lockfiles and Go module metadata; license file names are listed without interpreting their text. No packages or scanners were installed. `.data` was not accessed.

For npm, version comes from lockfiles. Direct means present in the corresponding root `package.json` dependency/devDependency sections; otherwise transitive/platform-specific. License is the lockfile field, falling back to an already installed package `package.json` field. Missing declarations are `UNKNOWN`.

## npm: `package-lock.json`

| Scope | Package | Pinned version | package.json license field |
|---|---|---:|---|
| direct | `@axe-core/playwright` | `4.13.0` | `MPL-2.0` |
| direct | `@playwright/test` | `1.64.1` | `Apache-2.0` |
| transitive/platform | `axe-core` | `4.13.0` | `MPL-2.0` |
| direct | `jsqr` | `1.4.0` | `Apache-2.0` |
| transitive/platform | `playwright` | `1.64.1` | `Apache-2.0` |
| transitive/platform | `playwright-core` | `1.64.1` | `Apache-2.0` |
| direct | `pngjs` | `7.0.0` | `MIT` |

## npm: `web/package-lock.json`

| Scope | Package | Pinned version | package.json license field |
|---|---|---:|---|
| transitive/platform | `@codemirror/autocomplete` | `6.20.3` | `MIT` |
| transitive/platform | `@codemirror/commands` | `6.11.1` | `MIT` |
| direct | `@codemirror/lang-go` | `6.0.1` | `MIT` |
| direct | `@codemirror/language` | `6.13.1` | `MIT` |
| transitive/platform | `@codemirror/lint` | `6.9.7` | `MIT` |
| transitive/platform | `@codemirror/search` | `6.7.2` | `MIT` |
| direct | `@codemirror/state` | `6.7.6` | `MIT` |
| transitive/platform | `@codemirror/streamparser` | `6.0.0` | `MIT` |
| transitive/platform | `@codemirror/view` | `6.43.14` | `MIT` |
| transitive/platform | `@lezer/common` | `1.5.3` | `MIT` |
| transitive/platform | `@lezer/go` | `1.0.1` | `MIT` |
| direct | `@lezer/highlight` | `1.2.5` | `MIT` |
| transitive/platform | `@lezer/lr` | `1.4.11` | `MIT` |
| transitive/platform | `@marijn/find-cluster-break` | `1.0.4` | `MIT` |
| transitive/platform | `@oxc-project/types` | `0.154.0` | `MIT` |
| transitive/platform | `@rolldown/binding-android-arm-eabi` | `1.2.14` | `MIT` |
| transitive/platform | `@rolldown/binding-android-arm64` | `1.2.14` | `MIT` |
| transitive/platform | `@rolldown/binding-darwin-arm64` | `1.2.14` | `MIT` |
| transitive/platform | `@rolldown/binding-darwin-x64` | `1.2.14` | `MIT` |
| transitive/platform | `@rolldown/binding-freebsd-x64` | `1.2.14` | `MIT` |
| transitive/platform | `@rolldown/binding-linux-arm-gnueabihf` | `1.2.14` | `MIT` |
| transitive/platform | `@rolldown/binding-linux-arm64-gnu` | `1.2.14` | `MIT` |
| transitive/platform | `@rolldown/binding-linux-arm64-musl` | `1.2.14` | `MIT` |
| transitive/platform | `@rolldown/binding-linux-ppc64-gnu` | `1.2.14` | `MIT` |
| transitive/platform | `@rolldown/binding-linux-s390x-gnu` | `1.2.14` | `MIT` |
| transitive/platform | `@rolldown/binding-linux-x64-gnu` | `1.2.14` | `MIT` |
| transitive/platform | `@rolldown/binding-linux-x64-musl` | `1.2.14` | `MIT` |
| transitive/platform | `@rolldown/binding-openharmony-arm64` | `1.2.14` | `MIT` |
| transitive/platform | `@rolldown/binding-win32-arm64-msvc` | `1.2.14` | `MIT` |
| transitive/platform | `@rolldown/binding-win32-x64-msvc` | `1.2.14` | `MIT` |
| transitive/platform | `@rolldown/pluginutils` | `1.0.1` | `MIT` |
| transitive/platform | `@types/debug` | `4.1.13` | `MIT` |
| transitive/platform | `@types/estree` | `1.0.9` | `MIT` |
| transitive/platform | `@types/estree-jsx` | `1.0.5` | `MIT` |
| transitive/platform | `@types/hast` | `3.0.5` | `MIT` |
| transitive/platform | `@types/mdast` | `4.0.4` | `MIT` |
| transitive/platform | `@types/ms` | `2.1.0` | `MIT` |
| direct | `@types/react` | `19.3.0` | `MIT` |
| direct | `@types/react-dom` | `19.3.0` | `MIT` |
| transitive/platform | `@types/unist` | `3.0.3` | `MIT` |
| transitive/platform | `@typescript/typescript-aix-ppc64` | `7.0.2` | `Apache-2.0` |
| transitive/platform | `@typescript/typescript-darwin-arm64` | `7.0.2` | `Apache-2.0` |
| transitive/platform | `@typescript/typescript-darwin-x64` | `7.0.2` | `Apache-2.0` |
| transitive/platform | `@typescript/typescript-freebsd-arm64` | `7.0.2` | `Apache-2.0` |
| transitive/platform | `@typescript/typescript-freebsd-x64` | `7.0.2` | `Apache-2.0` |
| transitive/platform | `@typescript/typescript-linux-arm` | `7.0.2` | `Apache-2.0` |
| transitive/platform | `@typescript/typescript-linux-arm64` | `7.0.2` | `Apache-2.0` |
| transitive/platform | `@typescript/typescript-linux-loong64` | `7.0.2` | `Apache-2.0` |
| transitive/platform | `@typescript/typescript-linux-mips64el` | `7.0.2` | `Apache-2.0` |
| transitive/platform | `@typescript/typescript-linux-ppc64` | `7.0.2` | `Apache-2.0` |
| transitive/platform | `@typescript/typescript-linux-riscv64` | `7.0.2` | `Apache-2.0` |
| transitive/platform | `@typescript/typescript-linux-s390x` | `7.0.2` | `Apache-2.0` |
| transitive/platform | `@typescript/typescript-linux-x64` | `7.0.2` | `Apache-2.0` |
| transitive/platform | `@typescript/typescript-netbsd-arm64` | `7.0.2` | `Apache-2.0` |
| transitive/platform | `@typescript/typescript-netbsd-x64` | `7.0.2` | `Apache-2.0` |
| transitive/platform | `@typescript/typescript-openbsd-arm64` | `7.0.2` | `Apache-2.0` |
| transitive/platform | `@typescript/typescript-openbsd-x64` | `7.0.2` | `Apache-2.0` |
| transitive/platform | `@typescript/typescript-sunos-x64` | `7.0.2` | `Apache-2.0` |
| transitive/platform | `@typescript/typescript-win32-arm64` | `7.0.2` | `Apache-2.0` |
| transitive/platform | `@typescript/typescript-win32-x64` | `7.0.2` | `Apache-2.0` |
| transitive/platform | `@ungap/structured-clone` | `1.4.0` | `ISC` |
| direct | `@vitejs/plugin-react` | `6.1.2` | `MIT` |
| transitive/platform | `bail` | `2.0.2` | `MIT` |
| transitive/platform | `ccount` | `2.0.1` | `MIT` |
| transitive/platform | `character-entities` | `2.0.2` | `MIT` |
| transitive/platform | `character-entities-html4` | `2.1.0` | `MIT` |
| transitive/platform | `character-entities-legacy` | `3.0.0` | `MIT` |
| transitive/platform | `character-reference-invalid` | `2.0.1` | `MIT` |
| direct | `codemirror` | `6.0.2` | `MIT` |
| transitive/platform | `comma-separated-tokens` | `2.0.3` | `MIT` |
| transitive/platform | `crelt` | `1.0.7` | `MIT` |
| transitive/platform | `csstype` | `3.2.3` | `MIT` |
| transitive/platform | `debug` | `4.4.3` | `MIT` |
| transitive/platform | `decode-named-character-reference` | `1.3.0` | `MIT` |
| transitive/platform | `dequal` | `2.0.3` | `MIT` |
| transitive/platform | `detect-libc` | `2.1.2` | `Apache-2.0` |
| transitive/platform | `devlop` | `1.1.0` | `MIT` |
| transitive/platform | `estree-util-is-identifier-name` | `3.0.0` | `MIT` |
| transitive/platform | `extend` | `3.0.2` | `MIT` |
| transitive/platform | `fdir` | `6.5.0` | `MIT` |
| transitive/platform | `fsevents` | `2.3.3` | `MIT` |
| transitive/platform | `hast-util-to-jsx-runtime` | `2.3.6` | `MIT` |
| transitive/platform | `hast-util-whitespace` | `3.0.0` | `MIT` |
| transitive/platform | `html-url-attributes` | `3.0.1` | `MIT` |
| transitive/platform | `inline-style-parser` | `0.2.7` | `MIT` |
| transitive/platform | `is-alphabetical` | `2.0.1` | `MIT` |
| transitive/platform | `is-alphanumerical` | `2.0.1` | `MIT` |
| transitive/platform | `is-decimal` | `2.0.1` | `MIT` |
| transitive/platform | `is-hexadecimal` | `2.0.1` | `MIT` |
| transitive/platform | `is-plain-obj` | `4.1.0` | `MIT` |
| transitive/platform | `lightningcss` | `1.33.0` | `MPL-2.0` |
| transitive/platform | `lightningcss-android-arm64` | `1.33.0` | `MPL-2.0` |
| transitive/platform | `lightningcss-darwin-arm64` | `1.33.0` | `MPL-2.0` |
| transitive/platform | `lightningcss-darwin-x64` | `1.33.0` | `MPL-2.0` |
| transitive/platform | `lightningcss-freebsd-x64` | `1.33.0` | `MPL-2.0` |
| transitive/platform | `lightningcss-linux-arm-gnueabihf` | `1.33.0` | `MPL-2.0` |
| transitive/platform | `lightningcss-linux-arm64-gnu` | `1.33.0` | `MPL-2.0` |
| transitive/platform | `lightningcss-linux-arm64-musl` | `1.33.0` | `MPL-2.0` |
| transitive/platform | `lightningcss-linux-x64-gnu` | `1.33.0` | `MPL-2.0` |
| transitive/platform | `lightningcss-linux-x64-musl` | `1.33.0` | `MPL-2.0` |
| transitive/platform | `lightningcss-win32-arm64-msvc` | `1.33.0` | `MPL-2.0` |
| transitive/platform | `lightningcss-win32-x64-msvc` | `1.33.0` | `MPL-2.0` |
| transitive/platform | `longest-streak` | `3.1.0` | `MIT` |
| transitive/platform | `mdast-util-from-markdown` | `2.1.0` | `MIT` |
| transitive/platform | `mdast-util-mdx-expression` | `2.0.1` | `MIT` |
| transitive/platform | `mdast-util-mdx-jsx` | `3.2.0` | `MIT` |
| transitive/platform | `mdast-util-mdxjs-esm` | `2.0.1` | `MIT` |
| transitive/platform | `mdast-util-phrasing` | `4.1.0` | `MIT` |
| transitive/platform | `mdast-util-to-hast` | `13.2.1` | `MIT` |
| transitive/platform | `mdast-util-to-markdown` | `2.2.0` | `MIT` |
| transitive/platform | `mdast-util-to-string` | `4.0.0` | `MIT` |
| transitive/platform | `micromark` | `4.0.3` | `MIT` |
| transitive/platform | `micromark-core-commonmark` | `2.0.4` | `MIT` |
| transitive/platform | `micromark-factory-destination` | `2.0.1` | `MIT` |
| transitive/platform | `micromark-factory-label` | `2.0.1` | `MIT` |
| transitive/platform | `micromark-factory-space` | `2.1.0` | `MIT` |
| transitive/platform | `micromark-factory-title` | `2.0.1` | `MIT` |
| transitive/platform | `micromark-factory-whitespace` | `2.0.1` | `MIT` |
| transitive/platform | `micromark-util-character` | `2.1.1` | `MIT` |
| transitive/platform | `micromark-util-chunked` | `2.0.1` | `MIT` |
| transitive/platform | `micromark-util-classify-character` | `2.0.1` | `MIT` |
| transitive/platform | `micromark-util-combine-extensions` | `2.0.1` | `MIT` |
| transitive/platform | `micromark-util-decode-numeric-character-reference` | `2.0.2` | `MIT` |
| transitive/platform | `micromark-util-decode-string` | `2.0.1` | `MIT` |
| transitive/platform | `micromark-util-edit-map` | `1.0.0` | `MIT` |
| transitive/platform | `micromark-util-encode` | `2.0.1` | `MIT` |
| transitive/platform | `micromark-util-html-tag-name` | `2.0.1` | `MIT` |
| transitive/platform | `micromark-util-normalize-identifier` | `2.0.1` | `MIT` |
| transitive/platform | `micromark-util-resolve-all` | `2.0.1` | `MIT` |
| transitive/platform | `micromark-util-sanitize-uri` | `2.0.1` | `MIT` |
| transitive/platform | `micromark-util-subtokenize` | `2.1.0` | `MIT` |
| transitive/platform | `micromark-util-symbol` | `2.0.1` | `MIT` |
| transitive/platform | `micromark-util-types` | `2.0.3` | `MIT` |
| transitive/platform | `ms` | `2.1.3` | `MIT` |
| transitive/platform | `nanoid` | `3.3.20` | `MIT` |
| transitive/platform | `parse-entities` | `4.0.2` | `MIT` |
| transitive/platform | `@types/unist` | `2.0.11` | `MIT` |
| transitive/platform | `picocolors` | `1.1.1` | `ISC` |
| transitive/platform | `picomatch` | `4.0.7` | `MIT` |
| transitive/platform | `postcss` | `8.5.30` | `MIT` |
| transitive/platform | `property-information` | `7.2.0` | `MIT` |
| direct | `react` | `19.3.1` | `MIT` |
| direct | `react-dom` | `19.3.1` | `MIT` |
| direct | `react-markdown` | `10.1.0` | `MIT` |
| transitive/platform | `remark-parse` | `11.0.0` | `MIT` |
| transitive/platform | `remark-rehype` | `11.1.2` | `MIT` |
| transitive/platform | `rolldown` | `1.2.14` | `MIT` |
| transitive/platform | `scheduler` | `0.28.0` | `MIT` |
| transitive/platform | `source-map-js` | `1.2.2` | `BSD-3-Clause` |
| transitive/platform | `space-separated-tokens` | `2.0.2` | `MIT` |
| transitive/platform | `stringify-entities` | `4.0.4` | `MIT` |
| transitive/platform | `style-mod` | `4.1.4` | `MIT` |
| transitive/platform | `style-to-js` | `1.1.21` | `MIT` |
| transitive/platform | `style-to-object` | `1.0.14` | `MIT` |
| transitive/platform | `tinyglobby` | `0.2.17` | `MIT` |
| transitive/platform | `trim-lines` | `3.0.1` | `MIT` |
| transitive/platform | `trough` | `2.2.0` | `MIT` |
| direct | `typescript` | `7.0.2` | `Apache-2.0` |
| transitive/platform | `unified` | `11.0.5` | `MIT` |
| transitive/platform | `unist-util-is` | `6.0.1` | `MIT` |
| transitive/platform | `unist-util-position` | `5.0.0` | `MIT` |
| transitive/platform | `unist-util-stringify-position` | `4.0.0` | `MIT` |
| transitive/platform | `unist-util-visit` | `5.1.0` | `MIT` |
| transitive/platform | `unist-util-visit-parents` | `6.0.2` | `MIT` |
| transitive/platform | `vfile` | `6.0.3` | `MIT` |
| transitive/platform | `vfile-message` | `4.0.3` | `MIT` |
| direct | `vite` | `8.3.4` | `MIT` |
| transitive/platform | `w3c-keyname` | `2.2.8` | `MIT` |
| transitive/platform | `zwitch` | `2.0.4` | `MIT` |

## Go modules

Go version selection used the repository-pinned `.tools/go/bin/go` (`go list -m all`, `GOTOOLCHAIN=local`) and local module cache. Root dependency markers use `go.mod`; nested module markers use each nested `go.mod`. `UNKNOWN` means no SPDX expression was inferred: only license filenames are recorded.

### `go.mod`

| Scope | Module | Pinned version | SPDX/license field | License file(s) in local module cache |
|---|---|---:|---|---|
| transitive | `github.com/google/pprof` | `v0.0.0-20260802141513-ef3492d7dac3` | `UNKNOWN` | `LICENSE`, `third_party/svgpan/LICENSE` |
| direct | `github.com/google/uuid` | `v1.6.0` | `UNKNOWN` | `LICENSE` |
| direct | `github.com/gpdf-dev/gpdf` | `v1.0.13` | `UNKNOWN` | `LICENSE` |
| transitive | `github.com/hashicorp/golang-lru/v2` | `v2.0.7` | `UNKNOWN` | `LICENSE` |
| direct | `github.com/mattn/go-isatty` | `v0.0.24` | `UNKNOWN` | `LICENSE` |
| direct | `github.com/ncruces/go-strftime` | `v1.0.0` | `UNKNOWN` | `LICENSE` |
| direct | `github.com/remyoudompheng/bigfft` | `v0.0.0-20230129092748-24d4a6f8daec` | `UNKNOWN` | `LICENSE` |
| direct | `golang.org/x/image` | `v0.47.0` | `UNKNOWN` | `LICENSE` |
| transitive | `golang.org/x/mod` | `v0.41.0` | `UNKNOWN` | `LICENSE` |
| transitive | `golang.org/x/sync` | `v0.24.0` | `UNKNOWN` | `LICENSE` |
| direct | `golang.org/x/sys` | `v0.49.0` | `UNKNOWN` | `LICENSE` |
| direct | `golang.org/x/text` | `v0.43.0` | `UNKNOWN` | `LICENSE` |
| transitive | `golang.org/x/tools` | `v0.51.0` | `UNKNOWN` | `LICENSE` |
| transitive | `modernc.org/cc/v4` | `v4.29.7` | `UNKNOWN` | `LICENSE`, `testdata/jhjourdan/LICENSE` |
| transitive | `modernc.org/ccgo/v4` | `v4.36.1` | `UNKNOWN` | `LICENSE` |
| transitive | `modernc.org/fileutil` | `v1.4.0` | `UNKNOWN` | `LICENSE`, `falloc/LICENSE`, `hdb/LICENSE`, `storage/LICENSE` |
| transitive | `modernc.org/gc/v2` | `v2.6.5` | `UNKNOWN` | `LICENSE` |
| transitive | `modernc.org/gc/v3` | `v3.1.5` | `UNKNOWN` | `LICENSE` |
| transitive | `modernc.org/goabi0` | `v0.2.0` | `UNKNOWN` | `LICENSE` |
| direct | `modernc.org/libc` | `v1.77.1` | `UNKNOWN` | `LICENSE`, `LICENSE-3RD-PARTY.md` |
| direct | `modernc.org/mathutil` | `v1.7.1` | `UNKNOWN` | `LICENSE`, `mersenne/LICENSE` |
| direct | `modernc.org/memory` | `v1.12.1` | `UNKNOWN` | `LICENSE`, `LICENSE-GO`, `LICENSE-LOGO`, `LICENSE-MMAP-GO` |
| transitive | `modernc.org/opt` | `v0.2.0` | `UNKNOWN` | `LICENSE` |
| transitive | `modernc.org/sortutil` | `v1.2.1` | `UNKNOWN` | `LICENSE` |
| direct | `modernc.org/sqlite` | `v1.60.1` | `UNKNOWN` | `LICENSE`, `LICENSE-3RD-PARTY.md`, `LICENSE-SQLITE`, `LICENSE-SQLITE_VEC` |
| transitive | `modernc.org/strutil` | `v1.2.1` | `UNKNOWN` | `LICENSE` |
| transitive | `modernc.org/token` | `v1.1.0` | `UNKNOWN` | `LICENSE` |

### `content/projects/p1/reference/go.mod`

| Scope | Module | Pinned version | SPDX/license field | License file(s) in local module cache |
|---|---|---:|---|---|
| — | — | — | — | No external dependencies |

### `content/projects/p1/starter/go.mod`

| Scope | Module | Pinned version | SPDX/license field | License file(s) in local module cache |
|---|---|---:|---|---|
| — | — | — | — | No external dependencies |

### `content/projects/p2/reference/go.mod`

| Scope | Module | Pinned version | SPDX/license field | License file(s) in local module cache |
|---|---|---:|---|---|
| direct | `github.com/jackc/pgpassfile` | `v1.0.0` | `UNKNOWN` | `LICENSE` |
| direct | `github.com/jackc/pgservicefile` | `v0.0.0-20240606120523-5a60cdf6a761` | `UNKNOWN` | `LICENSE` |
| direct | `github.com/jackc/pgx/v5` | `v5.11.0` | `UNKNOWN` | `LICENSE` |
| direct | `github.com/jackc/puddle/v2` | `v2.2.2` | `UNKNOWN` | `LICENSE` |
| transitive | `github.com/kr/pretty` | `v0.3.0` | `UNKNOWN` | cache entry unavailable |
| transitive | `github.com/pmezard/go-difflib` | `v1.0.0` | `UNKNOWN` | `LICENSE` |
| transitive | `github.com/stretchr/objx` | `v0.1.0` | `UNKNOWN` | cache entry unavailable |
| transitive | `github.com/stretchr/testify` | `v1.11.1` | `UNKNOWN` | `LICENSE` |
| transitive | `golang.org/x/mod` | `v0.27.0` | `UNKNOWN` | cache entry unavailable |
| direct | `golang.org/x/sync` | `v0.17.0` | `UNKNOWN` | `LICENSE` |
| direct | `golang.org/x/text` | `v0.29.0` | `UNKNOWN` | `LICENSE` |
| transitive | `golang.org/x/tools` | `v0.36.0` | `UNKNOWN` | cache entry unavailable |
| transitive | `gopkg.in/check.v1` | `v1.0.0-20201130134442-10cb98267c6c` | `UNKNOWN` | cache entry unavailable |
| transitive | `gopkg.in/yaml.v3` | `v3.0.1` | `UNKNOWN` | `LICENSE`, `NOTICE` |

### `content/projects/p2/starter/go.mod`

| Scope | Module | Pinned version | SPDX/license field | License file(s) in local module cache |
|---|---|---:|---|---|
| — | — | — | — | No external dependencies |

### `content/projects/p3/reference/go.mod`

| Scope | Module | Pinned version | SPDX/license field | License file(s) in local module cache |
|---|---|---:|---|---|
| direct | `github.com/jackc/pgpassfile` | `v1.0.0` | `UNKNOWN` | `LICENSE` |
| direct | `github.com/jackc/pgservicefile` | `v0.0.0-20240606120523-5a60cdf6a761` | `UNKNOWN` | `LICENSE` |
| direct | `github.com/jackc/pgx/v5` | `v5.11.0` | `UNKNOWN` | `LICENSE` |
| direct | `github.com/jackc/puddle/v2` | `v2.2.2` | `UNKNOWN` | `LICENSE` |
| transitive | `github.com/kr/pretty` | `v0.3.0` | `UNKNOWN` | cache entry unavailable |
| transitive | `github.com/pmezard/go-difflib` | `v1.0.0` | `UNKNOWN` | `LICENSE` |
| transitive | `github.com/stretchr/objx` | `v0.1.0` | `UNKNOWN` | cache entry unavailable |
| transitive | `github.com/stretchr/testify` | `v1.11.1` | `UNKNOWN` | `LICENSE` |
| transitive | `golang.org/x/mod` | `v0.27.0` | `UNKNOWN` | cache entry unavailable |
| direct | `golang.org/x/sync` | `v0.17.0` | `UNKNOWN` | `LICENSE` |
| direct | `golang.org/x/text` | `v0.29.0` | `UNKNOWN` | `LICENSE` |
| transitive | `golang.org/x/tools` | `v0.36.0` | `UNKNOWN` | cache entry unavailable |
| transitive | `gopkg.in/check.v1` | `v1.0.0-20201130134442-10cb98267c6c` | `UNKNOWN` | cache entry unavailable |
| transitive | `gopkg.in/yaml.v3` | `v3.0.1` | `UNKNOWN` | `LICENSE`, `NOTICE` |

### `content/projects/p3/starter/go.mod`

| Scope | Module | Pinned version | SPDX/license field | License file(s) in local module cache |
|---|---|---:|---|---|
| direct | `github.com/jackc/pgpassfile` | `v1.0.0` | `UNKNOWN` | `LICENSE` |
| direct | `github.com/jackc/pgservicefile` | `v0.0.0-20240606120523-5a60cdf6a761` | `UNKNOWN` | `LICENSE` |
| direct | `github.com/jackc/pgx/v5` | `v5.11.0` | `UNKNOWN` | `LICENSE` |
| direct | `github.com/jackc/puddle/v2` | `v2.2.2` | `UNKNOWN` | `LICENSE` |
| transitive | `github.com/kr/pretty` | `v0.3.0` | `UNKNOWN` | cache entry unavailable |
| transitive | `github.com/pmezard/go-difflib` | `v1.0.0` | `UNKNOWN` | `LICENSE` |
| transitive | `github.com/stretchr/objx` | `v0.1.0` | `UNKNOWN` | cache entry unavailable |
| transitive | `github.com/stretchr/testify` | `v1.11.1` | `UNKNOWN` | `LICENSE` |
| transitive | `golang.org/x/mod` | `v0.27.0` | `UNKNOWN` | cache entry unavailable |
| direct | `golang.org/x/sync` | `v0.17.0` | `UNKNOWN` | `LICENSE` |
| direct | `golang.org/x/text` | `v0.29.0` | `UNKNOWN` | `LICENSE` |
| transitive | `golang.org/x/tools` | `v0.36.0` | `UNKNOWN` | cache entry unavailable |
| transitive | `gopkg.in/check.v1` | `v1.0.0-20201130134442-10cb98267c6c` | `UNKNOWN` | cache entry unavailable |
| transitive | `gopkg.in/yaml.v3` | `v3.0.1` | `UNKNOWN` | `LICENSE`, `NOTICE` |

### `content/projects/p4/reference/go.mod`

| Scope | Module | Pinned version | SPDX/license field | License file(s) in local module cache |
|---|---|---:|---|---|
| — | — | — | — | No external dependencies |

### `content/projects/p4/starter/go.mod`

| Scope | Module | Pinned version | SPDX/license field | License file(s) in local module cache |
|---|---|---:|---|---|
| — | — | — | — | No external dependencies |

### `content/projects/p5/reference/go.mod`

| Scope | Module | Pinned version | SPDX/license field | License file(s) in local module cache |
|---|---|---:|---|---|
| — | — | — | — | No external dependencies |

### `content/projects/p5/starter/go.mod`

| Scope | Module | Pinned version | SPDX/license field | License file(s) in local module cache |
|---|---|---:|---|---|
| — | — | — | — | No external dependencies |

## Limitations

- Go `UNKNOWN` does not mean a dependency lacks a license; this report deliberately does not parse license file contents or determine SPDX identifiers. Third-party code embedded in dependencies, generated assets, fonts, OCI images, and external tools are outside this module/package inventory.
- npm platform-specific packages are included if they appear in a lockfile, even if not installed on this host. Installed `package.json` metadata was consulted only when available; lockfile version remains authoritative.
- Go module graphs are those returned locally by the pinned Go tool for each module. A zero-dependency lesson starter has no external module rows. This inventory does not assess compatibility, license obligations, notices required by distribution, or the owner’s choice of project/content license.
- License paths are relative to each module cache entry and omit the machine-specific cache root. No file outside the repository and local module cache was inspected.
