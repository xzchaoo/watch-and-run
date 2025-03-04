#!/usr/bin/env bash
set -e

pwd

go build -o /tmp/no-sigterm-app ./cmds/no-sigterm-app

cd `dirname $0`/../..

rm -rf output/

mkdir -p output/

go build -o output/watch-and-run.exe ./cmds/watch-and-run

md5sum output/watch-and-run.exe

date

#/tmp/no-sigterm-app
