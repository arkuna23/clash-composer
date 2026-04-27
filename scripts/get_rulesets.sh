#!/usr/bin/env bash

download() {
    local name="$1"
    local url="https://cdn.jsdelivr.net/gh/Loyalsoldier/clash-rules@release/${name}.txt"
    local path="rulesets/default/${name}.yaml"

    printf "Downloading \e[1m$name\e[0m from $url\n"

    curl -o $path "$url"
    echo "$name $path" >> "rulesets/default/record.txt"
}

mkdir -p "rulesets/default"
rm -f "rulesets/default/record.txt"

download 'reject'
download 'icloud'
download 'apple'
download 'google'
download 'proxy'
download 'direct'
download 'private'
download 'gfw'
download 'tld-not-cn'
download 'telegramcidr'
download 'cncidr'
download 'lancidr'
download 'applications'
