#!/usr/bin/env python3

import json
import os
import re
import redis
from argparse import ArgumentParser, Namespace as Obj

redis_clients = {}

ApplDB = 0
CountersDB = 2
ConfigDB = 4
StateDB = 6

overwrite = False


def main():
    p = ArgumentParser(
        description="""Populates test data in essential tables like PORT,
        DEVICE_METADATA, SWITCH_TABLE, USER_TABLE etc.
        """,
    )
    p.add_argument("-p", "--port", help="Redis port number (default 6379)",
                   type=int, default=6379)
    p.add_argument("--platconf", help="""Path to a platform config file
                   for reading PORT settings (default './platform.json')""",
                   default=guess_platconf_file())
    p.add_argument("--overwrite", help="Overwrite existing entry fields",
                   action="store_true")
    p.add_argument("--flushdb", help="Flush db entries first",
                   action="store_true")

    args = p.parse_args()

    with open(args.platconf, "r") as f:
        platconf = json.load(f)

    if args.overwrite:
        global overwrite
        overwrite = args.overwrite

    try:
        for i in (ApplDB, CountersDB, ConfigDB, StateDB):
            redis_clients[i] = redis.Redis(
                host="localhost", port=args.port, db=i,
                decode_responses=True,
            )

        if args.flushdb:
            for n, db in redis_clients.items():
                print(f"[{n}] flushdb")
                db.flushdb()

        for name, cfg in platconf.items():
            create_confdb_entry(name, cfg)
            create_appldb_entry(name, cfg)
            create_intf_counter(name, cfg)
            create_lldp_entry(name, cfg)

        init_device_metadata()
        init_feature_flags()
        init_user_db()
        set_config_db_inited()

    finally:
        for db in redis_clients.values():
            db.close()


def set_config_db_inited():
    d = redis_clients[ConfigDB]
    if not d.exists("CONFIG_DB_INITIALIZED"):
        d.set("CONFIG_DB_INITIALIZED", "1")


def init_device_metadata():
    db_hmset(ConfigDB, "DEVICE_METADATA|localhost", {
        "hwsku":    "generic",
        "hostname": "sonic",
        "platform": "generic",
        "mac":      "00:01:02:03:04:05",
    })


def init_feature_flags():
    db_hmset(ConfigDB, "HARDWARE|ACCESS_LIST", {
        "COUNTER_MODE": "per-rule",
        "LOOKUP_MODE":  "optimized",
    })
    db_hmset(ApplDB, "SWITCH_TABLE:switch", {
        "subinterface_supported": "true",
        "stp_supported":          "true",
        "drop_monitor_supported": "true",
        "vlan_mapping_supported": "true",
    })
    db_hmset(StateDB, "TUNNEL_FEATURE_TABLE|vxlan", {"state": "enable"})
    db_hmset(StateDB, "TUNNEL_CAPABILITY_TABLE|vxlan", {"state": "enable"})
    db_hmset(StateDB, "STP_TABLE|GLOBAL", {"max_stp_inst": "254"})


def init_user_db():
    user = os.getenv("USER")
    db_hmset(StateDB, f"USER_TABLE|{user}", {"roles@": "admin"})


def create_confdb_entry(name, cfg):
    key = f"PORT|{name}"
    values = {
        "admin_status": "up",
        "mtu": 9100,
    }
    if "index" in cfg:
        values["index"] = at(cfg["index"], 0)
    if "alias_at_lanes" in cfg:
        values["alias"] = at(cfg["alias_at_lanes"], 0)
    if "lanes" in cfg:
        values["lanes"] = cfg["lanes"]
    if "default_brkout_mode" in cfg:
        values["speed"] = brkout_mode_to_speed(cfg["default_brkout_mode"])
    db_hmset(ConfigDB, key, values)


def create_appldb_entry(name, cfg):
    key = f"PORT_TABLE:{name}"
    values = {
        "admin_status": "up",
        "oper_status": "up",
        "oper_status_change_uptime": 10000,
        "mtu": 9100,
    }
    if "index" in cfg:
        values["index"] = at(cfg["index"], 0)
    if "lanes" in cfg:
        values["lanes"] = cfg["lanes"]
    if "default_brkout_mode" in cfg:
        speed = brkout_mode_to_speed(cfg["default_brkout_mode"])
        values["speed"] = speed
        values["oper_speed"] = speed
    db_hmset(ApplDB, key, values)


