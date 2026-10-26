#!/usr/bin/env python3
# scripts/import-postgres.py
"""Convert this app's pg_dump custom archive into a new SQLite file."""

import argparse
import datetime
import os
from pathlib import Path
import re
import sqlite3
import subprocess


def decode_copy(value):
    if value == r"\N":
        return None
    escapes = {"b": "\b", "f": "\f", "n": "\n", "r": "\r", "t": "\t", "v": "\v"}

    def unescape(match):
        text = match.group(1)
        if text.startswith("x") and len(text) > 1:
            return chr(int(text[1:], 16))
        if text[0] in "01234567":
            return chr(int(text, 8))
        return escapes.get(text, text)

    return re.sub(r"\\(x[0-9a-fA-F]{1,2}|[0-7]{1,3}|.)", unescape, value)


def read_dump(dump):
    text = subprocess.run(
        ["pg_restore", "--data-only", "--no-owner", "--no-privileges", "-f", "-", str(dump)],
        capture_output=True, text=True, check=True,
    ).stdout
    tables = {}
    current = None
    for line in text.splitlines():
        if current is not None:
            if line == r"\.":
                current = None
                continue
            values = [decode_copy(v) for v in line.split("\t")]
            if len(values) != len(columns):
                raise ValueError(f"wrong column count in {current}")
            tables[current][1].append(values)
            continue
        match = re.fullmatch(r"COPY public\.(\w+) \(([^)]+)\) FROM stdin;", line)
        if match:
            current = match[1]
            columns = match[2].split(", ")
            if current in tables:
                raise ValueError(f"duplicate COPY for {current}")
            tables[current] = (columns, [])
    if current is not None:
        raise ValueError("unterminated COPY")
    if set(tables) != {"links", "github_node_id_aliases", "schema_migrations"}:
        raise ValueError(f"unexpected dump tables: {sorted(tables)}")
    sequence = re.search(r"setval\('public\.links_id_seq', (\d+), (true|false)\)", text)
    if not sequence:
        raise ValueError("links sequence is missing")
    return tables, int(sequence[1]) if sequence[2] == "true" else int(sequence[1]) - 1


def normalize(column, value):
    if value is None:
        return None
    if column in {"id", "link_id", "github_database_id", "fail_count"}:
        return int(value)
    if column.endswith("_at") or column == "verify_after":
        stamp = datetime.datetime.fromisoformat(value)
        if stamp.tzinfo is None:
            raise ValueError(f"timestamp has no timezone: {column}")
        return stamp.astimezone(datetime.timezone.utc).strftime("%Y-%m-%d %H:%M:%S.%f")
    return value


def convert(dump, destination):
    tables, sequence = read_dump(dump)
    destination.parent.mkdir(parents=True, exist_ok=True, mode=0o700)
    # Never overwrite an existing database, including a previous import.
    fd = os.open(destination, os.O_CREAT | os.O_EXCL | os.O_WRONLY, 0o600)
    os.close(fd)
    try:
        with sqlite3.connect(destination) as db:
            db.execute("PRAGMA foreign_keys=ON")
            root = Path(__file__).resolve().parents[1]
            migrations = sorted((root / "core/datastore/migrations").glob("*.sql"))
            if not migrations:
                raise ValueError("SQLite migrations are missing")
            db.execute("CREATE TABLE schema_migrations (name TEXT PRIMARY KEY, applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP)")
            for migration in migrations:
                db.executescript(migration.read_text())
                db.execute("INSERT INTO schema_migrations(name) VALUES (?)", (migration.name,))
            for table in ["links", "github_node_id_aliases"]:
                columns, rows = tables[table]
                actual = [r[1] for r in db.execute(f"PRAGMA table_info({table})")]
                if columns != actual:
                    raise ValueError(f"dump columns do not match SQLite schema: {table}")
                rows = [tuple(normalize(c, v) for c, v in zip(columns, row)) for row in rows]
                placeholders = ",".join("?" for _ in columns)
                db.executemany(f"INSERT INTO {table} VALUES ({placeholders})", rows)
                restored = db.execute(f"SELECT * FROM {table}").fetchall()
                if sorted(restored) != sorted(rows):
                    raise ValueError(f"data comparison failed: {table}")
                print(f"{table}: {len(rows)} rows, all fields verified")
            highest = db.execute("SELECT COALESCE(MAX(id), 0) FROM links").fetchone()[0]
            sequence = max(sequence, highest)
            db.execute("DELETE FROM sqlite_sequence WHERE name='links'")
            db.execute("INSERT INTO sqlite_sequence(name, seq) VALUES ('links', ?)", (sequence,))
            if db.execute("PRAGMA foreign_key_check").fetchall():
                raise ValueError("foreign key check failed")
            if db.execute("PRAGMA integrity_check").fetchone() != ("ok",):
                raise ValueError("integrity check failed")
            db.commit()
            print(f"integrity: ok; next link ID: {sequence + 1}")
    except BaseException:
        destination.unlink()
        raise


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("dump", type=Path)
    parser.add_argument("destination", type=Path)
    args = parser.parse_args()
    convert(args.dump, args.destination)


if __name__ == "__main__":
    main()
