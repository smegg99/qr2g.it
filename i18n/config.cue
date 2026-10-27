// i18n/config.cue
defaultLocale: "en"
locales: ["en", "pl"]
targets: ["web", "backend"]
roots: ["locales"]
output: "generated"
outputs: {web: "../web/i18n/generated", backend: "../common/messages"}
go: {package: "messages"}
namespaces: {
 app: {targets: ["web"]}
 header: {targets: ["web"]}
 home: {targets: ["web"]}
 help: {targets: ["web"]}
 settings: {targets: ["web"]}
 docs: {targets: ["web"]}
 errors: {targets: ["web"]}
 generator: {targets: ["web"]}
 details: {targets: ["web"]}
 design: {targets: ["web"]}
 exporter: {targets: ["web"]}
 history: {targets: ["web"]}
 stats: {targets: ["web"]}
 explore: {targets: ["web"]}
 contact: {targets: ["web"]}
 result: {targets: ["web"]}
 missing: {targets: ["web"]}
 cli: {targets: ["backend"]}
 page: {targets: ["backend"]}
}
scan: ["../web/app", "../cmd", "../api"]
dynamicKeys: ["errors.", "settings.theme.", "docs.", "docs.status.", "docs.copy", "docs.copied", "generator.outcome.", "result.steps.", "generator.prefill.", "design.fields.", "design.invalid.", "design.ecHelper.", "contact.categories.", "missing."]
