S99AB's theme and stylesheet package is pinned to the commit in `s99ab.json`.
The npm registry does not publish this package. Shipping its package archive
keeps production builds independent of a sibling checkout or a GitHub SSH key.

The archive was made from an untouched `git archive` of that commit, using
`pnpm pack` in the extracted source directory. It contains the package's own
source and license. `just check-generated` verifies its checksum, version, and
package declaration. Update the archive and provenance together when changing
the shared dependency; do not edit files inside the archive.
