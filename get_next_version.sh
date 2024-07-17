#!/bin/bash

# Fetch the latest tags from the remote
git fetch --tags

# Get the latest tag and split it into components
latest_tag=$(git describe --tags $(git rev-list --tags --max-count=1) 2>/dev/null)
if [ -z "$latest_tag" ]; then
  latest_tag="1.0.0"
fi
IFS='.' read -r -a version_parts <<< "$latest_tag"

# Default to incrementing the patch version
new_major=${version_parts[0]}
new_minor=${version_parts[1]}
new_patch=$((version_parts[2] + 1))

# Check commit messages for version bump keywords
if git log -1 | grep -q 'BREAKING CHANGE'; then
    new_major=$((new_major + 1))
    new_minor=0
    new_patch=0
elif git log -1 | grep -q 'feat'; then
    new_minor=$((new_minor + 1))
    new_patch=0
fi

# Output the new version
echo "$new_major.$new_minor.$new_patch"