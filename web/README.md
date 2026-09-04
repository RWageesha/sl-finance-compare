# Asset drop-in guide

Everything in this folder is served as-is at the site root (Nuxt convention —
`public/banks/hnb.svg` becomes `/banks/hnb.svg`). Name files exactly as
below and I can wire each one in with a one-line path change, no other code
changes needed.

## Bank logos → `public/banks/`

Filename must match the bank's slug exactly (these are the same slugs
already used in the URLs, e.g. `/banks/hnb`) — lowercase, no spaces:

| File name       | Bank                             |
|------------------|-----------------------------------|
| `hnb.svg`         | Hatton National Bank              |
| `combank.svg`      | Commercial Bank of Ceylon         |
| `boc.svg`          | Bank of Ceylon                    |
| `sampath.svg`      | Sampath Bank                      |
| `ndb.svg`          | National Development Bank         |
| `seylan.svg`       | Seylan Bank                       |
| `dfcc.svg`         | DFCC Bank                         |
| `peoples.svg`      | People's Bank                     |
| `nsb.svg`          | National Savings Bank             |
| `panasia.svg`      | Pan Asia Banking Corporation      |
| `union.svg`        | Union Bank of Colombo             |
| `amana.svg`        | Amana Bank                        |

SVG preferred (scales cleanly, small, works on light/dark backgrounds).
PNG is fine too — just keep the same base name (e.g. `hnb.png`) and tell me
which banks are PNG vs SVG. Only sending some of the 12 is fine; the rest
keep the generic icon until you have their logo.

## Site favicon → `public/`

| File name       | Replaces                                    |
|------------------|----------------------------------------------|
| `favicon.ico`     | The current file — still Nuxt's default, never customized for FindRate |
| `favicon.svg`      | Optional — modern browsers prefer this over the .ico when both exist |

## Social share preview → `public/`

| File name       | Purpose                                                        |
|------------------|------------------------------------------------------------------|
| `og-image.png`     | Shown when a page link is shared on social/chat apps. 1200×630px, doesn't exist yet |

## Hero / illustration images → `public/hero/`

Only needed if you want to replace the current CSS-only skyline on the
homepage with a real image:

| File name       | Purpose                                    |
|------------------|-----------------------------------------------|
| `skyline.jpg` (or `.png`/`.svg`) | Homepage hero background |

Anything else you send — just tell me what it's for and I'll pick a
matching name and update this file.
