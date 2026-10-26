// i18n/config.cue
defaultLocale: "en"
locales: ["en", "pl"]
targets: ["web"]
roots: ["locales"]
output: "generated"
outputs: {web: "../web/i18n/generated"}
namespaces: {
 app: {targets: ["web"]}
 header: {targets: ["web"]}
 home: {targets: ["web"]}
 help: {targets: ["web"]}
 settings: {targets: ["web"]}
 docs: {targets: ["web"]}
}
scan: ["../web/app"]
dynamicKeys: ["home.errors.", "settings.theme.", "home.qr.controls.ec.", "home.qr.controls.ecHelper.", "docs.short.title", "docs.inspect.title", "docs.resolve.title", "docs.create.title", "docs.limits.title", "docs.status.", "docs.copy", "docs.copied"]
