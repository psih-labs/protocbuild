#!/usr/bin/env bash
main() {
  local tmpcmdsfile=tmpcmnds
  if [ -f $tmpcmdsfile ]; then
    while read cmd; do
    $cmd
    done < $tmpcmdsfile
  fi    
}

main