#!/usr/bin/env python3
# scripts/check-generated.py
"""Check generated config, contract outputs, the shared theme package, and deploy manifest."""
import hashlib
import json
import re
from pathlib import Path
import shlex
import shutil
import subprocess
import tarfile
import tempfile

ROOT = Path(__file__).resolve().parents[1]


def run(*args, cwd=ROOT):
    subprocess.run(args, cwd=cwd, check=True)


def check_config():
    cue = shlex.split(subprocess.check_output(['go', 'tool', '-n', 'cue'], cwd=ROOT, text=True))
    with tempfile.TemporaryDirectory(prefix='qr2git-generated-') as directory:
        target = Path(directory) / 'schema'
        target.mkdir()
        shutil.copy2(ROOT / 'schema/config.cue', target / 'config.cue')
        run(*cue, 'exp', 'gengotypes', './schema', cwd=directory)
        generated = target / 'cue_types_schema_gen.go'
        run('gofmt', '-s', '-w', str(generated))
        if generated.read_bytes() != (ROOT / 'common/config/cue_types_gen.go').read_bytes():
            raise ValueError('config types are stale; run just gen-types')
        for filename in ['config.json', 'config.json.dev', 'config.json.deploy']:
            config = Path(directory) / 'config.json'
            shutil.copy2(ROOT / filename, config)
            run(*cue, 'vet', '-d', '#Config', str(ROOT / 'schema/config.cue'), str(config))


def check_contract():
    run('python3', 'scripts/shared-contracts.py', '--check')
    redocly = str(ROOT / 'contracts/node_modules/.bin/redocly')
    config = str(ROOT / 'contracts/backend/redocly.yaml')
    run(redocly, 'lint', '--config', config, 'backend@v1')
    with tempfile.TemporaryDirectory(prefix='qr2git-contract-') as directory:
        bundle = Path(directory) / 'openapi.yaml'
        run(redocly, 'bundle', '--config', config, 'backend@v1',
            '--component-renaming-conflicts-severity=error', '-o', str(bundle))
        if bundle.read_bytes() != (ROOT / 'contracts/backend/dist/openapi.yaml').read_bytes():
            raise ValueError('contract bundle is stale; run just contracts')

        server = Path(directory) / 'api.gen.go'
        oapi = shlex.split(subprocess.check_output(['go', 'tool', '-n', 'oapi-codegen'], cwd=ROOT, text=True))
        generator = Path(directory) / 'oapi-codegen.yaml'
        generator.write_text((ROOT / 'api/v1/oapi-codegen.yaml').read_text()
                             .replace('output: api/v1/gen/api.gen.go', f'output: {server}'))
        run(*oapi, '-config', str(generator), str(bundle))
        if server.read_bytes() != (ROOT / 'api/v1/gen/api.gen.go').read_bytes():
            raise ValueError('Go server boundary is stale; run just generate-backend')

        client = Path(directory) / 'client'
        script = (
            "import('@hey-api/openapi-ts').then(m => m.createClient({"
            f"input: {json.dumps(str(bundle))}, output: {json.dumps(str(client))}, logs: {{ level: 'silent' }},"
            "plugins: ['@hey-api/client-fetch', '@pinia/colada']}))"
        )
        run('node', '-e', script, cwd=ROOT / 'web')
        tracked = ROOT / 'web/app/generated/api'
        want = sorted(p.relative_to(tracked) for p in tracked.rglob('*') if p.is_file())
        got = sorted(p.relative_to(client) for p in client.rglob('*') if p.is_file())
        if want != got or any((tracked / f).read_bytes() != (client / f).read_bytes() for f in want):
            raise ValueError('TypeScript client is stale; run just generate-frontend')


def check_theme():
    provenance = json.loads((ROOT / 'web/vendor/s99ab.json').read_text())
    archive = ROOT / 'web/vendor' / provenance['archive']
    if hashlib.sha256(archive.read_bytes()).hexdigest() != provenance['sha256']:
        raise ValueError('shared theme archive checksum does not match its provenance')
    with tarfile.open(archive) as bundle:
        package = json.load(bundle.extractfile('package/package.json'))
    if package['name'] != provenance['package'] or package['version'] != provenance['version']:
        raise ValueError('shared theme archive contains a different package')
    app = json.loads((ROOT / 'web/package.json').read_text())
    if app['dependencies'][provenance['package']] != 'file:vendor/' + provenance['archive']:
        raise ValueError('frontend theme dependency does not match its provenance')


def check_deploy():
    deploy = json.loads((ROOT / 'deploy.json').read_text())
    config = json.loads((ROOT / 'config.json.deploy').read_text())
    port = int(config['server']['listen_addr'].rsplit(':', 1)[1])
    if deploy['check']['port'] != port or deploy['check']['path'] != '/ready':
        raise ValueError('deployment check must use the configured listener and /ready')
    if deploy['env']['CONFIG_PATH'] != 'config.json.deploy':
        raise ValueError('deployment must load config.json.deploy')
    if 'go build' not in '\n'.join(deploy['build']):
        raise ValueError('deployment does not build the backend')


if __name__ == '__main__':
    if not re.fullmatch(r"\d+\.\d+\.\d+", (ROOT / "VERSION").read_text().strip()):
        raise ValueError("VERSION must contain MAJOR.MINOR.PATCH")
    check_config()
    check_contract()
    check_theme()
    check_deploy()
    print('Config, contract bundle, Go boundary, TypeScript client, shared fragments, theme provenance and deploy settings are current')
