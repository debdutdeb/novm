{
  releases: [
    .[] | select(.draft | not) | {
      tag: .tag_name,
      url: .html_url,
      published_at: .published_at,
      assets: [
        .assets[] | {
          os: (if (.name | test("^novm-[^-]+-[^-]+$")) then (.name | split("-")[1]) else null end),
          arch: (if (.name | test("^novm-[^-]+-[^-]+$")) then (.name | split("-")[2]) else null end),
          download_url: .browser_download_url
        }
      ]
    }
  ]
}
