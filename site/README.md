# emod website

The public site at <https://emod.saphix.app> — a marketing homepage, the docs section, and the
browser diagram viewer under `/viewer/`. Built with [Hugo](https://gohugo.io) and the
[Hextra](https://imfing.github.io/hextra) theme.

## Local development

Hugo, Go and Task are pinned in `mise.toml`, so from the repo root:

```sh
mise install
task site:run      # dev server with live reload
task site:build    # production build
```

The output lands in `site/public/` (gitignored). `site:build` uses the `baseURL` from
`hugo.yaml` unless `BASE_URL` is set in the environment (the deploy workflow sets it).

The viewer is built separately (`task build:web` → `web/`) and copied into
`site/static/viewer/` by the deploy workflow, so it is served under `/viewer/`. To preview the
viewer locally, run `task build:web` and serve `web/` with any static file server.

## Structure

| Path | Contents |
| --- | --- |
| `hugo.yaml` | Site config: title, `baseURL`, Hextra theme, nav menu. |
| `content/_index.md` | The marketing homepage. |
| `content/docs/` | The docs section (getting started, DSL reference, diagrams, editors, viewer). |
| `static/CNAME` | The custom domain `emod.saphix.app`. |
| `static/images/` | Homepage artwork. |
| `assets/css/` | Homepage layout overrides. |
| `go.mod` / `go.sum` | Pin the Hextra theme as a Hugo module. |

## Deploy

`.github/workflows/pages.yaml` builds the web viewer, copies it into `site/static/viewer/`, builds
`./site`, and deploys to GitHub Pages on every push to `main`. It uses
`actions/configure-pages` → `actions/deploy-pages`, so the site source must be set to
**GitHub Actions** in the repository's Pages settings.

### Custom domain (one-time)

`emod.saphix.app` is a subdomain, so it needs a CNAME record, plus the domain configured in
GitHub:

1. At your DNS provider for `saphix.app`, add a CNAME record:
   - **Host:** `emod`
   - **Value:** `hpcsc.github.io`
2. In the repo **Settings → Pages**, set **Custom domain** to `emod.saphix.app`.
   GitHub verifies the CNAME and serves the site at that domain (HTTPS via a Let's Encrypt
   certificate, issued automatically).

Until the custom domain is configured, the site is available at
`https://hpcsc.github.io/emod/`.
