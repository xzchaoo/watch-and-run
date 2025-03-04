#!/usr/bin/env bash
set -e

cd `dirname $0`/../..

go run ./cmds/watch-and-run ./cmds/watch-and-run/war.toml
