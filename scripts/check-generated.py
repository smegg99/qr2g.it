#!/usr/bin/env python3
# scripts/check-generated.py
"""Check generated config types, the shared theme package, and deploy manifest."""
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


def check_swagger():
    swag = shlex.split(subprocess.check_output(['go', 'tool', '-n', 'swag'], cwd=ROOT, text=True))
    with tempfile.TemporaryDirectory(prefix='qr2git-swagger-') as directory:
        target = Path(directory) / 'docs'
        run(*swag, 'init', '-g', 'main.go', '--dir', str(ROOT), '--output', str(target), '--parseInternal', '--quiet')
        for name in ['docs.go', 'swagger.json', 'swagger.yaml']:
            if (target / name).read_bytes() != (ROOT / 'docs' / name).read_bytes():
                raise ValueError('Swagger output is stale; run just gen-swagger')


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
    check_swagger()
    check_theme()
    check_deploy()
    print('Config and Swagger generation, shared theme provenance, and deploy settings are current')
