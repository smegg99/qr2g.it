#!/usr/bin/env python3
"""Download a consistent snapshot of the production SQLite database over SSH."""

import argparse
from contextlib import closing
import os
from pathlib import Path
import shlex
import sqlite3
import subprocess
import tempfile


REMOTE_BACKUP = """
from contextlib import closing
import json
from pathlib import Path
import shutil
import sqlite3
import sys
import tempfile

app = Path('/opt/qr2git/app')
config = json.loads((app / 'config.json.deploy').read_text())
source = (app / config['db']['path']).resolve()
with tempfile.TemporaryDirectory(prefix='qr2git-download-') as directory:
    snapshot = Path(directory) / 'snapshot.sqlite'
    with closing(sqlite3.connect(source.as_uri() + '?mode=ro', uri=True)) as db:
        with closing(sqlite3.connect(snapshot)) as backup:
            db.backup(backup)
    with snapshot.open('rb') as stream:
        shutil.copyfileobj(stream, sys.stdout.buffer)
"""


def download(destination, host):
    destination.parent.mkdir(parents=True, exist_ok=True, mode=0o700)
    for suffix in ('-wal', '-shm'):
        if Path(str(destination) + suffix).exists():
            raise ValueError(f'close the database preview before replacing {destination}')
    fd, temporary = tempfile.mkstemp(prefix=f'.{destination.name}.', dir=destination.parent)
    temporary = Path(temporary)
    try:
        with os.fdopen(fd, 'wb') as stream:
            subprocess.run(
                ['ssh', '-o', 'BatchMode=yes', '--', host,
                 'sudo -n python3 -c ' + shlex.quote(REMOTE_BACKUP)],
                stdout=stream, check=True,
            )
        with closing(sqlite3.connect(temporary)) as db:
            db.execute('PRAGMA journal_mode=DELETE')
            if db.execute('PRAGMA integrity_check').fetchall() != [('ok',)]:
                raise ValueError('downloaded database failed its integrity check')
            if db.execute('PRAGMA foreign_key_check').fetchall():
                raise ValueError('downloaded database has broken foreign keys')
            links = db.execute('SELECT COUNT(*) FROM links').fetchone()[0]
            aliases = db.execute('SELECT COUNT(*) FROM github_node_id_aliases').fetchone()[0]
        os.replace(temporary, destination)
    finally:
        temporary.unlink(missing_ok=True)
    print(f'Saved full database to {destination}: {links} links, {aliases} aliases; integrity ok')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('destination', type=Path)
    parser.add_argument('--host', default='ovh')
    args = parser.parse_args()
    try:
        download(args.destination, args.host)
    except (OSError, sqlite3.Error, ValueError, subprocess.CalledProcessError) as error:
        parser.exit(1, f'{error}\n')


if __name__ == '__main__':
    main()
