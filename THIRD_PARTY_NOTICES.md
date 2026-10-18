# Third-party notices

## Current source distribution

The Git source archive does not include Go module source, `node_modules`, generated `web/dist`, or prebuilt application binaries. Go and npm dependencies are pinned by `go.mod`/`go.sum` and `package-lock.json` files and are fetched by their package managers; each package retains its own license and notices. The project MIT and CC BY 4.0 grants do not relicense those dependencies.

The only third-party asset currently vendored in the source tree is:

| Asset | Upstream attribution | License notice |
|---|---|---|
| `internal/credentialrender/assets/DejaVuSans.ttf` | Bitstream/DejaVu; imported glyphs include Arev/Tavmjong Bah | [`LICENSE-DejaVu.txt`](internal/credentialrender/assets/LICENSE-DejaVu.txt), distributed beside the font |

The exact dependency/version and observed license-file inventory is [`m13-license-inventory-e2bad43-20261018.json`](docs/validation/m13-license-inventory-e2bad43-20261018.json). It is cache-only evidence and does not prove every unavailable package payload or legal obligation.

## Future binary or bundled-web distribution

Before distributing a Go binary, generated web bundle, installer, or OCI image, regenerate notices from the exact linked modules and bundled npm production dependencies. Preserve upstream copyright/conditions/disclaimers and any `NOTICE`/third-party license collections, including the notices supplied by `modernc.org/sqlite` when its code is included. Exclude development-only packages only after confirming they are not in the distributed artifact. The project has not published such artifacts.
