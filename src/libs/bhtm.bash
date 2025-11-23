#!/usr/bin/env bash

headers='{"foo":"bar"}'

headers() {
  case "$1" in
    "get")
      if [[ "$2" == "" ]]; then
        printf "no header provided"
      fi
      ;;
    *)
      printf "$headers" | jq
      ;;
  esac
}

linesToHTML() {
  local stdin=$(< /dev/stdin) 
  echo "${stdin}" | sed 's|^|<p>|g' | sed 's|$|</p>|g'
}
