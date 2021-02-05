#!/bin/sh
set -e -x

main() {
  local tmpcmdsfile=workspace/tmpcmnds
  if [ -f $tmpcmdsfile ]; then
    while read cmd; do
    $cmd
    done < $tmpcmdsfile
  fi    
}

main