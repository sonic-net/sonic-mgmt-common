"""Run the unchanged Go assertions using only declared native/runtime inputs."""

import json
import os
from pathlib import Path, PurePosixPath
import posixpath
import shutil
import socket
import subprocess
import sys
import tarfile
import tempfile
import time


def unpack(archive, destination):
    def relocatable(member, root):
        if member.issym() and member.linkname.startswith("/"):
            target = posixpath.relpath(member.linkname.lstrip("/"), str(PurePosixPath(member.name).parent))
            member = member.replace(linkname=target)
        return tarfile.data_filter(member, root)
    with tarfile.open(archive) as tar:
        tar.extractall(destination, filter=relocatable)


def main():
    binary, libraries, payload, schema, dbconfig, fixture, initializer, platform = [
        Path(p).absolute() for p in sys.argv[1:9]
    ]
    initialize = sys.argv[9] == "1"
    # AF_UNIX limits the endpoint length; Bazel's default test directory can
    # already exceed that limit. The private directory is removed in all paths.
    with tempfile.TemporaryDirectory(prefix="mgmt-go-", dir="/tmp") as temporary:
        root = Path(temporary)
        runtime, work = root / "runtime", root / "work"
        runtime.mkdir()
        work.mkdir()
        unpack(libraries, runtime)
        unpack(payload, runtime)
        (work / "testdata").mkdir()
        # Match cvl/Makefile: copy JSON fixtures, then generated YIN only.
        # Raw YANG alongside YIN lets libyang resolve imports to unprocessed
        # models, bypassing CVL's error prefixes and config-only filtering.
        for source in fixture.parent.glob("*.json"):
            shutil.copyfile(source, work / "testdata" / source.name)
        shutil.copytree(schema, work / "testdata/schema", dirs_exist_ok=True)
        loader = runtime / "usr/lib/x86_64-linux-gnu/ld-linux-x86-64.so.2"
        if not loader.exists():
            loader = runtime / "lib/x86_64-linux-gnu/ld-linux-x86-64.so.2"
        paths = sorted({str(p.parent) for p in runtime.rglob("lib*.so*") if p.is_file()})
        libpath = ":".join(paths)
        environment = dict(os.environ)
        environment.pop("GO_TEST_RUN_FROM_BAZEL", None)
        environment.update({
            "SONIC_TEST_REDIS_MANAGED": "1",
            # The outer py_test owns reporting. rules_go's optional XML wrapper
            # re-execs the ELF directly, losing this declared runtime loader.
            "GO_TEST_WRAP": "0",
            "DB_CONFIG_PATH": str(root / "database_config.json"),
            "CVL_SCHEMA_PATH": str(work / "testdata/schema"),
            "CVL_CFG_FILE": str(runtime / "usr/sbin/cvl_cfg.json"),
            "YANG_MODELS_PATH": str(runtime / "usr/models/yang"),
        })
        # Each suite has independent databases and process lifetime.
        with socket.socket() as listener:
            listener.bind(("127.0.0.1", 0))
            port = listener.getsockname()[1]
        unix_socket = str(root / "redis.sock")
        config = json.loads(dbconfig.read_text())
        for instance in config["INSTANCES"].values():
            instance.update(hostname="127.0.0.1", port=port, unix_socket_path=unix_socket)
        for name, dbid in [("LOGLEVEL_DB", 3), ("ERROR_DB", 8), ("EVENT_DB", 9)]:
            config["DATABASES"].setdefault(name, {"id": dbid, "separator": "|", "instance": "redis"})
        (root / "database_config.json").write_text(json.dumps(config))
        redis = runtime / "usr/bin/redis-server"
        if not loader.is_file() or not redis.is_file():
            raise ValueError("Declared AMD64 test runtime is incomplete")
        with (root / "redis.log").open("w+") as log:
            service = subprocess.Popen([
                str(loader), "--library-path", libpath, str(redis),
                "--bind", "127.0.0.1", "--port", str(port),
                "--unixsocket", unix_socket, "--save", "", "--appendonly", "no",
                "--databases", "32", "--dir", str(root),
            ], stdout=log, stderr=subprocess.STDOUT)
            try:
                deadline = time.monotonic() + 15
                while service.poll() is None and time.monotonic() < deadline:
                    try:
                        with socket.create_connection(("127.0.0.1", port), timeout=0.2) as client:
                            client.sendall(b"*1\r\n$4\r\nPING\r\n")
                            if client.recv(64).strip() == b"+PONG":
                                break
                    except OSError:
                        pass
                    time.sleep(0.05)
                else:
                    log.seek(0)
                    raise RuntimeError("Private Redis failed to start:\n" + log.read())
                if initialize:
                    # The same owner fixture used by azure-pipelines.yml.
                    subprocess.run([str(initializer), "--port", str(port),
                                    "--platconf", str(platform), "--overwrite"],
                                   cwd=work, env=environment, check=True)
                subprocess.run([str(loader), "--library-path", libpath, str(binary), "-test.v"],
                               cwd=work, env=environment, check=True)
            finally:
                service.terminate()
                try:
                    service.wait(timeout=5)
                except subprocess.TimeoutExpired:
                    service.kill()
                    service.wait()


if __name__ == "__main__":
    main()