def create_intf_counter(name, cfg):
    oid = f"oid:0x{0x1000000000000+ifid(name):x}"
    db_hmset(CountersDB, "COUNTERS_PORT_NAME_MAP", {name: oid})
    db_hmset(CountersDB, f"COUNTERS:{oid}", {
        "SAI_PORT_STAT_IF_IN_OCTETS":           0,
        "SAI_PORT_STAT_IF_IN_UCAST_PKTS":       0,
        "SAI_PORT_STAT_IF_IN_NON_UCAST_PKTS":   0,
        "SAI_PORT_STAT_IF_IN_BROADCAST_PKTS":   0,
        "SAI_PORT_STAT_IF_IN_MULTICAST_PKTS":   0,
        "SAI_PORT_STAT_IF_IN_ERRORS":           0,
        "SAI_PORT_STAT_IF_IN_DISCARDS":         0,
        "SAI_PORT_STAT_IF_OUT_OCTETS":          0,
        "SAI_PORT_STAT_IF_OUT_UCAST_PKTS":      0,
        "SAI_PORT_STAT_IF_OUT_NON_UCAST_PKTS":  0,
        "SAI_PORT_STAT_IF_OUT_BROADCAST_PKTS":  0,
        "SAI_PORT_STAT_IF_OUT_MULTICAST_PKTS":  0,
        "SAI_PORT_STAT_IF_OUT_ERRORS":          0,
        "SAI_PORT_STAT_IF_OUT_DISCARDS":        0,
    })


# _lldp_config holds remaining LLDP_ENTRY_TABLE data to be created
_lldp_config = [
    Obj(switch=1, ports=[0, 1, 2], i=0),
    Obj(switch=2, ports=[8, 12], i=0),
]


def create_lldp_entry(name, cfg):
    if len(_lldp_config) == 0:
        return
    r = _lldp_config[0]
    p = r.ports.pop(0)
    r.i += 1
    if len(r.ports) == 0:
        _lldp_config.pop(0)

    db_hmset(ApplDB, f"LLDP_ENTRY_TABLE:{name}", {
        "lldp_rem_index":               r.switch,
        "lldp_rem_chassis_id":          f"00:aa:00:00:00:{r.switch:02x}",
        "lldp_rem_chassis_id_subtype":  "4",
        "lldp_rem_man_addr":            f"172.17.2.{r.switch},fd00::{r.switch}",
        "lldp_rem_port_id":             f"Ethernet{p}",
        "lldp_rem_port_desc":           f"Port 1/{r.i}",
        "lldp_rem_port_id_subtype":     "7",
        "lldp_rem_sys_cap_enabled":     "28 00",
        "lldp_rem_sys_cap_supported":   "28 00",
        "lldp_rem_sys_desc":            f"SONiC simulator {r.switch}; Hwsku: generic",
        "lldp_rem_sys_name":            f"sonic{r.switch}",
        "lldp_rem_time_mark":           "5000",
    })


def ifid(name):
    if name.startswith("Ethernet"):
        return int(name[8:])
    return None


def at(v, index):
    toks = v.split(",")
    if index < len(toks):
        return toks[index].strip()
    return ""


def brkout_mode_to_speed(mode):
    speed = 0
    m = re.search(r"(\d+)x(\d+)([MG])", mode)
    try:
        speed = int(m.group(2))
        if m.group(3) == "G":
            speed = speed * 1000
    except:
        print(f"[ERROR] invalid breakout mode: {mode}")
    return speed


def guess_platconf_file():
    return os.path.join(os.path.dirname(__file__), "platform.json")


def db_delete(index, pattern):
    db = redis_clients[index]
    keys = db.keys(pattern)
    if keys:
        [print(f"[{index}] DEL {k}") for k in sorted(keys)]
        db.delete(*keys)


def db_hmset(index, key, value):
    if not overwrite:
        current = redis_clients[index].hgetall(key)
        [value.pop(k, None) for k in current]
    if not value:
        return  # no new values
    toks = [f"{k} {repr(v)}" for k, v in value.items()]
    print(f"[{index}] HMSET {repr(key)} {' '.join(toks)}")
    redis_clients[index].hmset(key, value)


if __name__ == "__main__":
    main()
