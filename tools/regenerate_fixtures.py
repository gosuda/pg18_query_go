#!/usr/bin/env python3
import argparse
import ast
import gzip
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile

ROOT = Path(__file__).resolve().parents[1]
REFERENCE = "github.com/pganalyze/pg_query_go/v6@v6.2.3-0.20260902212127-e6a9b9881a8b"
REFERENCE_SUM = "h1:m5ghqLoahohP8fyHAe9V2YZnLDXlco24KRjuu2jjbDY="


def records(path):
    with gzip.open(path, "rt", encoding="utf-8") as source:
        for line in source:
            yield json.loads(line)


def canonical_archive(path, values):
    with path.open("wb") as output:
        with gzip.GzipFile(filename="", mode="wb", fileobj=output, mtime=0) as archive:
            for value in values:
                archive.write((json.dumps(value, sort_keys=True, separators=(",", ":")) + "\n").encode())


def corrected_records(original, corrected):
    corrections = 0
    for before, after in zip(records(original), records(corrected), strict=True):
        raw_json = before.pop("pl_json", None)
        fixed_json = after.pop("pl_json", None)
        if before != after:
            raise ValueError(f"Serializer patch changed non-JSON fields: {before['file']}:{before['line']}")
        if raw_json != fixed_json:
            try:
                json.loads(raw_json)
            except json.JSONDecodeError:
                pass
            else:
                raise ValueError("Serializer patch changed valid upstream JSON")
            json.loads(fixed_json)
            after["pl_json_upstream"] = raw_json
            corrections += 1
        if fixed_json is not None:
            after["pl_json"] = fixed_json
        yield after
    print(f"Corrected malformed promise-variable JSON in {corrections} fixtures")


def main():
    parser = argparse.ArgumentParser(description="Regenerate independent PostgreSQL reference fixtures outside the target module")
    parser.add_argument("--check", action="store_true", help="compare fixture contents without writing repository files")
    args = parser.parse_args()
    environment = {**os.environ, "CGO_ENABLED": "1", "GOWORK": "off"}
    with tempfile.TemporaryDirectory(prefix="pg18-oracle-") as directory:
        temporary = Path(directory)
        download = subprocess.run(["go", "mod", "download", "-json", REFERENCE], cwd=temporary,
                                  env=environment, check=True, capture_output=True, text=True)
        module = json.loads(download.stdout)
        if module["Sum"] != REFERENCE_SUM:
            raise ValueError("Reference module checksum mismatch")
        reference = temporary / "reference"
        shutil.copytree(module["Dir"], reference)

        def run_oracle(name, *arguments):
            subprocess.run(["go", "run", str(ROOT / "tools" / "oracle" / name), *map(str, arguments)],
                           cwd=reference, env=environment, check=True)

        inputs = [ast.literal_eval(line.strip().removesuffix(","))
                  for line in (ROOT / "testdata" / "libpg_query" / "deparse_tests.c").read_text().splitlines()
                  if line.lstrip().startswith('"')]
        input_path = temporary / "deparse-inputs.json"
        input_path.write_text(json.dumps(inputs))
        deparse_raw = temporary / "deparse-raw.gz"
        run_oracle("deparse.go", input_path, deparse_raw)
        corpus_raw = temporary / "corpus-raw.gz"
        run_oracle("corpus.go", ROOT / "testdata" / "postgresql-18.4", corpus_raw)

        serializer = reference / "parser" / "pg_query_json_plpgsql.c"
        original_source = serializer.read_text()
        marker = "case PLPGSQL_DTYPE_VAR:"
        if original_source.count(marker) != 2:
            raise ValueError("Reference promise-serializer patch no longer applies")
        serializer.chmod(serializer.stat().st_mode | 0o200)
        serializer.write_text(original_source.replace(marker, marker + "\n\t\tcase PLPGSQL_DTYPE_PROMISE:"))
        corpus_fixed = temporary / "corpus-fixed.gz"
        run_oracle("corpus.go", ROOT / "testdata" / "postgresql-18.4", corpus_fixed)

        outputs = {
            "libpg_query_deparse.json.gz": records(deparse_raw),
            "postgresql-18.4.json.gz": corrected_records(corpus_raw, corpus_fixed),
        }
        for name, values in outputs.items():
            generated = temporary / name
            canonical_archive(generated, values)
            destination = ROOT / "testdata" / name
            if args.check:
                for expected, actual in zip(records(destination), records(generated), strict=True):
                    if expected != actual:
                        raise ValueError(f"Fixture drift in {name}")
                print(f"Verified {name}")
            else:
                shutil.copyfile(generated, destination)
                print(f"Wrote {name}: sha256={hashlib.sha256(generated.read_bytes()).hexdigest()}")


if __name__ == "__main__":
    main()
