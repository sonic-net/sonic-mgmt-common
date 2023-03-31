#!/usr/bin/env python3
################################################################################
#                                                                              #
#  Copyright 2023 Broadcom. The term Broadcom refers to Broadcom Inc. and/or   #
#  its subsidiaries.                                                           #
#                                                                              #
#  Licensed under the Apache License, Version 2.0 (the "License");             #
#  you may not use this file except in compliance with the License.            #
#  You may obtain a copy of the License at                                     #
#                                                                              #
#     http://www.apache.org/licenses/LICENSE-2.0                               #
#                                                                              #
#  Unless required by applicable law or agreed to in writing, software         #
#  distributed under the License is distributed on an "AS IS" BASIS,           #
#  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.    #
#  See the License for the specific language governing permissions and         #
#  limitations under the License.                                              #
#                                                                              #
################################################################################

import json
import os
import redis
import sys
from argparse import ArgumentParser


def main():
    p = ArgumentParser(
        description="""Creates a database config json file by copying
        contents from a source file. Adds/removes the unix socket path in the INSTANCES sections
        by reading the correct value from redis config (redis-cli -p PORT config get unixsocket).
        """)
    p.add_argument(
        "-s", "--source", dest="srcfile", metavar="SRCFILE",
        default=os.path.join(os.path.dirname(__file__), "database_config.json"),
        help="""Source database config json file.
        Defaults to {sonic-mgmt-common}/tools/test/database_config.json""")
    p.add_argument(
        "-o", "--outfile",
        help="Output database config json file. Defaults to SRCFILE itself (overwrites)")

    opt = p.parse_args()

    with open(opt.srcfile) as f:
        db_config = json.load(f)

    fix_unix_socket_path(opt, db_config)

    if not opt.outfile:
        opt.outfile = opt.srcfile
    elif os.path.isdir(opt.outfile):
        opt.outfile = os.path.join(opt.outfile, os.path.basename(opt.srcfile))
    elif not os.path.exists(os.path.dirname(opt.outfile)):
        os.makedirs(os.path.dirname(opt.outfile))

    with open(opt.outfile, "w") as f:
        s = json.dumps(db_config, indent=4)
        f.write(s)


def fix_unix_socket_path(opt, db_config):
    if not "INSTANCES" in db_config:
        return
    for name, inst in db_config["INSTANCES"].items():
        if "port" in inst:
            fix_instance(name, inst)


def fix_instance(inst_name, inst_config):
    r = redis.Redis(
        host=inst_config.get("hostname", "127.0.0.1"),
        port=inst_config.get("port"),
        decode_responses=True,
    )
    try:
        redis_config = r.config_get("unixsocket")
    except:
        print(f"Could not fix instance '{inst_name}': {sys.exc_info()[1]}")
        return False
    finally:
        r.close()

    if "unixsocket" in redis_config:
        inst_config["unix_socket_path"] = redis_config["unixsocket"]
    elif "unix_socket_path" in inst_config:
        inst_config.pop("unix_socket_path")
    return True


if __name__ == "__main__":
    main()
