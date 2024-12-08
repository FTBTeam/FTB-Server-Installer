#!/bin/bash

# Define the version
version=$GITHUB_REF_NAME

# Strip the 'v' prefix if present
if [[ $version == v* ]]; then
    version="${version#v}"
fi

# Read the changelog
if [[ ! -f "changelog.md" ]]; then
    echo "changelog.md not found"
    exit 1
fi
changelog=$(<changelog.md)

# Initialize foundBins array
declare -A foundBins=()

# Walk through the releases directory
while read -r file; do
    filename=$(basename "$file")
    if [[ $filename =~ ^ftb-server-([a-zA-Z0-9]+)-([a-zA-Z0-9]+)(\.exe)?$ ]]; then
        bin_name="${BASH_REMATCH[1]}-${BASH_REMATCH[2]}"
        bin_url="https://cdn.feed-the-beast.com/bin/server-installer/v${version}/ftb-server-${bin_name}${BASH_REMATCH[3]}"
        foundBins["$bin_name"]="$bin_url"
        echo "$bin_name"
    fi
done < <(find release -type f)  # Use process substitution instead of a pipe

# Create JSON body
assets_json="[]"
for name in "${!foundBins[@]}"; do
    asset_json=$(jq -n --arg name "$name" --arg url "${foundBins[$name]}" '{name: $name, url: $url}')
    assets_json=$(jq ". + [$asset_json]" <<<"$assets_json")
done

post_body=$(jq -n \
    --arg name "$version" \
    --arg description "$changelog" \
    --argjson assets "$assets_json" \
    '{name: $name, description: $description, assets: $assets}')

# Write JSON to file
echo "$post_body" > ./post-body.json
