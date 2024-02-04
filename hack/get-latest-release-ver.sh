#!/bin/bash

prefix="$(echo $1 | cut -d '.' -f 1-2)"

while read -r release; do
    release_prefix=$(echo "$release" | cut -d '.' -f 1-2)
    if [[ "$release_prefix" == "$prefix" ]]; then
        echo "$release"
        exit 0
    fi
done <<< "$(curl https://api.github.com/repos/kubernetes-sigs/cloud-provider-azure/releases | jq -r .[].tag_name)"

echo "latest"
